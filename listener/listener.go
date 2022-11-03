package listener

import (
	"Fsocks/handler"
	"net"
	"strconv"
)

func Listen(port int) error {

	p := strconv.Itoa(port)
	l, err := net.Listen("tcp", "127.0.0.1:"+p)
	if err != nil {
		return err
	}
	for {
		c, err := l.Accept()
		if err != nil {
			continue
		}
		go func() {
			addr, err := handleSocks5(c)
			if err != nil {
				c.Close()
			}
			task := handler.TcpTask{Lc: c, Addr: addr}
			err = handler.Distribute(task)
			if err != nil {
				c.Close()
			}
		}()
	}
	return nil
}
