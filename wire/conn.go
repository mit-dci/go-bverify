package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/mit-dci/go-bverify/logging"
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

// ReadNextMessage reads a type, length and then payload from the transport and
// returns the message type and payload to the caller.
func (c *Connection) ReadNextMessage() (MessageType, []byte, error) {
	bType := make([]byte, 1)
	bLen := make([]byte, 4)

	//logging.Debugf("[%p] Reading type", c)

	n, err := io.ReadFull(c.conn, bType)
	if err != nil {
		return 0x00, nil, err
	}
	if n != 1 {
		return 0x00, nil, fmt.Errorf("Wrong length read for type : expected 1, got %d", n)
	}
	//logging.Debugf("[%p] Read Type %x", c, bType)

	//logging.Debugf("[%p] Reading len", c)
	n, err = io.ReadFull(c.conn, bLen)
	if err != nil {
		return 0x00, nil, err
	}
	if n != 4 {
		return 0x00, nil, fmt.Errorf("Wrong length read for length : expected 4, got %d", n)
	}

	l := binary.BigEndian.Uint32(bLen)

	//logging.Debugf("[%p] Read Len %d", c, l)

	bMsg := make([]byte, l)
	if l > 0 {
		//logging.Debugf("[%p] Reading msg", c)

		n, err = io.ReadFull(c.conn, bMsg)
		if err != nil {
			return 0x00, nil, err
		}
		if n != int(l) {
			return 0x00, nil, fmt.Errorf("Wrong length read for body : expected %d, got %d", l, n)
		}

	}

	return MessageType(bType[0]), bMsg, nil
}

// WriteMessage writes a message to the transport of the given type t and payload m
// it uses a  Mutex to prevent two threads writing at the same time.
func (c *Connection) WriteMessage(t MessageType, m []byte) error {

	bMsg := make([]byte, 5+len(m))
	bMsg[0] = byte(t)
	binary.BigEndian.PutUint32(bMsg[1:], uint32(len(m)))
	if len(m) > 0 {
		copy(bMsg[5:], m)
	}
	c.writeLock.Lock()
	n, err := c.conn.Write(bMsg)
	if err != nil {
		logging.Errorf("[%p] Error writing message: %s", c, err.Error())
		c.writeLock.Unlock()
		return err
	}
	if n != len(bMsg) {
		err = fmt.Errorf("Not all bytes written. Expected %d, got %d", len(m), n)
		logging.Errorf("[%p] Error writing message: %s", c, err.Error())
		c.writeLock.Unlock()
		return err
	}
	c.writeLock.Unlock()
	return nil
}
