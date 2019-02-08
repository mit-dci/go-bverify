package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Connection struct {
	conn net.Conn
}

func NewConnection(c net.Conn) *Connection {
	return &Connection{conn: c}
}

func (c *Connection) Close() {
	c.conn.Close()
}

func (c *Connection) ReadNextMessage() (MessageType, []byte, error) {
	bType := make([]byte, 1)
	bLen := make([]byte, 2)
	n, err := io.ReadFull(c.conn, bType)
	if n != 1 {
		return 0x00, nil, fmt.Errorf("Wrong length read : expected 1, got %d", n)
	}
	if err != nil {
		return 0x00, nil, err
	}

	n, err = io.ReadFull(c.conn, bLen)
	if n != 2 {
		return 0x00, nil, fmt.Errorf("Wrong length read : expected 2, got %d", n)
	}
	if err != nil {
		return 0x00, nil, err
	}

	l := binary.BigEndian.Uint16(bLen)
	bMsg := make([]byte, l)
	n, err = io.ReadFull(c.conn, bMsg)
	if n != int(l) {
		return 0x00, nil, fmt.Errorf("Wrong length read : expected %d, got %d", l, n)
	}
	if err != nil {
		return 0x00, nil, err
	}

	return MessageType(bType[0]), bMsg, nil
}

func (c *Connection) WriteMessage(t MessageType, m []byte) error {
	bMsg := make([]byte, 3)
	bMsg[0] = byte(t)
	binary.BigEndian.PutUint16(bMsg[1:], uint16(len(m)))
	bMsg := append(bMsg, m)
	n, err := c.conn.Write(bMsg)
	if n != len(bMsg) {
		return fmt.Errorf("Not all bytes written. Expected %d, got %d", len(bMsg), n)
	}
	if err != nil {
		return err
	}
	return nil
}
