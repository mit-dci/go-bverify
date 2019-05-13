package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

// Connection is a wrapper around the raw net.Conn and allows to easily read
// and write messages from/to the wire
type Connection struct {
	conn      net.Conn
	writeLock sync.Mutex
}

// NewConnection creates a new Connection with the given net.Conn as underlying
// transport
func NewConnection(c net.Conn) *Connection {
	return &Connection{conn: c, writeLock: sync.Mutex{}}
}

// Close closes the network connection
func (c *Connection) Close() error {
	return c.conn.Close()
}

func (c *Connection) Write(p []byte) (n int, err error) {
	c.writeLock.Lock()
	n, err = c.conn.Write(p)
	c.writeLock.Unlock()
	return
}

func (c *Connection) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	return
}

// ReadNextMessage reads a type, length and then payload from the transport and
// returns the message type and payload to the caller.
func (c *Connection) ReadNextMessage() (MessageType, []byte, error) {
	bType := make([]byte, 1)
	bLen := make([]byte, 4)
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
	if n != 4 {
		return 0x00, nil, fmt.Errorf("Wrong length read for length : expected 4, got %d", n)
	}

	l := binary.BigEndian.Uint32(bLen)
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

// WriteMessagePrefix writes a message prefix to the transport of the given type t and length l
func (c *Connection) writeMessagePrefix(t MessageType, l int) error {

	bMsg := make([]byte, 5)
	bMsg[0] = byte(t)
	binary.BigEndian.PutUint32(bMsg[1:], uint32(l))
	n, err := c.conn.Write(bMsg)
	if err != nil {
		return err
	}
	if n != 5 {
		return fmt.Errorf("Not all bytes written. Expected 5, got %d", n)
	}

	return nil
}

func (c *Connection) WriteMessageToStream(t MessageType, l int, write func(io.Writer) (int, error)) error {
	c.writeLock.Lock()
	err := c.writeMessagePrefix(t, l)
	if err != nil {
		c.writeLock.Unlock()
		return err
	}
	n, err := write(c.conn)
	if err != nil {
		c.writeLock.Unlock()
		return err
	}
	if n != l {
		c.writeLock.Unlock()
		return fmt.Errorf("Not all bytes written. Expected %d, got %d", l, n)
	}
	c.writeLock.Unlock()
	return nil
}

// WriteMessage writes a message to the transport of the given type t and payload m
// it uses a  Mutex to prevent two threads writing at the same time.
func (c *Connection) WriteMessage(t MessageType, m []byte) error {
	c.writeLock.Lock()
	err := c.writeMessagePrefix(t, len(m))
	if err != nil {
		c.writeLock.Unlock()
		return err
	}
	n, err := c.conn.Write(m)
	if err != nil {
		c.writeLock.Unlock()
		return err
	}
	if n != len(m) {
		c.writeLock.Unlock()
		return fmt.Errorf("Not all bytes written. Expected %d, got %d", len(m), n)
	}
	c.writeLock.Unlock()
	return nil
}
