package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/tidwall/buntdb"

	btcwire "github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/client/rpc"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/utils"
	"github.com/mit-dci/go-bverify/wire"
)

var db *buntdb.DB
var lastServerCommitment *wire.Commitment
var cli *client.Client
var maidenHash []byte

const ()

func main() {
	var err error
	hostName := flag.String("host", "localhost", "Host to connect to")
	hostPort := flag.Int("port", 9100, "Port to connect to")
	flag.Parse()

	// This is a fixed hash that the server will commit to first before even
	// becoming available to clients. This to ensure we always get the entire
	// chain when requesting commitments.
	maidenHash, _ = hex.DecodeString("523e59cfc5235b915dc89de188d87449453b083a8b7d97c1ee64d875da403361")

	os.MkdirAll(utils.ClientDataDirectory(), 0700)

	db, err = buntdb.Open(path.Join(utils.ClientDataDirectory(), "data.db"))
	if err != nil {
		panic(err)
	}

	logging.SetLogLevel(int(logging.LogLevelDebug))

	logFilePath := path.Join(utils.ClientDataDirectory(), "b_verify_client.log")
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logFile.Close()
	logging.SetLogFile(logFile)

	// generate key
	keyFile := path.Join(utils.ClientDataDirectory(), "privkey.hex")
	key32 := [32]byte{}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		rand.Read(key32[:])
		ioutil.WriteFile(keyFile, key32[:], 0600)
	} else if err != nil {
		panic(err)
	} else {
		key, err := ioutil.ReadFile(keyFile)
		if err != nil {
			panic(err)
		}
		copy(key32[:], key)
	}

	logging.Debugf("Starting new client and connecting to %s:%d...", *hostName, *hostPort)
	cli, err = client.NewClient(key32[:], fmt.Sprintf("%s:%d", *hostName, *hostPort))
	if err != nil {
		panic(err)
	}

	logging.Debugf("Connected, starting SPV connection")
	go func() {
		err = cli.StartSPV()
		if err != nil {
			panic(err)
		}
	}()

	logging.Debugf("Starting receive loop")
	go cli.ReceiveLoop()

	loadStuff()

	rpcServer := rpc.NewRpcServer(cli)
	go func() {
		err = rpcServer.Start()
		if err != nil {
			panic(err)
		}
	}()

	verifyLoop()
}

func loadStuff() error {
	hash := [32]byte{}
	err := db.View(func(tx *buntdb.Tx) error {
		lastCommitHash, err := tx.Get("commitment-last")
		if err != nil {
			return err
		}
		copy(hash[:], []byte(lastCommitHash))
		return nil
	})
	if err != nil {
		if err != buntdb.ErrNotFound {
			return err
		}
		return nil
	}

	lastServerCommitment, err = getCommitment(hash)
	return err
}

func getCommitment(hash [32]byte) (*wire.Commitment, error) {
	var c *wire.Commitment
	err := db.View(func(tx *buntdb.Tx) error {
		b, err := tx.Get(fmt.Sprintf("commitment-%x", hash))
		if err != nil {
			return err
		}

		c = wire.CommitmentFromBytes([]byte(b))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func verifyCommitment(c *wire.Commitment) error {
	logging.Debugf("Verifying commitment %x (block %x)", c.Commitment, c.IncludedInBlock[:])

	// First and foremost, check if the block specified by the server is actually
	// known to us in the header chain.
	header, err := cli.GetBlockHeaderByHash(c.IncludedInBlock)
	if err != nil {
		return err
	} else {
		logging.Debugf("Found the block specified in the commitment in our header chain")
	}

	// Now that we've found the header, we should verify the provided merkle proof
	// actually checks out with the block's merkle root.
	checksOut := c.MerkleProof.Check(c.TxHash, &header.MerkleRoot)
	if !checksOut {
		return fmt.Errorf("Merkle proof is incorrect")
	} else {
		logging.Debugf("Merkle proof is correct")
	}

	// This already looks very good, but in order to prove non-equivocation (the server
	// might have made _more_ commitments than just this one. In order to check that,
	// we check that the first input to the transaction is actually output 1 (the change)
	// of the last commitment transaction. This is not needed for the "maiden" commitment
	// which always has the hash 523e59cfc5235b915dc89de188d87449453b083a8b7d97c1ee64d875da403361

	tx := btcwire.NewMsgTx(1)
	tx.Deserialize(bytes.NewBuffer(c.RawTx))

	if bytes.Equal(c.Commitment[:], maidenHash) {
		logging.Debugf("Skipping the TXO chain check since this is the first commitment")
	} else {
		if tx.TxIn[0].PreviousOutPoint.Index != 1 || tx.TxIn[0].PreviousOutPoint.Hash.IsEqual(lastServerCommitment.TxHash) {
			return fmt.Errorf("Commitment transaction's first input is not the change output of the last commitment. This breaks the chain and is invalid.")
		}
	}
	logging.Debugf("Everything checks out, this commitment is valid!")

	return nil
}

func saveCommitment(c *wire.Commitment) error {
	err := db.Update(func(dtx *buntdb.Tx) error {
		key := fmt.Sprintf("commitment-%x", c.Commitment)
		_, _, err := dtx.Set(key, string(c.Bytes()), nil)
		if err != nil {
			return err
		}
		_, _, err = dtx.Set("commitment-last", string(c.Commitment[:]), nil)
		return err
	})
	if err != nil {
		return err
	}
	lastServerCommitment = c
	return nil
}

func verifyLoop() {
	for {
		lastCommitHash := [32]byte{}
		if lastServerCommitment != nil {
			copy(lastCommitHash[:], lastServerCommitment.Commitment[:])
		}

		logging.Debugf("Fetching server commitments since our last known commitment %x", lastCommitHash)

		hist, err := cli.GetCommitmentHistory(lastCommitHash)
		if err != nil {
			panic(err)
		}

		logging.Debugf("Got %d commitments", len(hist))

		for _, c := range hist {
			err = verifyCommitment(c)
			if err != nil {
				panic(err)
			}

			err = saveCommitment(c)
			if err != nil {
				panic(err)
			}
		}

		time.Sleep(time.Second * 20)

	}
}
