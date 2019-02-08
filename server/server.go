package server

import (
	"net"

	"github.com/mit-dci/go-bverify/wire"
)

func RunServer() {
	addr, err := net.ResolveTCPAddr("tcp", ":9100")
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", addr)

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		c := wire.NewConnection(conn)
		go func(wc *wire.Connection) {
			for {
				t, m, e := wc.ReadNextMessage()
				if e != nil {
					wc.Close()
					return
				}

				e = ProcessMessage(t, m)
				if e != nil {
					wc.Close()
					return
				}

			}
		}(c)
	}

}

func ProcessMessage(t wire.MessageType, m []byte) error {
	return nil
}
