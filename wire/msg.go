package wire

import (
	"bytes"
)

type MessageType byte

const (
	MessageTypeCreateLog               MessageType = 0x01
	MessageTypeAppendLog               MessageType = 0x02
	MessageTypeAck                     MessageType = 0x03
	MessageTypeError                   MessageType = 0x04
	MessageTypeProofUpdate             MessageType = 0x05
	MessageTypeRequestProof            MessageType = 0x06
	MessageTypeProof                   MessageType = 0x07
	MessageTypeSubscribeProofUpdates   MessageType = 0x08
	MessageTypeUnsubscribeProofUpdates MessageType = 0x09
)

type RequestProofMessage struct {
	LogIDs [][32]byte
}

func (m *RequestProofMessage) Bytes() []byte {
	var buf bytes.Buffer
	for _, logID := range m.LogIDs {
		buf.Write(logID[:])
	}
	return buf.Bytes()
}

func NewRequestProofMessage(logIDs [][32]byte) *RequestProofMessage {
	msg := new(RequestProofMessage)
	msg.LogIDs = logIDs
	return msg
}

func NewRequestProofMessageFromBytes(b []byte) (*RequestProofMessage, error) {
	msg := new(RequestProofMessage)
	msg.LogIDs = make([][32]byte, 0)
	buf := bytes.NewBuffer(b)
	for buf.Len() > 0 {
		var logID [32]byte
		copy(logID[:], buf.Next(32))
		msg.LogIDs = append(msg.LogIDs, logID)
	}
	return msg, nil
}
