package handler

import (
	"Fsocks/config"
	"Fsocks/shadow"
	"errors"
	"io"
	"log"
	"net"
	"sync"
)

type ProxyType = string

const (
	SS ProxyType = "ss"
)

type ProxyHandler interface {
	Handle(task TcpTask) error
}

type TcpTask struct {
	Lc       net.Conn
	server   string
	port     int
	Addr     []byte
	cipher   string
	password string
}

var (
	ssHandler SSHandler
)

func Distribute(task TcpTask) error {
	pc := config.GetProxyConf()
	if pc == nil {
		return errors.New("no proxy config")
	}
	task.server = pc.Server
	task.port = pc.Port
	task.cipher = pc.Cipher
	task.password = pc.Password
	var handler ProxyHandler
	switch pc.PType {
	case SS:
		handler = ssHandler
	}
	go handler.Handle(task)
	return nil
}

func relay(c net.Conn, cc shadow.CipherStream) {
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		_, err := io.Copy(c, cc)
		if err != nil {
			log.Println("relay error1", err)
			c.Close()
			cc.Close()
		}
		wg.Done()
	}()
	_, err := io.Copy(cc, c)
	if err != nil {
		log.Println("relay error", err)
		c.Close()
		cc.Close()
	}
	wg.Wait()
}
