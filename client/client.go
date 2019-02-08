package client

import (
	"crypto/rand"
	"io/ioutil"
	"net"

	"github.com/mit-dci/lit/sig64"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"

	"github.com/mit-dci/zksigma/btcec"

	"github.com/mit-dci/go-bverify/wire"
)

type Client struct {
	conn   wire.Connection
	key    btcec.PrivateKey
	pubKey [33]byte
}

func NewClient(key []byte) (*Client, error) {
	c, err := net.Dial("tcp", "localhost:9100")
	if err != nil {
		return nil, err
	}

	key, err := ioutil.ReadFile("privkey.hex")
	if err != nil {
		key = new([]byte, 32)
		rand.Read(key)
		ioutil.WriteFile("privkey.hex", key, 0600)
	}

	priv := btcec.PrivKeyFromBytes(btcec.S256(), key)
	var pk [33]byte
	copy(pk[:], priv.PubKey().SerializeCompressed())
	return &Client{conn: wire.NewConnection(c), key: priv, pubKey: pk}, nil
}

func (c *Client) StartLog(initialStatement []byte) ([32]byte, error) {

	l := wire.NewSignedCreateLogStatement(c.pubKey, initialStatement)
	hash := fastsha256.Sum256(l.CreateStatement.Bytes())
	sig, err := c.key.Sign(hash)

	if err != nil {
		return nil, err
	}
	csig, err := sig64.SigCompress(sig)
	if err != nil {
		return nil, err
	}
	l.Signature = csig

	err := c.conn.WriteMessage(wire.MessageTypeCreateLog, l.Bytes())
	if err != nil {
		return nil, err
	}
	var logId [32]byte
	copy(logId[:], hash)

	return logId

}

func (c *Client) AppendLog(idx uint64, logId [32]byte, statement []byte) error {
	l := wire.NewSignedLogStatement(idx, logId, statement)
	hash := fastsha256.Sum256(l.Statement.Bytes())
	sig, err := c.key.Sign(hash)

	if err != nil {
		return err
	}
	csig, err := sig64.SigCompress(sig)
	if err != nil {
		return err
	}
	l.Signature = csig

	err := c.conn.WriteMessage(wire.MessageTypeAppendLog, l.Bytes())
	if err != nil {
		return err
	}

	return nil
}
