package client

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/mit-dci/go-bverify/mpt"

	"github.com/mit-dci/go-bverify/logging"

	"github.com/tidwall/buntdb"
)

// updateProofs will be called after a new commitment has been properly verified
// and committted. We will request an updated proof for our logIDs and verify
// if the proofs are correct.
func (c *Client) updateProofs() error {
	logging.Debugf("Updating proofs")

	// First, create an array of all the logIDs we are keeping in this client
	logIds, err := c.GetAllLogIDs()
	if err != nil {
		return err
	}

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
	//
	// Store the logIdx per log so we can commit that

	logIdxes := map[[32]byte]uint64{}

	for _, l := range logIds {
		// Get the LogID from the proof
		val, err := proof.Get(l[:])
		if err != nil {
			return err
		}

		// Find the witness value in our history of values
		valueIdx := int64(-1)
		c.db.View(func(tx *buntdb.Tx) error {
			tx.DescendRange("", fmt.Sprintf("loghash-%x-999999999", l), fmt.Sprintf("loghash-%x-00000000/", l), func(key, value string) bool {
				if bytes.Equal([]byte(value), val) {
					logging.Debugf("Found matching value in key %s\n", key)
					logging.Debugf("Found matching value in key %s\n", key[73:])
					valueIdx, _ = strconv.ParseInt(key[73:], 10, 64)
					logging.Debugf("Found matching value in idx %d\n", valueIdx)
					return false
				}
				return true
			})
			return nil
		})
		if valueIdx == -1 {
			// We don't know about the value the server committed. This is pretty catastrophic if we're the ones
			// that are maintaining the log. Though not if we're just following it. If we are following the log,
			// then this situation means our log's proof became invalid.
			if c.IsForeignLog(l) {
				// Ignore
				continue
			}

			return fmt.Errorf("The value in the proof does not match any of the values we know. Not good.")
		} else {
			logIdxes[l] = uint64(valueIdx)
		}
	}

	logging.Debugf("Proof checks out, storing it in database")

	// Store the proof in our database
	return c.db.Update(func(tx *buntdb.Tx) error {
		key := fmt.Sprintf("proof-%x", rootHash)
		_, _, err := tx.Set(key, string(proof.Bytes()), nil)
		if err != nil {
			return err
		}

		for _, l := range logIds {
			idx, ok := logIdxes[l]
			if ok {
				key = fmt.Sprintf("logcommitment-%x-%09d", l[:], idx)
				_, _, err := tx.Set(key, string(rootHash), nil)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (c *Client) GetProofForCommitment(commitment [32]byte, logIds [][]byte) (*mpt.PartialMPT, error) {
	var fullTree *mpt.FullMPT
	// Proof can be stored for more than one logId and we might want the proof for only one. Further slim
	// down the PartialMPT by loading it as FullMPT and further decreasing it
	err := c.db.View(func(tx *buntdb.Tx) error {
		proof, err := tx.Get(fmt.Sprintf("proof-%x", commitment))
		if err != nil {
			return err
		}

		fullTree, err = mpt.NewFullMPTFromBytes([]byte(proof))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mpt.NewPartialMPTIncludingKeys(fullTree, logIds)
}
