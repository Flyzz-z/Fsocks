package handler

import (
	"Fsocks/shadow"
	"log"
	"net"
	"strconv"
)

type SSHandler struct {
}

func (h SSHandler) Handle(task TcpTask) error {
	remote := net.JoinHostPort(task.server, strconv.Itoa(task.port))
	rc, err := net.Dial("tcp", remote)
	if err != nil {
		return err
	}

	crc := shadow.NewCipherStream(rc, task.password, task.cipher)
	err = crc.InitWriter(task.Addr)
	if err != nil {
		log.Println("initwriter fail")
		return err
	}
	err = crc.InitReader()
	if err != nil {
		log.Println("initReader fail", err)
		return err
	}
	relay(task.Lc, crc)
	return nil
}
