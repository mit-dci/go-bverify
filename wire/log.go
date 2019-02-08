package wire

import (
	"bytes"

	"github.com/mit-dci/lit/wire"
)

type SignedCreateLogStatement struct {
	Signature       [64]byte
	CreateStatement *CreateLogStatement
}

type CreateLogStatement struct {
	ControllingKey   [33]byte
	InitialStatement []byte
}

type SignedLogStatement struct {
	Signature [64]byte
	Statement *LogStatement
}

type LogStatement struct {
	LogID     [32]byte
	Index     uint64
	Statement []byte
}

func NewSignedCreateLogStatement(controllingKey [33]byte, initialStatement []byte) *SignedCreateLogStatement {
	ret := new(SignedCreateLogStatement)
	ret.CreateStatement = new(CreateLogStatement)
	ret.CreateStatement.ControllingKey = controllingKey
	ret.CreateStatement.InitialStatement = initialStatement
	return ret
}

func NewSignedLogStatement(index uint64, logID [32]byte, statement []byte) *SignedLogStatement {
	ret := new(SignedLogStatement)
	ret.Statement = new(LogStatement)
	ret.Statement.Index = index
	ret.Statement.LogID = logID
	ret.Statement.Statement = statement
	return ret
}

func (ls *LogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(ls.LogID[:])
	WriteVarInt(&buf, ls.Index)
	WriteVarBytes(&buf, ls.Statement)
	return buf.Bytes()
}

func (sls *SignedLogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(sls.Signature[:])
	buf.Write(sls.Statement.Bytes())
	return buf.Bytes()
}

func (cls *CreateLogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(cls.ControllingKey[:])
	WriteVarBytes(&buf, cls.InitialStatement)
	return buf.Bytes()
}

func (scls *SignedCreateLogStatement) Bytes() []byte {
	var buf bytes.Buffer
	buf.Write(scls.Signature[:])
	buf.Write(scls.CreateStatement.Bytes())
	return buf.Bytes()
}

func NewLogStatementFromBytes(b []byte) *LogStatement {
	buf := bytes.NewBuffer(b)
	ls := new(LogStatement)
	buf.Read(ls.LogID[:])
	idx, _ := wire.ReadVarInt(buf, 0)
	statement, _ := wire.ReadVarBytes(buf, 0, 256, "statement")
	ls.Index = idx
	ls.Statement = statement
	return ls
}

func NewSignedLogStatementFromBytes(b []byte) *SignedLogStatement {
	buf := bytes.NewBuffer(b)
	sls := new(SignedLogStatement)
	buf.Read(sls.Signature[:])
	sls.Statement = NewLogStatementFromBytes(buf.Bytes())
	return sls
}

func NewCreateLogStatementFromBytes(b []byte) *CreateLogStatement {
	buf := bytes.NewBuffer(b)
	cls := new(CreateLogStatement)
	buf.Read(cls.ControllingKey[:])
	statement, _ := wire.ReadVarBytes(buf, 0, 256, "statement")
	cls.InitialStatement = statement
	return cls
}

func NewSignedCreateLogStatementFromBytes(b []byte) *SignedCreateLogStatement {
	buf := bytes.NewBuffer(b)
	scls := new(SignedCreateLogStatement)
	buf.Read(scls.Signature[:])
	scls.CreateStatement = NewCreateLogStatementFromBytes(buf.Bytes())
	return scls
}
