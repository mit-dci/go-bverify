package wire

import (
	"bytes"
	"encoding/binary"
)

type MessageType byte

const (
	// [C > S]     MessageTypeCreateLog is used to request the server for
	//             creating a new log
	MessageTypeCreateLog MessageType = 0x01

	// [C > S]     MessageTypeAppendLog is used to request the server to
	//             append a new statement to an existing log
	MessageTypeAppendLog MessageType = 0x02

	// [C > S > C] MessageTypeRequestProof is sent both from server to client
	//             and client to server to acknowledge the receipt of a
	//             message when necessary
	MessageTypeAck MessageType = 0x03

	// [S > C]     MessageTypeError is sent to the server in response to
	//             a message that the server failed to process
	MessageTypeError MessageType = 0x04

	// [S > C]     MessageTypeProofUpdate is sent to the client when a new
	//             commitment has been made including a delta to the proof
	//             update. Requirement is that the client has subscribed to
	//             such updates
	MessageTypeProofUpdate MessageType = 0x05

	// [C > S]     MessageTypeRequestProof is sent to the server to request a
	//             full proof for a set of logs
	MessageTypeRequestProof MessageType = 0x06

	// [S > C]     MessageTypeProof is sent to the client in response to the
	//             MessageTypeRequestProof containing the proof
	MessageTypeProof MessageType = 0x07

	// [C > S]     MessageTypeSubscribeProofUpdates is sent to the server to
	//			  subscribe to live delta proof updates with every new
	//             commitment
	MessageTypeSubscribeProofUpdates MessageType = 0x08

	// [C > S]     MessageTypeUnsubscribeProofUpdates is sent to the server to
	//             subscribe to live delta proof updates with every new commitment
	MessageTypeUnsubscribeProofUpdates MessageType = 0x09

	// [C > S] 	   MessageTypeRequestDeltaProof  is sent to the server to request a
	//             delta proof since the last commitment for a set of logs
	MessageTypeRequestDeltaProof MessageType = 0x0A

	// [S > C]     MessageTypeDeltaProof is sent to the client in response to the
	//             MessageTypeRequestDeltaProof containing the proof
	MessageTypeDeltaProof MessageType = 0x0B

	// [C > S] 	  MessageTypeRequestCommitmentHistory  is sent to the server to
	//             request the commitment details of a set of historic commitments
	MessageTypeRequestCommitmentHistory MessageType = 0x0C

	// [S > C]     MessageTypeCommitmentHistory is sent to the client in response to the
	//             MessageTypeRequestCommitmentHistory containing the commitment details
	MessageTypeCommitmentHistory MessageType = 0x0D

	// [C > S] 	  MessageTypeRequestCommitmentDetails  is sent to the server to
	//             request the commitment details of a single commitment
	MessageTypeRequestCommitmentDetails MessageType = 0x0E

	// [S > C]     MessageTypeCommitmentDetails is sent to the client in response to the
	//             MessageTypeRequestCommitmentDetails containing the commitment details
	MessageTypeCommitmentDetails MessageType = 0x0F
)

// RequestProofMessage is the payload to a MessageTypeRequestProof
type RequestProofMessage struct {
	// The LogIDs to request the proof for
	LogIDs [][32]byte
}

// Bytes serializes a RequestProofMessage to a byte slice
func (m *RequestProofMessage) Bytes() []byte {
	var buf bytes.Buffer
	for _, logID := range m.LogIDs {
		buf.Write(logID[:])
	}
	return buf.Bytes()
}

// NewRequestProofMessage is a convenience function for creating a new
// RequestProofMessage from an array of logIDs
func NewRequestProofMessage(logIDs [][32]byte) *RequestProofMessage {
	msg := new(RequestProofMessage)
	msg.LogIDs = logIDs
	return msg
}

// NewRequestProofMessageFromBytes deserializes a byte slice into a
// RequestProofMessage
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

// RequestCommitmentDetailsMessage is the payload to a
// MessageTypeRequestCommitmentDetails
type RequestCommitmentDetailsMessage struct {
	Commitment [32]byte
}

// Bytes serializes a RequestCommitmentDetailsMessage to a byte slice
func (m *RequestCommitmentDetailsMessage) Bytes() []byte {
	return m.Commitment[:]
}

// NewRequestCommitmentDetailsMessage is a convenience function for creating a new
// RequestCommitmentDetailsMessage from a single commitment
func NewRequestCommitmentDetailsMessage(commitment [32]byte) *RequestCommitmentDetailsMessage {
	msg := new(RequestCommitmentDetailsMessage)
	msg.Commitment = commitment
	return msg
}

// NewRequestCommitmentDetailsMessageFromBytes deserializes a byte slice into a
// RequestCommitmentDetailsMessage
func NewRequestCommitmentDetailsMessageFromBytes(b []byte) (*RequestCommitmentDetailsMessage, error) {
	msg := new(RequestCommitmentDetailsMessage)
	copy(msg.Commitment[:], b[:])
	return msg, nil
}

// RequestCommitmentHistoryMessage is the payload to a
// MessageTypeRequestCommitmentHistory
type RequestCommitmentHistoryMessage struct {
	SinceCommitment [32]byte
}

// Bytes serializes a RequestCommitmentHistoryMessage to a byte slice
func (m *RequestCommitmentHistoryMessage) Bytes() []byte {
	return m.SinceCommitment[:]
}

// NewRequestCommitmentHistoryMessage is a convenience function for creating a new
// RequestCommitmentHistoryMessage from the sinceCommitment
func NewRequestCommitmentHistoryMessage(sinceCommitment [32]byte) *RequestCommitmentHistoryMessage {
	msg := new(RequestCommitmentHistoryMessage)
	msg.SinceCommitment = sinceCommitment
	return msg
}

// NewRequestCommitmentHistoryMessageFromBytes deserializes a byte slice into a
// RequestCommitmentHistoryMessage
func NewRequestCommitmentHistoryMessageFromBytes(b []byte) (*RequestCommitmentHistoryMessage, error) {
	msg := new(RequestCommitmentHistoryMessage)
	copy(msg.SinceCommitment[:], b[:])
	return msg, nil
}

// CommitmentDetailsMessage is the payload to a MessageTypeCommitmentDetails
type CommitmentDetailsMessage struct {
	// The commitment details
	Commitment *Commitment
}

// Bytes serializes a RequestCommitmentHistoryMessage to a byte slice
func (m *CommitmentDetailsMessage) Bytes() []byte {
	return m.Commitment.Bytes()
}

// NewCommitmentDetailsMessage is a convenience function for creating a new
// CommitmentDetailsMessage from a single commitment object
func NewCommitmentDetailsMessage(c *Commitment) *CommitmentDetailsMessage {
	msg := new(CommitmentDetailsMessage)
	msg.Commitment = c
	return msg
}

// NewCommitmentDetailsMessageFromBytes deserializes a byte slice into a
// CommitmentDetailsMessage
func NewCommitmentDetailsMessageFromBytes(b []byte) (*CommitmentDetailsMessage, error) {
	msg := new(CommitmentDetailsMessage)
	msg.Commitment = CommitmentFromBytes(b)
	return msg, nil
}

// CommitmentHistoryMessage is the payload to a MessageTypeCommitmentHistory
type CommitmentHistoryMessage struct {
	// List of the commitment details
	Commitments []*Commitment
}

// Bytes serializes a CommitmentHistoryMessage to a byte slice
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

// NewCommitmentHistoryMessage is a convenience function for creating a new
// CommitmentHistoryMessage from an array of commitment objects
func NewCommitmentHistoryMessage(c []*Commitment) *CommitmentHistoryMessage {
	msg := new(CommitmentHistoryMessage)
	msg.Commitments = c
	return msg
}

// NewCommitmentHistoryMessageFromBytes deserializes a byte slice into a
// CommitmentHistoryMessage
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
