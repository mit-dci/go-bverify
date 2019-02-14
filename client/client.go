package client

import (
	"fmt"
	"net"

	"github.com/mit-dci/go-bverify/crypto/sig64"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"

	"github.com/mit-dci/go-bverify/crypto/btcec"

	"github.com/mit-dci/go-bverify/wire"
)

type Client struct {
	conn   *wire.Connection
	key    *btcec.PrivateKey
	ack    chan bool
	pubKey [33]byte
}

func NewClient(key []byte) (*Client, error) {
	c, err := net.Dial("tcp", "127.0.0.1:9100")
	if err != nil {
		return nil, err
	}
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)
	var pk [33]byte
	copy(pk[:], pub.SerializeCompressed())

	cli := &Client{conn: wire.NewConnection(c), key: priv, pubKey: pk, ack: make(chan bool, 1)}
	go cli.ReceiveLoop()

	return cli, nil
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
		}

		if t == wire.MessageTypeError {
			fmt.Printf("Received error: %s\n", string(p))
			return
		}

		if t == wire.MessageTypeProofUpdate {
			fmt.Printf("Received proof update: [%x]\n", p)
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
