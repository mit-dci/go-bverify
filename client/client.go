// Client package is responsible for connecting to a server, and if wanted
// checking proofs of both the chain commitment and the inclusion of our logs
// in the commitment
package client

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/utils"
	"github.com/tidwall/buntdb"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
	"github.com/mit-dci/go-bverify/bitcoin/coinparam"
	btcwire "github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/client/uspv"
	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/crypto/sig64"
	"github.com/mit-dci/go-bverify/mpt"
	"github.com/mit-dci/go-bverify/wire"
)

type Client struct {
	// The connection to the server
	conn *wire.Connection

	// The key we're using to sign our statements
	key      *btcec.PrivateKey
	keyBytes []byte
	pubKey   [33]byte

	// The SPV connection to the blockchain
	spv *uspv.SPVCon

	// Channels for receiving replies from the server from
	// the receive loop
	ack           chan bool
	errChan       chan error
	proof         chan *mpt.PartialMPT
	commitDetails chan *wire.Commitment
	commitHistory chan []*wire.Commitment

	// You can set these function pointers to receive events
	// from the client (errors and proof updates)
	OnError       func(error, *Client)
	OnProofUpdate func([]byte, *Client)

	// The local data stored by the client
	db *buntdb.DB

	// The address we connected to when NewClient() was used
	addr string

	// The simple HTTP RPC server you can use to write
	// new logs and statements
	rpcServer *RpcServer

	// A cache of the last server commitment
	lastServerCommitment *wire.Commitment

	// Clients can run in simple mode (where they're just a means to communicate
	// with the server) or as full client (that runs checks on the commitments,
	// fetches and caches bitcoin headers, etcetera).
	fullClient bool

	// FastMode means you can log more than once every commitment. This is achieved
	// by forming a hash-chain of statements. The proof size will however grow linearly
	// with the amount of statements between the commitments, you will need to include
	// the hashes of all statements up until the next commitment
	FastMode bool

	// Ready indicates the client is ready to serve commands (only relevant for FullClient)
	Ready bool

	// When connecting using NewClient() the address is kept. When there's a failure
	// the server will disconnect us for, it will automatically reconnect.
	ReconnectOnFailure bool
}

// NewClientWithConnection creates a new b_verify client using the provided
// private key bytes and net.Conn
func NewClientWithConnection(key []byte, c net.Conn) (*Client, error) {

	// Create the keypair from the passed in byte array
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key)

	// Pre-generate and cache the public key
	var pk [33]byte
	copy(pk[:], pub.SerializeCompressed())

	// Create the Client struct we're going to return
	cli := &Client{
		conn:          wire.NewConnection(c),
		spv:           new(uspv.SPVCon),
		keyBytes:      key,
		key:           priv,
		pubKey:        pk,
		commitDetails: make(chan *wire.Commitment),
		commitHistory: make(chan []*wire.Commitment),
		proof:         make(chan *mpt.PartialMPT),
		ack:           make(chan bool),
		errChan:       make(chan error),
		fullClient:    false,
	}

	// Start the loop that processes incoming response messages
	go cli.ReceiveLoop()

	return cli, nil
}

// NewClient will create a new Client that connects to the server and port
// specified in addr
func NewClient(key []byte, addr string) (*Client, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	cli, err := NewClientWithConnection(key, c)
	if err != nil {
		return nil, err
	}
	cli.ReconnectOnFailure = true
	cli.addr = addr
	return cli, nil
}

// UsesKey return true if this client is using the key passed in. We don't
// want to expose the key as a public variable but if you know the key
// yourself, we can tell you if it matches.
func (c *Client) UsesKey(key []byte) bool {
	return bytes.Equal(key, c.keyBytes)
}

// StartSPV will initiate the process of downloading headers from the blockchain
// to allow us to check SPV proofs for the commitment transactions
func (c *Client) StartSPV() error {
	return c.spv.Start(&coinparam.TestNet3Params)
}

// SPVSynced returns true if there's no more headers to fetch for us (and we can
// assume we've synced up all the headers from the blockchain)
func (c *Client) SPVSynced() bool {
	return c.spv.Synced
}

func (c *Client) SPVHeight() int32 {
	return c.spv.GetHeaderTipHeight()
}

// SPVAskHeaders will (re)initiate the header synchronization if we have the idea
// that we might have stale data, we can call this to talk to our known peers and
// fetch new headers.
func (c *Client) SPVAskHeaders() error {
	return c.spv.AskForHeaders()
}

func (c *Client) Reconnect() {
	logging.Debug("Reconnecting to server")
	newConn, err := net.Dial("tcp", c.addr)
	if err != nil {
		logging.Errorf("Could not reconnect to server: %s", err.Error())
		go func(cli *Client) {
			time.Sleep(5 * time.Second)
			cli.Reconnect()
		}(c)
		return
	}
	c.conn = wire.NewConnection(newConn)
	go c.ReceiveLoop()
}

// ReceiveLoop will fetch new messages as they come in on the wire (from the server)
// and try to process them accordingly.
func (c *Client) ReceiveLoop() {
	for {
		t, p, err := c.conn.ReadNextMessage()
		if err != nil {
			logging.Debugf("Error reading message from server connection: %s", err.Error())
			// If we can't read from this transport anymore, or we receive invalid
			// data, we should close the connection and exit the receive loop
			c.conn.Close()
			if c.ReconnectOnFailure {
				c.Reconnect()
			}
			return
		}

		// If we receive an Ack message, send a boolean over the ack channel.
		// client functions that expect an ack will wait by reading from this
		// channel
		if t == wire.MessageTypeAck {
			select {
			case c.ack <- true:
			default:
				logging.Warn("Received ACK when no one was listening for it")
			}
			continue
		}

		// If we receive an error from the server, we should call the OnError
		// hook if it's set, and then exit the receive loop
		if t == wire.MessageTypeError {
			err := fmt.Errorf("%s", string(p))
			logging.Debugf("Received error on wire: %s", err.Error())
			if c.OnError != nil {
				go c.OnError(err, c)
			}
			select {
			case c.errChan <- err:
			default:
			}
			c.conn.Close()
			if c.ReconnectOnFailure {
				c.Reconnect()
			}
			return
		}

		// MessageTypeProofUpdate is an automatic message with the
		// delta since the last proof update, provided we have subscribed using
		// SubscribeProofUpdates. We will call the OnProofUpdate hook with the
		// message body.
		if t == wire.MessageTypeProofUpdate {
			if c.OnProofUpdate != nil {
				go c.OnProofUpdate(p, c)
			}
			continue
		}

		// MessageTypeProof is a full proof that's been requested
		// by the client. Hence, we try to parse it as a MPT and then send it over the
		// proof channel. This channel is read from in the RequestProof method.
		if t == wire.MessageTypeProof {
			buf := bytes.NewBuffer(p)
			mpt, err := mpt.DeserializeNewPartialMPT(buf)
			if err != nil {
				// Something wrong parsing the returned MPT data. Close the connection
				// and exit the loop.
				c.conn.Close()
				return
			}

			// Whoever requested the proof is listening on c.proof for the result
			// so send it there
			select {
			case c.proof <- mpt:
			default:
			}
			continue
		}

		// MessageTypeCommitmentDetails contains the details of a single commitment.
		// This is requested by the client using GetCommitmentDetails
		if t == wire.MessageTypeCommitmentDetails {
			msg, err := wire.NewCommitmentDetailsMessageFromBytes(p)
			if err != nil {
				// Something wrong parsing the returned commitment details.
				// Close the connection and exit the loop.
				c.conn.Close()
				return
			}

			// Whoever requested the commitment details is listening on
			// c.commitDetails for the result so send it there
			select {
			case c.commitDetails <- msg.Commitment:
			default:
			}
			continue
		}

		// MessageTypeCommitmentHistory contains the details of all commitments
		// (optionally since a particular commitment). This is requested by the
		// client using GetCommitmentHistory. The server returns this as  a
		// collection of Commitment objects.
		if t == wire.MessageTypeCommitmentHistory {
			msg, err := wire.NewCommitmentHistoryMessageFromBytes(p)
			if err != nil {
				// Something wrong parsing the returned commitment details.
				// Close the connection and exit the loop.
				c.conn.Close()
				return
			}
			// Whoever requested the commitments is listening on
			// c.commitHistory for the result so send it there
			select {
			case c.commitHistory <- msg.Commitments:
			default:
			}
			continue
		}
	}
}

// Run will kick off the full client functionality, that also stores its key in
// a keyfile in the user's home directory, keep track of log statements and their
// proofs, as well as commitments and their merkle proof to the blockchain block
// headers. This is the "complete" functionality for b_verify.
func (c *Client) Run(resync bool) error {
	var err error

	// Once Run() is called, we are a full client
	c.fullClient = true

	logging.Debugf("Using data directory %s", utils.ClientDataDirectory())

	// Create the data directory. If it exists, this will yield an error
	// but we're ignoring that.
	err = os.MkdirAll(utils.ClientDataDirectory(), 0700)
	if err == nil {
		logging.Debugf("Created data directory %s", utils.ClientDataDirectory())
	} else {
		logging.Debugf("Could not create data directory: %s", err.Error())

	}

	// Open the database to store commitments and logs
	c.db, err = buntdb.Open(path.Join(utils.ClientDataDirectory(), "data.db"))
	if err != nil {
		return err
	}

	// Configure the log level and log file path
	logging.SetLogLevel(int(logging.LogLevelDebug))
	logFilePath := path.Join(utils.ClientDataDirectory(), "b_verify_client.log")
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logFile.Close()
	logging.SetLogFile(logFile)

	// Load the client-side signing key from a file.
	keyFile := path.Join(utils.ClientDataDirectory(), "privkey.hex")
	key32 := [32]byte{}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		// The keyfile does not exist. Let's generate a key and write it.
		rand.Read(key32[:])
		ioutil.WriteFile(keyFile, key32[:], 0600)
	} else if err != nil {
		return err
	} else {
		key, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return err
		}
		copy(key32[:], key)
	}

	// Now that we have read the key we can configure the client to use
	// these keys
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key32[:])
	var pk [33]byte
	copy(pk[:], pub.SerializeCompressed())
	c.keyBytes = key32[:]
	c.pubKey = pk
	c.key = priv

	if resync {
		err = c.ClearCommitments()
		if err != nil {
			return err
		}
	}

	// Start the SPV process that downloads headers from the blockchain
	go func() {
		err := c.StartSPV()
		if err != nil {
			panic(err)
		}
	}()

	// Load things into memory from our database
	c.loadStuff()

	// Start the RPC server as a means to create and append to logs
	c.rpcServer = NewRpcServer(c)
	go func() {
		err = c.rpcServer.Start()
		if err != nil {
			panic(err)
		}
	}()

	c.Ready = true

	// Start the verification loop that checks server commitments against
	// the blockchain and proofs against the commitment
	c.verifyLoop()

	return nil
}

// StartLogText is a convenience function called by the RPC server to start
// a new log with a particular piece of information. This function will take
// care of hashing it before passing it to StartLog. It will also keep the
// clear text (preimage) in our database for future verification purposes.
func (c *Client) StartLogText(initialStatement string) ([32]byte, error) {
	statementHash := fastsha256.Sum256([]byte(initialStatement))

	// Create the log using the hash
	logId, err := c.StartLog(statementHash[:])
	if err != nil {
		return [32]byte{}, err
	}

	if c.fullClient {
		// Store the preimage in the database
		err := c.db.Update(func(dtx *buntdb.Tx) error {
			key := fmt.Sprintf("logpreimage-%x-000000000", logId[:])
			_, _, err := dtx.Set(key, initialStatement, nil)
			return err
		})
		if err != nil {
			return [32]byte{}, err
		}
	}

	return logId, nil
}

// StartLog will start a new log on the server. It will locally calculate the
// LogID, since that's deterministic. It will sign the CreateLog instruction
// with the client's key and then send it to the server. It will wait for the
// server to have acknowledged the log.
func (c *Client) StartLog(initialStatement []byte) ([32]byte, error) {

	// Create the message
	l := wire.NewSignedCreateLogStatement(c.pubKey, initialStatement)

	// Calculate the Log ID
	logId := fastsha256.Sum256(l.CreateStatement.Bytes())

	// Coincidentally, the logID is the same as the hash we need to sign.
	// So here we sign that, and add the signature to the outgoing message
	sig, err := c.key.Sign(logId[:])
	if err != nil {
		return [32]byte{}, err
	}
	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		return [32]byte{}, err
	}
	l.Signature = csig

	// Send the message to the server
	err = c.conn.WriteMessage(wire.MessageTypeCreateLog, l.Bytes())
	if err != nil {
		return [32]byte{}, err
	}

	serverHash := fastsha256.Sum256(l.Bytes())

	// Wait for ack
	select {
	case <-c.ack:
	case err = <-c.errChan:
		return [32]byte{}, err
	case <-time.After(10 * time.Second):
		return [32]byte{}, fmt.Errorf("Timeout waiting for ACK")
	}

	// If we're running as a full client, we should store the log
	if c.fullClient {
		err := c.db.Update(func(dtx *buntdb.Tx) error {
			// Store the hash of the log
			key := fmt.Sprintf("loghash-%x-000000000", logId[:])
			_, _, err := dtx.Set(key, string(serverHash[:]), nil)
			if err != nil {
				return err
			}

			// Store the index as "last one for this log"
			key = fmt.Sprintf("lastidx-%x", logId[:])
			_, _, err = dtx.Set(key, "0", nil)
			if err != nil {
				return err
			}

			// Write this marker key to allow us to enumerate all logs
			key = fmt.Sprintf("log-%x", logId[:])
			_, _, err = dtx.Set(key, string("1"), nil)
			return err
		})
		if err != nil {
			return [32]byte{}, err
		}
	}

	return logId, nil
}

// GetLastHash returns the last known hash for the given log
func (c *Client) GetLastHash(logId [32]byte) (int64, [32]byte, error) {
	idx := int64(-1)
	hash := [32]byte{}
	err := c.db.View(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("lastidx-%x", logId[:])
		val, err := tx.Get(key)
		if err != nil {
			if err == buntdb.ErrNotFound {
				return nil
			}
			return err
		}

		idx, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}

		key = fmt.Sprintf("loghash-%x-%09d", logId[:], idx)
		val, err = tx.Get(key)
		if err != nil {
			return err
		}
		copy(hash[:], []byte(val))

		return nil
	})
	return idx, hash, err
}

// GetLastCommittedHash returns the last committed hash for the given log
func (c *Client) GetLastCommittedLog(logId [32]byte) (int64, [32]byte, error) {
	idx := int64(-1)
	hash := [32]byte{}
	err := c.db.View(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("lastidx-%x", logId[:])
		val, err := tx.Get(key)
		if err != nil {
			if err == buntdb.ErrNotFound {
				return nil
			}
			return err
		}

		idx, err = strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}

		return nil
	})

	for {
		if idx < 0 {
			return -1, hash, fmt.Errorf("No committed hash found")
		}

		if c.IsCommitted(logId, uint64(idx)) {
			val := ""
			err = c.db.View(func(tx *buntdb.Tx) error {
				key := fmt.Sprintf("loghash-%x-%09d", logId[:], idx)
				val, err = tx.Get(key)
				if err != nil {
					return err
				}
				copy(hash[:], []byte(val))
				return nil
			})
			if val != "" {
				break
			}
		}

		idx--
	}

	return idx, hash, err
}

func (c *Client) GetLogPreimage(logId [32]byte, idx uint64) (string, error) {
	preimage := ""
	err := c.db.View(func(tx *buntdb.Tx) error {
		var err error
		key := fmt.Sprintf("logpreimage-%x-%09d", logId[:], idx)
		preimage, err = tx.Get(key)
		if err != nil {
			key := fmt.Sprintf("logpreimage-%x-%d", logId[:], idx)
			preimage, err = tx.Get(key)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return preimage, err
}

// RequestProof asks the server for a full proof of the passed in LogIDs. These
// are the LogIDs returned from CreateLog() or CreateLogText(). The return value
// is a partial MPT that only contains the paths from these logs to the root.
func (c *Client) RequestProof(logIds [][32]byte) (*mpt.PartialMPT, error) {
	// Create the wire message and send it to the server
	msg := wire.NewRequestProofMessage(logIds)
	err := c.conn.WriteMessage(wire.MessageTypeRequestProof, msg.Bytes())
	if err != nil {
		return nil, err
	}

	var proof *mpt.PartialMPT
	// Wait for the proof response and return it to the client

	select {
	case proof = <-c.proof:
	case err = <-c.errChan:
		return nil, err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("Timeout waiting for proof")
	}

	return proof, nil
}

func (c *Client) GetAllLogIDs() ([][32]byte, error) {
	logIds := make([][32]byte, 0)
	err := c.db.View(func(tx *buntdb.Tx) error {
		tx.AscendRange("", "log-", "log.", func(key, value string) bool {
			logId, _ := hex.DecodeString(key[4:])
			logId32 := [32]byte{}
			copy(logId32[:], logId)
			logIds = append(logIds, logId32)
			return true
		})
		return nil
	})
	return logIds, err
}

func (c *Client) GetLogCommitment(logId [32]byte, idx uint64) ([32]byte, error) {
	commitmentHash := [32]byte{}
	err := c.db.View(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("logcommitment-%x-%09d", logId[:], idx)
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		copy(commitmentHash[:], []byte(val))
		return nil
	})
	return commitmentHash, err
}

// SubscribeProofUpdates will tell the server that we want to receive delta
// proofs as soon as the server commits a new value to the chain. The server
// will then send us a ProofUpdate message automatically, which the client can
// read out by setting a function pointer to OnProofUpdate
func (c *Client) SubscribeProofUpdates() error {
	// Create the wire message and send it to the server
	err := c.conn.WriteMessage(wire.MessageTypeSubscribeProofUpdates, []byte{})
	if err != nil {
		return err
	}

	// Wait for ack
	select {
	case <-c.ack:
	case err = <-c.errChan:
		return err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("Timeout waiting for ACK")
	}

	return nil
}

// UnsubscribeProofUpdates will tell the server to stop sending us automatic
// proof updates. In that case, the proofs will have to be requested manually
func (c *Client) UnsubscribeProofUpdates() error {
	// Create the wire message and send it to the server
	err := c.conn.WriteMessage(wire.MessageTypeUnsubscribeProofUpdates, []byte{})
	if err != nil {
		return err
	}

	// Wait for ack
	select {
	case <-c.ack:
	case err = <-c.errChan:
		return err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("Timeout waiting for ACK")
	}

	return nil
}

// AppendLogText is a convenience function called by the RPC server to append
// a particular piece of information to the log. This function will take
// care of hashing it before passing it to AppendLog. It will also keep the
// clear text (preimage) in our database for future verification purposes.
func (c *Client) AppendLogText(idx uint64, logId [32]byte, statement string) error {
	statementHash := fastsha256.Sum256([]byte(statement))

	// Call Appendlog with the hashed statement
	err := c.AppendLog(idx, logId, statementHash[:])
	if err != nil {
		return err
	}

	// If we're running as a full client, we should store the log
	if c.fullClient {
		// Store the preimage in our database
		err := c.db.Update(func(dtx *buntdb.Tx) error {
			key := fmt.Sprintf("logpreimage-%x-%09d", logId[:], idx)
			_, _, err := dtx.Set(key, statement, nil)
			return err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) IsCommitted(logId [32]byte, idx uint64) bool {
	committed := false
	c.db.View(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("logcommitment-%x-%09d", logId[:], idx)
		val, err := tx.Get(key)
		if err != nil {
			return err
		}
		if len(val) > 0 {
			committed = true
		}
		return nil
	})
	return committed
}

func (c *Client) GetForeignLogIDAndHash(statement *wire.ForeignStatement) ([32]byte, [32]byte, error) {
	statementHash := fastsha256.Sum256([]byte(statement.StatementPreimage))
	hash := [32]byte{}
	logId := [32]byte{}
	if statement.InitialStatement {
		s := &wire.SignedCreateLogStatement{
			CreateStatement: &wire.CreateLogStatement{
				ControllingKey:   statement.PubKey,
				InitialStatement: statementHash[:],
			},
			Signature: statement.Signature,
		}

		err := s.VerifySignature()
		if err != nil {
			return logId, hash, err
		}

		logId = fastsha256.Sum256(s.CreateStatement.Bytes())
		hash = fastsha256.Sum256(s.Bytes())
	} else {
		s := &wire.SignedLogStatement{
			Statement: &wire.LogStatement{
				Index:     statement.Index,
				LogID:     statement.LogID,
				Statement: statementHash[:],
			},
			Signature: statement.Signature,
		}

		err := s.VerifySignature(statement.PubKey)
		if err != nil {
			return logId, hash, err
		}

		logId = statement.LogID
		hash = fastsha256.Sum256(s.Bytes())
	}
	return logId, hash, nil
}

// AddForeignLog will keep updating the logID's proofs for the given statement
func (c *Client) AddForeignLog(statement *wire.ForeignStatement) error {
	logId, hash, err := c.GetForeignLogIDAndHash(statement)
	if err != nil {
		return err
	}
	return c.db.Update(func(dtx *buntdb.Tx) error {

		// Store the hash of the log
		key := fmt.Sprintf("loghash-%x-999999999", logId[:])
		_, _, err := dtx.Set(key, string(hash[:]), nil)
		if err != nil {
			return err
		}

		key = fmt.Sprintf("lastidx-%x", logId[:])
		_, _, err = dtx.Set(key, "999999999", nil)
		if err != nil {
			return err
		}

		// Store the proof (if it's in this foreign statement)
		if statement.Proof != nil {

			// Store the commitment
			key = fmt.Sprintf("logcommitment-%x-999999999", logId[:])
			_, _, err = dtx.Set(key, string(statement.Proof.Commitment()), nil)
			if err != nil {
				return err
			}

			// Store the proof
			key = fmt.Sprintf("foreignlogproof-%x-999999999", logId[:])
			_, _, err = dtx.Set(key, string(statement.Proof.Bytes()), nil)
			if err != nil {
				return err
			}

		}

		// Store the preimage of the log
		key = fmt.Sprintf("logpreimage-%x-999999999", logId[:])
		_, _, err = dtx.Set(key, statement.StatementPreimage, nil)
		if err != nil {
			return err
		}

		// Write this marker key to allow us to enumerate all logs
		key = fmt.Sprintf("log-%x", logId[:])
		_, _, err = dtx.Set(key, string("1"), nil)
		return err
	})
}

// IsFollowingLog returns true when the passed LogID is merely being followed
// by this client, and not being actively written to.
func (c *Client) IsForeignLog(logId [32]byte) bool {
	isForeign := false
	c.db.View(func(tx *buntdb.Tx) error {
		// Fetch idx 999999999 - its existence will learn us that this is imported.
		key := fmt.Sprintf("loghash-%x-999999999", logId[:])
		_, err := tx.Get(key)
		if err == nil {
			isForeign = true
		}
		return nil
	})
	return isForeign
}

func (c *Client) AppendLog(idx uint64, logId [32]byte, statement []byte) error {
	if c.fullClient {
		lastIdx, lastHash, err := c.GetLastHash(logId)
		if err != nil {
			return err
		}
		if c.FastMode {
			// If we're in FastMode, we have to make the hashchain
			newHash := fastsha256.Sum256(append(lastHash[:], statement...))
			copy(statement[:], newHash[:])
		} else {
			// Otherwise, we check if our last statement is properly committed,
			// we shouldn't send another statement if this isn't the case.
			if !c.IsCommitted(logId, uint64(lastIdx)) {
				return fmt.Errorf("Last statement has not yet been committed to chain. You have to wait for this, or use FastMode")
			}
		}

		if idx != uint64(lastIdx+1) {
			return fmt.Errorf("Received out-of-sync index for log [%x]: expected %d, got %d", logId, lastIdx+1, idx)
		}
	}

	// Create the message
	l, err := c.SignedAppendLog(idx, logId, statement)
	if err != nil {
		return err
	}

	// Calculate the hash the server will write to the log
	serverHash := fastsha256.Sum256(l.Bytes())

	// Send the message to the server
	err = c.conn.WriteMessage(wire.MessageTypeAppendLog, l.Bytes())
	if err != nil {
		return err
	}

	// Wait for ack
	select {
	case <-c.ack:
	case err = <-c.errChan:
		return err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("Timeout waiting for ACK")
	}

	if c.fullClient {
		err := c.db.Update(func(dtx *buntdb.Tx) error {
			// Store the log statement hash in our data
			key := fmt.Sprintf("loghash-%x-%09d", logId[:], idx)
			_, _, err := dtx.Set(key, string(serverHash[:]), nil)
			if err != nil {
				return err
			}

			// Store the hash as "last one for this log"
			key = fmt.Sprintf("lastidx-%x", logId[:])
			_, _, err = dtx.Set(key, fmt.Sprintf("%d", idx), nil)
			return err
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// SignedAppendLog is a convenience function to generate a SignedLogStatement
// message using the key in this client
func (c *Client) SignedAppendLog(idx uint64, logId [32]byte, statement []byte) (*wire.SignedLogStatement, error) {
	// Create the message
	l := wire.NewSignedLogStatement(idx, logId, statement)

	// Hash the statement, which is what we'll sign
	hash := fastsha256.Sum256(l.Statement.Bytes())

	// Sign the hash
	sig, err := c.key.Sign(hash[:])
	if err != nil {
		return nil, err
	}
	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		return nil, err
	}

	// Add the signature to the message and return it
	l.Signature = csig
	return l, nil
}

// GetCommitmentHistory will request the server to send over commitment details
// for every commitment since sinceCommitment. If sinceCommitment is an empty
// byte array, all commitments will be returned.
func (c *Client) GetCommitmentHistory(sinceCommitment [32]byte) ([]*wire.Commitment, error) {
	// Create the message and send it to the server
	msg := wire.NewRequestCommitmentHistoryMessage(sinceCommitment)
	err := c.conn.WriteMessage(wire.MessageTypeRequestCommitmentHistory, msg.Bytes())
	if err != nil {
		return nil, err
	}

	// Read from the response channel and return to the client
	// TODO: timeout?
	hist := <-c.commitHistory
	return hist, nil
}

// GetCommitmentDetails will request the server to send over commitment details
// for a single commitment. If commitment is an empty byte array, the details of
// the last commitment will be returned.
func (c *Client) GetCommitmentDetails(commitment [32]byte) (*wire.Commitment, error) {
	// Create the message and send it to the server
	msg := wire.NewRequestCommitmentDetailsMessage(commitment)
	err := c.conn.WriteMessage(wire.MessageTypeRequestCommitmentDetails, msg.Bytes())
	if err != nil {
		return nil, err
	}

	// Read from the response channel and return to the client
	// TODO: timeout?
	details := <-c.commitDetails
	return details, nil
}

// GetBlockHeaderByHash will return a single block header from the SPV data based on
// the blockhash
func (c *Client) GetBlockHeaderByHash(hash *chainhash.Hash) (*btcwire.BlockHeader, error) {
	return c.spv.GetHeaderByBlockHash(hash)
}

// ExportLog will create a ForeignStatement out of the last committed statement in the given
// log
func (c *Client) ExportLog(logId [32]byte) (*wire.ForeignStatement, error) {
	idx, _, err := c.GetLastCommittedLog(logId)
	if err != nil {
		return nil, fmt.Errorf("Error fetching last committed log: %s", err.Error())
	}

	if c.IsForeignLog(logId) {
		return nil, fmt.Errorf("Cannot export a foreign log. The original sender should export it.")
	}

	fs := &wire.ForeignStatement{}
	fs.Index = uint64(idx)
	fs.InitialStatement = (idx == 0)
	fs.LogID = logId
	fs.StatementPreimage, err = c.GetLogPreimage(logId, uint64(idx))
	if err != nil {
		return nil, fmt.Errorf("Error fetching last committed statement: %s", err.Error())
	}
	fs.PubKey = c.pubKey

	commitment, err := c.GetLogCommitment(logId, uint64(idx))
	if err != nil {
		return nil, fmt.Errorf("Error fetching commitment hash for last committed statement: %s", err.Error())
	}

	proof, err := c.GetProofForCommitment(commitment, [][]byte{logId[:]})
	if err != nil {
		return nil, fmt.Errorf("Error fetching commitment hash for last committed statement: %s", err.Error())
	}

	fs.Proof = proof

	// Recreate signature. Maybe this is somewhat ugly, but... figure it out later
	statementHash := fastsha256.Sum256([]byte(fs.StatementPreimage))
	signaturePayload := []byte{}
	if fs.InitialStatement {
		signaturePayload = wire.NewSignedCreateLogStatement(fs.PubKey, statementHash[:]).CreateStatement.Bytes()
	} else {
		signaturePayload = wire.NewSignedLogStatement(uint64(fs.Index), logId, statementHash[:]).Statement.Bytes()
	}
	signatureHash := fastsha256.Sum256(signaturePayload)
	sig, err := c.key.Sign(signatureHash[:])
	if err != nil {
		return nil, fmt.Errorf("Could not recreate signature: %s", err.Error())

	}

	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		return nil, fmt.Errorf("Could not compress signature: %s", err.Error())
	}

	fs.Signature = csig

	return fs, nil
}
