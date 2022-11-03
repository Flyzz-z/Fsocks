package shadow

import "C"
import (
	"bytes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"io"
	"net"
)

const (
	MAXPAYLOADSIZE int = 16383
)

type CipherType = string

const (
	AEAD_CHACHA20_POLY1305 CipherType = "chacha20-ietf-poly1305"
	AEAD_AES_256_GCM       CipherType = "aes-256-gcm"
	AEAD_AES_128_GCM       CipherType = "aes-128-gcm"
)

type Cipher struct {
	Aead  cipher.AEAD
	Nonce []byte
}

type CipherStream struct {
	C        net.Conn
	Cr       Cipher
	Cw       Cipher
	genKey   func(salt []byte) (key []byte, err error)
	genAead  func(key []byte) (aead cipher.AEAD, err error)
	saltSize int
}

func NewCipherStream(c net.Conn, password string, cipherType string) (cs CipherStream) {
	var genKey func([]byte) ([]byte, error)
	var genAead func([]byte) (cipher.AEAD, error)
	switch cipherType {
	case AEAD_CHACHA20_POLY1305:
		genKey = func(salt []byte) (key []byte, err error) {
			secret := kdf(password, 32)
			h := hkdf.New(sha1.New, secret, salt, []byte("ss-subkey"))
			key = make([]byte, 32)
			if _, err = io.ReadFull(h, key); err != nil {
				return nil, err
			}
			return key, nil
		}

		genAead = func(key []byte) (aead cipher.AEAD, err error) {
			aead, err = chacha20poly1305.New(key)
			return aead, err
		}

		cs.saltSize = 32
	case AEAD_AES_256_GCM:
		//key := sha256.Sum256([]byte(password))
		//block, _ := aes.NewCipher(key[:])
		//aead, _ = cipher.NewGCM(block)
	case AEAD_AES_128_GCM:

	}
	cs.C = c
	cs.genAead = genAead
	cs.genKey = genKey
	return cs
}

func (cs *CipherStream) InitReader() error {
	salt := make([]byte, cs.saltSize)
	if _, err := io.ReadFull(cs.C, salt); err != nil {
		return err
	}
	key, err := cs.genKey(salt)
	if err != nil {
		return err
	}
	cs.Cr.Aead, err = cs.genAead(key)
	if err != nil {
		return err
	}
	cs.Cr.Nonce = make([]byte, cs.Cr.Aead.NonceSize())
	return nil
}

func (cs *CipherStream) InitWriter(addr []byte) error {
	salt := make([]byte, cs.saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	key, err := cs.genKey(salt)
	if err != nil {
		return err
	}
	cs.Cw.Aead, err = cs.genAead(key)
	if err != nil {
		return err
	}
	cs.Cw.Nonce = make([]byte, cs.Cw.Aead.NonceSize())
	_, err = cs.C.Write(salt)
	if err != nil {
		return err
	}
	_, err = cs.Write(addr)
	if err != nil {
		return err
	}
	return nil
}

func (cs CipherStream) ReadFrom(r io.Reader) (n int64, err error) {
	ci := &cs.Cw
	buff := make([]byte, 2+ci.Aead.Overhead()+MAXPAYLOADSIZE+ci.Aead.Overhead())
	for {
		buf := buff
		payload := buf[2+ci.Aead.Overhead() : 2+ci.Aead.Overhead()+MAXPAYLOADSIZE+ci.Aead.Overhead()]
		nr, er := r.Read(payload)
		if nr > 0 {
			n += int64(nr)
			payload = payload[:nr+ci.Aead.Overhead()]
			//log.Println("stream.go ReadFrom ", payload)
			buf := buf[:2+ci.Aead.Overhead()+nr+ci.Aead.Overhead()]
			buf[0], buf[1] = byte(nr>>8), byte(nr)
			ci.Aead.Seal(buf[:0], ci.Nonce, buf[:2], nil)
			increment(ci.Nonce)
			ci.Aead.Seal(payload[:0], ci.Nonce, payload[:nr], nil)
			increment(ci.Nonce)
			_, ew := cs.C.Write(buf)
			//log.Println("stream.go ReadFrom encrypt write", buf)
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return n, err
}

func (cs CipherStream) WriteTo(w io.Writer) (n int64, err error) {
	ci := &cs.Cr
	buff := make([]byte, 2+ci.Aead.Overhead()+MAXPAYLOADSIZE+ci.Aead.Overhead())
	for {
		buf := buff
		nm, er := io.ReadFull(cs.C, buf[:2+ci.Aead.Overhead()])
		if er != nil {
			if nm == 0 && er == io.EOF {
				break
			}
			err = er
			break
		}
		_, er = ci.Aead.Open(buf[:0], ci.Nonce, buf[:2+ci.Aead.Overhead()], nil)
		if er != nil {
			err = er
			break
		}
		increment(ci.Nonce)
		nr := int(buf[0])<<8 + int(buf[1])
		_, er = io.ReadFull(cs.C, buf[:nr+ci.Aead.Overhead()])
		//log.Println("writeto", buf[:nr+ci.Aead.Overhead()])
		if er != nil {
			err = er
			break
		}
		_, er = ci.Aead.Open(buf[:0], ci.Nonce, buf[:nr+ci.Aead.Overhead()], nil)
		increment(ci.Nonce)
		//log.Println("stream.go WriteTo", buf[:nr])
		_, er = w.Write(buf[:nr])
		if er != nil {
			err = er
			break
		}
		n += int64(nr)
	}
	return n, err
}

func (ci CipherStream) Read(p []byte) (n int, err error) {

	return 0, nil
}

func (cs CipherStream) Write(p []byte) (n int, err error) {
	buf := bytes.NewBuffer(p)
	nr, err := cs.ReadFrom(buf)
	n = int(nr)
	return n, err
}

func (cs CipherStream) Close() error {
	return cs.C.Close()
}

func increment(b []byte) {
	for i := range b {
		b[i]++
		if b[i] != 0 {
			return
		}
	}
}

func kdf(password string, keyLen int) []byte {
	var b, prev []byte
	h := md5.New()
	for len(b) < keyLen {
		h.Write(prev)
		h.Write([]byte(password))
		b = h.Sum(b)
		prev = b[len(b)-h.Size():]
		h.Reset()
	}
	return b[:keyLen]
}
