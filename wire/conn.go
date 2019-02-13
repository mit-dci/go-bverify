package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

type Connection struct {
	conn      net.Conn
	writeLock sync.Mutex
}

func NewConnection(c net.Conn) *Connection {
	return &Connection{conn: c, writeLock: sync.Mutex{}}
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) ReadNextMessage() (MessageType, []byte, error) {
	bType := make([]byte, 1)
	bLen := make([]byte, 2)
	n, err := io.ReadFull(c.conn, bType)
	if err != nil {
		return 0x00, nil, err
	}
	if n != 1 {
		return 0x00, nil, fmt.Errorf("Wrong length read for type : expected 1, got %d", n)
	}

	n, err = io.ReadFull(c.conn, bLen)
	if err != nil {
		return 0x00, nil, err
	}
	if n != 2 {
		return 0x00, nil, fmt.Errorf("Wrong length read for length : expected 2, got %d", n)
	}

	l := binary.BigEndian.Uint16(bLen)
	bMsg := make([]byte, l)
	if l > 0 {
		n, err = io.ReadFull(c.conn, bMsg)
		if err != nil {
			return 0x00, nil, err
		}
		if n != int(l) {
			return 0x00, nil, fmt.Errorf("Wrong length read for body : expected %d, got %d", l, n)
		}

	}
	//fmt.Printf("< [%x%x%x]\n", bType, bLen, bMsg)

	return MessageType(bType[0]), bMsg, nil
}

func (c *Connection) WriteMessage(t MessageType, m []byte) error {
	c.writeLock.Lock()
	bMsg := make([]byte, 3)
	bMsg[0] = byte(t)
	binary.BigEndian.PutUint16(bMsg[1:], uint16(len(m)))
	bMsg = append(bMsg, m...)
	//fmt.Printf("> [%x]\n", bMsg)
	n, err := c.conn.Write(bMsg)
	if err != nil {
		c.writeLock.Unlock()
		return err
	}
	if n != len(bMsg) {
		c.writeLock.Unlock()
		return fmt.Errorf("Not all bytes written. Expected %d, got %d", len(bMsg), n)
	}
	c.writeLock.Unlock()
	return nil
}
