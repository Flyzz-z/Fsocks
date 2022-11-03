package listener

import (
	"io"
	"net"
	"strconv"
)

const maxAddrLen = 2 + 255 + 2

const (
	cmdConnect      = 1
	cmdBind         = 2
	cmdUDPAssociate = 3
)

const (
	atypIPv4   = 1
	atypDomain = 3
	atypIPv6   = 4
)

type Addr = []byte
type Port = uint16

func readAddr(rw io.ReadWriter) (string, error) {
	buf := make([]byte, maxAddrLen)
	_, err := io.ReadFull(rw, buf[:1])
	if err != nil {
		return "", err
	}
	n := 0
	var ip string
	switch buf[0] {
	case atypIPv4:
		n = 4
		_, err := io.ReadFull(rw, buf[:n])
		if err != nil {
			return "", err
		}
		ip = net.IP(buf[:n]).String()
	case atypDomain:
		_, err := io.ReadFull(rw, buf[:1])
		if err != nil {
			return "", err
		}
		n = int(buf[0])
		_, err = io.ReadFull(rw, buf[:n])
		if err != nil {
			return "", err
		}
		ip = string(buf[:n])
	case atypIPv6:
		n = 16
		_, err := io.ReadFull(rw, buf[:n])
		if err != nil {
			return "", err
		}
		ip = net.IP(buf[:n]).String()
	}

	_, err = io.ReadFull(rw, buf[n:n+2])
	if err != nil {
		return "", err
	}

	port := strconv.Itoa((int(buf[n]))<<8 + int(buf[n+1]))
	return ip + port, nil
}

func handleSocks5(rw io.ReadWriter) (Addr, error) {
	buf := make([]byte, maxAddrLen)
	_, err := io.ReadFull(rw, buf[:2])
	if err != nil {
		return nil, err
	}

	nMethod := buf[1]
	_, err = io.ReadFull(rw, buf[2:2+nMethod])
	if err != nil {
		return nil, err
	}
	_, err = rw.Write([]byte{5, 0})
	if err != nil {
		return nil, err
	}

	//handle req
	_, err = io.ReadFull(rw, buf[:3])
	if err != nil {
		return nil, err
	}
	cmd := buf[1]
	// get []byte addr
	_, err = io.ReadFull(rw, buf[:1])
	if err != nil {
		return nil, err
	}
	atyp := buf[0]
	var n int
	switch atyp {
	case atypIPv4:
		_, err = io.ReadFull(rw, buf[1:7])
		if err != nil {
			return nil, err
		}
		n = 7
	case atypIPv6:
		_, err := io.ReadFull(rw, buf[1:19])
		if err != nil {
			return nil, err
		}
		n = 19
	case atypDomain:
		_, err = io.ReadFull(rw, buf[1:2])
		if err != nil {
			return nil, err
		}
		_, err = io.ReadFull(rw, buf[2:buf[1]+2])
		if err != nil {
			return nil, err
		}
		n = int(buf[1] + 2)
	}
	server := buf[:n]
	if err != nil {
		return nil, err
	}

	//do reply
	switch cmd {
	case cmdConnect:
		_, err := rw.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		if err != nil {
			return nil, err
		}
	case cmdUDPAssociate:

	case cmdBind:

	}
	return server, nil
}
