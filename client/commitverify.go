package client

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/mit-dci/go-bverify/logging"

	"github.com/tidwall/buntdb"
)

// updateProofs will be called after a new commitment has been properly verified
// and committted. We will request an updated proof for our logIDs and verify
// if the proofs are correct.
func (c *Client) updateProofs() error {
	logging.Debugf("Updating proofs")

	// First, create an array of all the logIDs we are keeping in this client
	logIds := make([][32]byte, 0)
	c.db.View(func(tx *buntdb.Tx) error {
		tx.AscendRange("", "log-", "log.", func(key, value string) bool {
			logId, _ := hex.DecodeString(key[4:])
			logId32 := [32]byte{}
			copy(logId32[:], logId)
			logIds = append(logIds, logId32)
			return true
		})
		return nil
	})

	// Request the proofs from the server
	proof, err := c.RequestProof(logIds)
	if err != nil {
		return err
	}

	// Calculate the commitment from the partial tree we got from the
	// server and check if it is a known commitment
	rootHash := proof.Commitment()
	_, err = c.getCommitment(rootHash)
	if err != nil {
		return err
	}

	logging.Debugf("Commitment %x is known to us and valid", rootHash)

	// Next, ensure all our logs are in the proof and the values the server
	// reports is a value that we have written to the server at some point
	// in history (check if the server didn't commit garbage)
	for _, l := range logIds {
		// Get the LogID from the proof
		val, err := proof.Get(l[:])
		if err != nil {
			return err
		}

		// Find the witness value in our history of values
		valueIdx := int64(-1)
		c.db.View(func(tx *buntdb.Tx) error {
			tx.DescendRange("", fmt.Sprintf("loghash-%x-999999999", l), fmt.Sprintf("loghash-%x-000000000", l), func(key, value string) bool {
				if bytes.Equal([]byte(value), val) {
					valueIdx, _ = strconv.ParseInt(key[41:], 0, 64)
					return false
				}
				return true
			})
			return nil
		})

		if valueIdx == -1 {
			// We don't know about the value the server committed. This is pretty
			// catastrophic.
			return fmt.Errorf("The value in the proof does not match any of the values we know. Not good.")
		}
	}

	logging.Debugf("Proof checks out, storing it in database")

	// Store the proof in our database
	return c.db.Update(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("proof-%x", rootHash)
		_, _, err := tx.Set(key, string(proof.Bytes()), nil)
		return err
	})
}
