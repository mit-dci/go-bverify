package wire

import (
	"bytes"
	"crypto/rand"
	"net"
	"testing"
)

func TestConnectionSimple(t *testing.T) {
	c, s := net.Pipe()
	client, server := NewConnection(c), NewConnection(s)

	garbage := make([]byte, 256)
	rand.Read(garbage[:])
	go server.WriteMessage(MessageTypeAck, garbage)
	mt, p, err := client.ReadNextMessage()
	if err != nil {
		t.Error(err)
		return
	}
	if mt != MessageTypeAck {
		t.Errorf("Expected to receive Ack, but received %x", byte(mt))
		return
	}
	if !bytes.Equal(garbage, p) {
		t.Errorf("Message sent is not the one received.")
		return
	}
}

func TestConnectionDisconnect(t *testing.T) {
	c, s := net.Pipe()
	client, server := NewConnection(c), NewConnection(s)

	server.Close()
	client.ReadNextMessage()
	_, _, err := client.ReadNextMessage()
	if err == nil {
		t.Error("Expected error reading from closed connection, got none")
		return
	}
}

func TestConnectionSendWrongLength(t *testing.T) {
	c, s := net.Pipe()
	client := NewConnection(c)

	go func() {
		s.Write([]byte{0x01, 0x02})
		s.Close()
	}()
	client.ReadNextMessage()
	_, _, err := client.ReadNextMessage()
	if err == nil {
		t.Error("Expected error reading insufficient data, got none")
		return
	}
}

func TestConnectionSendTooLittleData(t *testing.T) {
	c, s := net.Pipe()
	client := NewConnection(c)

	go func() {
		s.Write([]byte{0x01, 0x00, 0x02, 0x01})
		s.Close()
	}()
	client.ReadNextMessage()
	_, _, err := client.ReadNextMessage()
	if err == nil {
		t.Error("Expected error reading insufficient data, got none")
		return
	}
}

func TestConnectionSendToClosedConnection(t *testing.T) {
	c, s := net.Pipe()
	client, server := NewConnection(c), NewConnection(s)

	server.Close()
	err := client.WriteMessage(MessageTypeAck, []byte{})
	if err == nil {
		t.Error("Expected error writing to closed connection, got none")
		return
	}
}
