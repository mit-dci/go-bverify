package server

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mit-dci/go-bverify/mpt"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"

	"github.com/mit-dci/go-bverify/wire"
)

type connectedClient struct {
	conn   *wire.Connection
	logIDs [][]byte
}

var clients = []*connectedClient{}

var logIDToPubKey = map[[32]byte][33]byte{}
var logIDLock = sync.RWMutex{}

var logIDIndex = map[[32]byte]uint64{}
var logIDIndexLock = sync.RWMutex{}
var logCounter = float64(0)

var fullmpt *mpt.FullMPT
var lastCommitment = [32]byte{}
var mptLock = sync.Mutex{}

func RunServer() {
	addr, err := net.ResolveTCPAddr("tcp", ":9100")
	if err != nil {
		panic(err)
	}

	fullmpt, _ = mpt.NewFullMPT()

	l, err := net.ListenTCP("tcp", addr)

	ticker := time.NewTicker(time.Millisecond * 500)
	go func() {
		for range ticker.C {
			lps := logCounter * 2
			logCounter = 0
			logIDLock.RLock()
			numLogs := len(logIDToPubKey)
			logIDLock.RUnlock()
			fmt.Printf("\rTracking [%d] logs at [%.2f] appends/sec          ", numLogs, lps)
		}
	}()

	commitTicker := time.NewTicker(time.Second * 5)
	go func() {
		for range commitTicker.C {
			Commit()
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		c := wire.NewConnection(conn)
		cli := &connectedClient{conn: c, logIDs: make([][]byte, 0)}
		clients = append(clients, cli)
		go func(client *connectedClient) {
			for {
				t, m, e := client.conn.ReadNextMessage()
				if e != nil {
					client.conn.Close()
					return
				}

				e = ProcessMessage(cli, t, m)
				if e != nil {
					client.conn.WriteMessage(wire.MessageTypeError, []byte(e.Error()))
					client.conn.Close()
					return
				}

			}
		}(cli)
	}

}

func ProcessMessage(c *connectedClient, t wire.MessageType, m []byte) error {
	if t == wire.MessageTypeCreateLog {
		pm, err := wire.NewSignedCreateLogStatementFromBytes(m)
		if err != nil {
			return err
		}
		return ProcessCreateLog(c, pm)
	}

	if t == wire.MessageTypeAppendLog {
		pm, err := wire.NewSignedLogStatementFromBytes(m)
		if err != nil {
			return err
		}
		return ProcessAppendLog(c, pm)
	}

	return fmt.Errorf("Unrecognized message type received")
}

func ProcessCreateLog(c *connectedClient, scls *wire.SignedCreateLogStatement) error {

	err := scls.VerifySignature()
	if err != nil {
		return err
	}

	hash := fastsha256.Sum256(scls.CreateStatement.Bytes())

	logIDLock.Lock()
	_, ok := logIDToPubKey[hash]
	if ok {
		logIDLock.Unlock()
		return fmt.Errorf("Duplicate log ID created: [%x]", hash)
	}
	logIDToPubKey[hash] = scls.CreateStatement.ControllingKey
	logIDLock.Unlock()

	err = ProcessLogStatement(hash, 0, scls.CreateStatement.InitialStatement)
	if err != nil {
		return err
	}

	c.SubscribeToLog(hash)

	logCounter++
	c.conn.WriteMessage(wire.MessageTypeAck, []byte{})
	return nil
}

func ProcessAppendLog(c *connectedClient, sls *wire.SignedLogStatement) error {
	logIDLock.RLock()
	pk, ok := logIDToPubKey[sls.Statement.LogID]
	logIDLock.RUnlock()

	if !ok {
		return fmt.Errorf("LogID does not exist")
	}

	err := sls.VerifySignature(pk)
	if err != nil {
		return err
	}

	err = ProcessLogStatement(sls.Statement.LogID, sls.Statement.Index, sls.Statement.Statement)
	if err != nil {
		return err
	}

	c.SubscribeToLog(sls.Statement.LogID)

	logCounter++
	c.conn.WriteMessage(wire.MessageTypeAck, []byte{})
	return nil
}

func (c *connectedClient) SubscribeToLog(logID [32]byte) {
	for _, lid := range c.logIDs {
		if bytes.Equal(lid[:], logID[:]) {
			return
		}
	}

	c.logIDs = append(c.logIDs, logID[:])

}

func ProcessLogStatement(logID [32]byte, index uint64, statement []byte) error {
	logIDIndexLock.Lock()
	idx, ok := logIDIndex[logID]
	if !ok && index != uint64(0) {
		return fmt.Errorf("Unexpected log index %d - expected 0", index)
	} else if ok && index != idx+1 {
		return fmt.Errorf("Unexpected log index %d - expected %d", index, idx+1)
	}
	logIDIndex[logID] = index
	logIDIndexLock.Unlock()

	mptLock.Lock()
	fullmpt.Insert(logID[:], statement)
	mptLock.Unlock()

	return nil
}

func Commit() error {
	mptLock.Lock()
	defer mptLock.Unlock()
	commitment := fullmpt.Commitment()
	if bytes.Equal(lastCommitment[:], commitment[:]) {

		return nil
	}
	copy(lastCommitment[:], commitment[:])
	delta, err := mpt.NewDeltaMPT(fullmpt)
	if err != nil {
		return err
	}
	for _, c := range clients {

		if len(c.logIDs) > 0 {
			clientDelta, err := delta.GetUpdatesForKeys(c.logIDs)
			if err != nil {
				return err
			}
			c.conn.WriteMessage(wire.MessageTypeProofUpdate, clientDelta.Bytes())
		}
	}

	fullmpt.Reset()
	return nil
}
