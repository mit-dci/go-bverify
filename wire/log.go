package wire

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/go-bverify/crypto"
)

// SignedCreateLogStatement is a log creation message including a signature
type SignedCreateLogStatement struct {
	Signature       [64]byte
	CreateStatement *CreateLogStatement
}

// CreateLogStatement is an unsigned log creation message
type CreateLogStatement struct {
	ControllingKey   [33]byte
	InitialStatement []byte
}

// SignedLogStatement is a log append message including a signature
type SignedLogStatement struct {
	Signature [64]byte
	Statement *LogStatement
}

// LogStatement is an unsigned log append message
type LogStatement struct {
	LogID     [32]byte
	Index     uint64
	Statement []byte
}

// NewSignedCreateLogStatement is a convenience function for creating a new
// SignedCreateLogStatement without the signature filled in
func NewSignedCreateLogStatement(controllingKey [33]byte, initialStatement []byte) *SignedCreateLogStatement {
	ret := new(SignedCreateLogStatement)
	ret.CreateStatement = new(CreateLogStatement)
	ret.CreateStatement.ControllingKey = controllingKey
	ret.CreateStatement.InitialStatement = initialStatement
	return ret
}

// NewSignedLogStatement is a convenience function for creating a new
// SignedLogStatement without the signature filled in
func NewSignedLogStatement(index uint64, logID [32]byte, statement []byte) *SignedLogStatement {
	ret := new(SignedLogStatement)
	ret.Statement = new(LogStatement)
	ret.Statement.Index = index
	ret.Statement.LogID = logID
	ret.Statement.Statement = statement
	return ret
}

// Bytes serializes a LogStatement to a byte slice
func (ls *LogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(ls.LogID[:])
	WriteVarInt(&buf, ls.Index)
	WriteVarBytes(&buf, ls.Statement)
	return buf.Bytes()
}

// Bytes serializes a SignedLogStatement to a byte slice
func (sls *SignedLogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(sls.Signature[:])
	buf.Write(sls.Statement.Bytes())
	return buf.Bytes()
}

// Bytes serializes a CreateLogStatement to a byte slice
func (cls *CreateLogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(cls.ControllingKey[:])
	WriteVarBytes(&buf, cls.InitialStatement)
	return buf.Bytes()
}

// Bytes serializes a SignedCreateLogStatement to a byte slice
func (scls *SignedCreateLogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(scls.Signature[:])
	buf.Write(scls.CreateStatement.Bytes())
	return buf.Bytes()
}

// NewLogStatementFromBytes deserializes a byte slice into a
// LogStatement
func NewLogStatementFromBytes(b []byte) (*LogStatement, error) {
	buf := bytes.NewBuffer(b)
	ls := new(LogStatement)
	buf.Read(ls.LogID[:])
	idx, err := ReadVarInt(buf)
	if err != nil {
		return nil, err
	}
	statement, err := ReadVarBytes(buf, 256, "statement")
	if err != nil {
		return nil, err
	}
	ls.Index = idx
	ls.Statement = statement
	return ls, nil
}

// NewSignedLogStatementFromBytes deserializes a byte slice into a
// SignedLogStatement
func NewSignedLogStatementFromBytes(b []byte) (*SignedLogStatement, error) {
	buf := bytes.NewBuffer(b)
	sls := new(SignedLogStatement)
	n, err := buf.Read(sls.Signature[:])
	if err != nil {
		return nil, err
	}
	if n < 64 {
		return nil, fmt.Errorf("Unexpected end of buffer")
	}

	sls.Statement, err = NewLogStatementFromBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}
	return sls, nil
}

// NewCreateLogStatementFromBytes deserializes a byte slice into a
// CreateLogStatement
func NewCreateLogStatementFromBytes(b []byte) (*CreateLogStatement, error) {
	buf := bytes.NewBuffer(b)
	cls := new(CreateLogStatement)
	n, err := buf.Read(cls.ControllingKey[:])
	if err != nil {
		return nil, err
	}
	if n < 33 {
		return nil, fmt.Errorf("Unexpected end of buffer")
	}
	statement, err := ReadVarBytes(buf, 256, "statement")
	if err != nil {
		return nil, err
	}
	cls.InitialStatement = statement
	return cls, nil
}

// NewSignedCreateLogStatementFromBytes deserializes a byte slice into a
// SignedCreateLogStatement
func NewSignedCreateLogStatementFromBytes(b []byte) (*SignedCreateLogStatement, error) {
	buf := bytes.NewBuffer(b)
	scls := new(SignedCreateLogStatement)
	n, err := buf.Read(scls.Signature[:])
	if err != nil {
		return nil, err
	}
	if n < 64 {
		return nil, fmt.Errorf("Unexpected end of buffer")
	}
	scls.CreateStatement, err = NewCreateLogStatementFromBytes(buf.Bytes())
	if err != nil {
		return nil, err
	}
	return scls, nil
}

// VerifySignature will verify if the signature in this SignedCreateLogStatement
// is valid
func (scls *SignedCreateLogStatement) VerifySignature() error {
	return crypto.VerifySig(scls.CreateStatement.Bytes(), scls.CreateStatement.ControllingKey, scls.Signature)
}

// VerifySignature will verify if the signature in this SignedLogStatement
// is valid
func (sls *SignedLogStatement) VerifySignature(controllingPubKey [33]byte) error {
	return crypto.VerifySig(sls.Statement.Bytes(), controllingPubKey, sls.Signature)
}
