package client

import (
	"bytes"
	"fmt"
	"net"

	"github.com/mit-dci/go-bverify/mpt"

	"github.com/mit-dci/go-bverify/crypto/sig64"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"

	"github.com/mit-dci/go-bverify/crypto/btcec"

	"github.com/mit-dci/go-bverify/wire"
)

type Client struct {
	conn          *wire.Connection
	key           *btcec.PrivateKey
	keyBytes      []byte
	ack           chan bool
	proof         chan *mpt.PartialMPT
	pubKey        [33]byte
	OnError       func(error, *Client)
	OnProofUpdate func([]byte, *Client)
}

func NewClientWithConnection(key []byte, c net.Conn) (*Client, error) {

	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)
	var pk [33]byte
	copy(pk[:], pub.SerializeCompressed())

	cli := &Client{conn: wire.NewConnection(c), keyBytes: key, key: priv, pubKey: pk, proof: make(chan *mpt.PartialMPT, 1), ack: make(chan bool, 1)}
	go cli.ReceiveLoop()
	return cli, nil
}

func NewClient(key []byte) (*Client, error) {
	c, err := net.Dial("tcp", "127.0.0.1:9100")
	if err != nil {
		return nil, err
	}

	return NewClientWithConnection(key, c)
}

func (c *Client) UsesKey(key []byte) bool {
	return bytes.Equal(key, c.keyBytes)
}

func (c *Client) ReceiveLoop() {
	for {
		t, p, err := c.conn.ReadNextMessage()
		if err != nil {
			c.conn.Close()
			return
		}

		if t == wire.MessageTypeAck {
			c.ack <- true
			continue
		}

		if t == wire.MessageTypeError {
			if c.OnError != nil {
				go c.OnError(fmt.Errorf("%s", string(p)), c)
			}
			return
		}

		if t == wire.MessageTypeProofUpdate {
			if c.OnProofUpdate != nil {
				go c.OnProofUpdate(p, c)
			}
			continue
		}

		if t == wire.MessageTypeProof {
			mpt, err := mpt.NewPartialMPTFromBytes(p)
			if err != nil {
				c.conn.Close()
				return
			}
			c.proof <- mpt
			continue
		}
	}
}

func (c *Client) StartLog(initialStatement []byte) ([32]byte, error) {

	l := wire.NewSignedCreateLogStatement(c.pubKey, initialStatement)
	hash := fastsha256.Sum256(l.CreateStatement.Bytes())
	sig, err := c.key.Sign(hash[:])

	if err != nil {
		return [32]byte{}, err
	}
	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		return [32]byte{}, err
	}
	l.Signature = csig

	err = c.conn.WriteMessage(wire.MessageTypeCreateLog, l.Bytes())
	if err != nil {
		return [32]byte{}, err
	}

	// Wait for ack
	<-c.ack

	return hash, nil

}

func (c *Client) RequestProof(logIds [][32]byte) (*mpt.PartialMPT, error) {
	msg := wire.NewRequestProofMessage(logIds)
	err := c.conn.WriteMessage(wire.MessageTypeRequestProof, msg.Bytes())
	if err != nil {
		return nil, err
	}

	proof := <-c.proof
	return proof, nil
}

func (c *Client) SubscribeProofUpdates() error {
	err := c.conn.WriteMessage(wire.MessageTypeSubscribeProofUpdates, []byte{})
	if err != nil {
		return err
	}
	<-c.ack

	return nil
}

func (c *Client) UnsubscribeProofUpdates() error {
	err := c.conn.WriteMessage(wire.MessageTypeUnsubscribeProofUpdates, []byte{})
	if err != nil {
		return err
	}
	<-c.ack

	return nil
}

func (c *Client) AppendLog(idx uint64, logId [32]byte, statement []byte) error {
	l := wire.NewSignedLogStatement(idx, logId, statement)
	hash := fastsha256.Sum256(l.Statement.Bytes())
	sig, err := c.key.Sign(hash[:])

	if err != nil {
		return err
	}
	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		return err
	}
	l.Signature = csig

	err = c.conn.WriteMessage(wire.MessageTypeAppendLog, l.Bytes())
	if err != nil {
		return err
	}

	// Wait for ack
	<-c.ack
	return nil
}
