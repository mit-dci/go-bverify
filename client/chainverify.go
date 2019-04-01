package client

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/mit-dci/go-bverify/utils"

	"github.com/mit-dci/go-bverify/logging"

	btcwire "github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/wire"
	"github.com/tidwall/buntdb"
)

// loadStuff loads the last server commitment into memory
func (c *Client) loadStuff() error {

	// First, we try to get the last commitment hash from the database
	hash := [32]byte{}
	err := c.db.View(func(tx *buntdb.Tx) error {
		lastCommitHash, err := tx.Get("commitment-last")
		if err != nil {
			return err
		}
		copy(hash[:], []byte(lastCommitHash))
		return nil
	})
	if err != nil {
		// If the error is a not found, we'll just return and leave the
		// lastServerCommitment empty.
		if err != buntdb.ErrNotFound {
			// Something else went wrong, return the error
			return err
		}
		return nil
	}

	// Fetch the commitment details of the last known commitment
	c.lastServerCommitment, err = c.getCommitment(hash[:])
	return err
}

// getCommitment retrieves the commitment details of a single commitment
// from the database by its hash
func (c *Client) getCommitment(hash []byte) (*wire.Commitment, error) {
	var comm *wire.Commitment
	err := c.db.View(func(tx *buntdb.Tx) error {
		b, err := tx.Get(fmt.Sprintf("commitment-%x", hash))
		if err != nil {
			return err
		}

		comm = wire.CommitmentFromBytes([]byte(b))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return comm, nil
}

// verifyCommitment will check a commitment's validity. It will verify the
// validity of the commitment transaction
func (c *Client) verifyCommitment(comm *wire.Commitment) error {
	logging.Debugf("Verifying commitment %x (block %s)", comm.Commitment, comm.IncludedInBlock.String())
	logging.Debugf("Transaction is %s", comm.TxHash.String())

	// First and foremost, check if the block specified by the server is actually
	// known to us in the header chain.
	retry := 10
	var header *btcwire.BlockHeader
	var err error

	// Start a retry loop. If the block isn't found right away, ask for headers
	// and try again
	for {
		header, err = c.GetBlockHeaderByHash(comm.IncludedInBlock)
		if err != nil {
			retry--
			if retry == 0 {
				return err
			}
			// First try to sync headers and then try again
			c.SPVAskHeaders()

			for {
				if c.SPVSynced() {
					break
				}
				time.Sleep(1 * time.Second)
			}
		} else {
			break
		}
	}

	logging.Debugf("Found the block specified in the commitment in our header chain")

	// Now that we've found the header, we should verify the provided merkle proof
	// actually checks out with the block's merkle root.
	checksOut := comm.MerkleProof.Check(comm.TxHash, &header.MerkleRoot)
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
	tx.Deserialize(bytes.NewBuffer(comm.RawTx))

	if bytes.Equal(comm.Commitment[:], utils.MaidenHash()) {
		logging.Debugf("Skipping the TXO chain check since this is the first commitment")
	} else {
		logging.Debugf("Previous outpoint expected: [%x/1] - First input: [%x/%d]", c.lastServerCommitment.TxHash[:], tx.TxIn[0].PreviousOutPoint.Hash[:], tx.TxIn[0].PreviousOutPoint.Index)
		if tx.TxIn[0].PreviousOutPoint.Index != 1 || !tx.TxIn[0].PreviousOutPoint.Hash.IsEqual(c.lastServerCommitment.TxHash) {
			return fmt.Errorf("Commitment transaction's first input is not the change output of the last commitment. This breaks the chain and is invalid.")
		}
	}
	logging.Debugf("Everything checks out, this commitment is valid!")

	return nil
}

// ClearCommitments delets the client-side cache of the server commitments. This will
// trigger a resynchronization of the commitments and proofs.
func (c *Client) ClearCommitments() error {
	keys := make([]string, 0)
	err := c.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendRange("", "commitment-", "commitment.", func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
	})
	if err != nil {
		return err
	}
	return c.db.Update(func(tx *buntdb.Tx) error {
		for _, k := range keys {
			_, err := tx.Delete(k)
			if err != nil {
				return err
			}
		}
		return nil
	})

}

func (c *Client) getAllCommitments() ([]*wire.Commitment, error) {
	returnVal := make([]*wire.Commitment, 0)
	err := c.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendRange("", "commitment-", "commitment.", func(key, value string) bool {
			if key != "commitment-last" {
				returnVal = append(returnVal, wire.CommitmentFromBytes([]byte(value)))
			}
			return true
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(returnVal, func(i, j int) bool {
		return returnVal[i].TriggeredAtBlockHeight < returnVal[j].TriggeredAtBlockHeight
	})

	return returnVal, nil
}

// saveCommitment writes the details of a commitment to our client-side database
func (c *Client) saveCommitment(comm *wire.Commitment) error {
	err := c.db.Update(func(dtx *buntdb.Tx) error {
		key := fmt.Sprintf("commitment-%x", comm.Commitment)
		_, _, err := dtx.Set(key, string(comm.Bytes()), nil)
		if err != nil {
			return err
		}

		// Store a mapping from block to commitment if we want to look up
		// the reverse
		key = fmt.Sprintf("block-%x", comm.IncludedInBlock[:])
		_, _, err = dtx.Set(key, string(comm.Commitment[:]), nil)
		if err != nil {
			return err
		}

		// Also store the last commitment
		_, _, err = dtx.Set("commitment-last", string(comm.Commitment[:]), nil)
		return err
	})
	if err != nil {
		return err
	}

	// Set the commitment as the last one
	c.lastServerCommitment = comm
	return nil
}

// verifyLoop is the full-client's main loop that will check validity of both the
// commitment transaction and the commitment proofs. It does this every 20 seconds
func (c *Client) verifyLoop() {
	for {
		if c.SPVSynced() {
			lastCommitHash := [32]byte{}
			if c.lastServerCommitment != nil {
				copy(lastCommitHash[:], c.lastServerCommitment.Commitment[:])
			}

			// Fetch server commitments since our last known commitment
			hist, err := c.GetCommitmentHistory(lastCommitHash)
			if err != nil {
				panic(err)
			}

			logging.Debugf("Got %d commitments", len(hist))

			// For each commitment, verify if it's correct and then save it
			retry := false
			for _, comm := range hist {
				err = c.verifyCommitment(comm)
				if err != nil {
					// If the block is not found, it could mean the commitment
					// transaction reorged. We'll get the updated blockhash
					// when we ask the server again - once the server has
					// processed the reorg.
					if err.Error() == "Block not found" {

						logging.Warnf("The server says commitment %x is in block %s, but we don't have that. Retry (could be reorg-related)", comm.Commitment[:], comm.IncludedInBlock.String())
						time.Sleep(time.Second * 20)
						retry = true
						break
					}
					panic(err)
				}

				err = c.saveCommitment(comm)
				if err != nil {
					panic(err)
				}
			}

			if retry {
				continue
			}

			// If we got new commitments, we should also update proofs
			if len(hist) > 0 {
				err = c.updateProofs()
				if err != nil {
					panic(err)
				}
			}

			time.Sleep(time.Second * 20)
		} else {
			time.Sleep(time.Second * 1)
		}

	}
}
