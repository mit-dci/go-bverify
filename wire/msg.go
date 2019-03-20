package wire

import (
	"bytes"
	"encoding/binary"
)

type MessageType byte

const (
	MessageTypeCreateLog                MessageType = 0x01
	MessageTypeAppendLog                MessageType = 0x02
	MessageTypeAck                      MessageType = 0x03
	MessageTypeError                    MessageType = 0x04
	MessageTypeProofUpdate              MessageType = 0x05
	MessageTypeRequestProof             MessageType = 0x06
	MessageTypeProof                    MessageType = 0x07
	MessageTypeSubscribeProofUpdates    MessageType = 0x08
	MessageTypeUnsubscribeProofUpdates  MessageType = 0x09
	MessageTypeRequestDeltaProof        MessageType = 0x0A
	MessageTypeDeltaProof               MessageType = 0x0B
	MessageTypeRequestCommitmentHistory MessageType = 0x0C
	MessageTypeCommitmentHistory        MessageType = 0x0D
	MessageTypeRequestCommitmentDetails MessageType = 0x0E
	MessageTypeCommitmentDetails        MessageType = 0x0F
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

type RequestCommitmentDetailsMessage struct {
	Commitment [32]byte
}

func (m *RequestCommitmentDetailsMessage) Bytes() []byte {
	return m.Commitment[:]
}

func NewRequestCommitmentDetailsMessage(commitment [32]byte) *RequestCommitmentDetailsMessage {
	msg := new(RequestCommitmentDetailsMessage)
	msg.Commitment = commitment
	return msg
}

func NewRequestCommitmentDetailsMessageFromBytes(b []byte) (*RequestCommitmentDetailsMessage, error) {
	msg := new(RequestCommitmentDetailsMessage)
	copy(msg.Commitment[:], b[:])
	return msg, nil
}

type RequestCommitmentHistoryMessage struct {
	SinceCommitment [32]byte
}

func (m *RequestCommitmentHistoryMessage) Bytes() []byte {
	return m.SinceCommitment[:]
}

func NewRequestCommitmentHistoryMessage(sinceCommitment [32]byte) *RequestCommitmentHistoryMessage {
	msg := new(RequestCommitmentHistoryMessage)
	msg.SinceCommitment = sinceCommitment
	return msg
}

func NewRequestCommitmentHistoryMessageFromBytes(b []byte) (*RequestCommitmentHistoryMessage, error) {
	msg := new(RequestCommitmentHistoryMessage)
	copy(msg.SinceCommitment[:], b[:])
	return msg, nil
}

type CommitmentDetailsMessage struct {
	Commitment *Commitment
}

func (m *CommitmentDetailsMessage) Bytes() []byte {
	return m.Commitment.Bytes()
}

func NewCommitmentDetailsMessage(c *Commitment) *CommitmentDetailsMessage {
	msg := new(CommitmentDetailsMessage)
	msg.Commitment = c
	return msg
}

func NewCommitmentDetailsMessageFromBytes(b []byte) (*CommitmentDetailsMessage, error) {
	msg := new(CommitmentDetailsMessage)
	msg.Commitment = CommitmentFromBytes(b)
	return msg, nil
}

type CommitmentHistoryMessage struct {
	Commitments []*Commitment
}

func (m *CommitmentHistoryMessage) Bytes() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, int32(len(m.Commitments)))
	for _, c := range m.Commitments {
		commitmentBytes := c.Bytes()
		binary.Write(&buf, binary.BigEndian, int32(len(commitmentBytes)))
		buf.Write(commitmentBytes)
	}

	return buf.Bytes()
}

func NewCommitmentHistoryMessage(c []*Commitment) *CommitmentHistoryMessage {
	msg := new(CommitmentHistoryMessage)
	msg.Commitments = c
	return msg
}

func NewCommitmentHistoryMessageFromBytes(b []byte) (*CommitmentHistoryMessage, error) {
	msg := new(CommitmentHistoryMessage)
	buf := bytes.NewBuffer(b)

	numCommitments := int32(0)
	binary.Read(buf, binary.BigEndian, &numCommitments)

	msg.Commitments = make([]*Commitment, numCommitments)
	for i := range msg.Commitments {
		commitmentLength := int32(0)
		binary.Read(buf, binary.BigEndian, &commitmentLength)
		msg.Commitments[i] = CommitmentFromBytes(buf.Next(int(commitmentLength)))
	}
	return msg, nil
}
