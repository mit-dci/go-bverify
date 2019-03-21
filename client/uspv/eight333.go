package uspv

import (
	"fmt"
	"os"

	"github.com/mit-dci/go-bverify/logging"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
	"github.com/mit-dci/go-bverify/bitcoin/wire"
)

const (
	// keyFileName and headerFileName are not referred in this file? -- takaya
	keyFileName    = "testseed.hex"
	headerFileName = "headers.bin"

	// VERSION hardcoded for now, probably ok...?
	// 70012 is for segnet... make this an init var?
	VERSION = 70015
)

// HashAndHeight is needed instead of just height in case a fullnode
// responds abnormally (?) by sending out of order merkleblocks.
// we cache a merkleroot:height pair in the queue so we don't have to
// look them up from the disk.
// Also used when inv messages indicate blocks so we can add the header
// and parse the txs in one request instead of requesting headers first.
type HashAndHeight struct {
	blockhash chainhash.Hash
	height    int32
	final     bool // indicates this is the last merkleblock requested
}

// NewRootAndHeight saves like 2 lines.
func NewRootAndHeight(b chainhash.Hash, h int32) (hah HashAndHeight) {
	hah.blockhash = b
	hah.height = h
	return
}

// IngestHeaders takes in a bunch of headers, checks them,
// and if they're OK, appends them to the local header file.
// If there are no headers, it assumes we're done and returns false.
// Otherwise it assumes there's more to request and returns true.
func (s *SPVCon) IngestHeaders(m *wire.MsgHeaders) (bool, error) {

	// headerChainLength is how many headers we give to the
	// verification function.  In bitcoin you never need more than 2016 previous
	// headers to figure out the validity of the next; some alcoins need more
	// though, like 4K or so.
	//	headerChainLength := 4096

	gotNum := int64(len(m.Headers))
	if gotNum > 0 {
		logging.Infof("got %d headers. Range:\n%s - %s\n",
			gotNum, m.Headers[0].BlockHash().String(),
			m.Headers[len(m.Headers)-1].BlockHash().String())
	} else {
		logging.Infof("got 0 headers, we're probably synced up")
		s.Synced = true
		return false, nil
	}

	s.headerMutex.Lock()
	// even though we will be doing a bunch without writing, should be
	// OK performance-wise to keep it locked for this function duration,
	// because verification is pretty quick.
	defer s.headerMutex.Unlock()

	reorgHeight, err := CheckHeaderChain(s.headerFile, m.Headers, s.Param)
	if err != nil {
		// insufficient depth reorg means we're still trying to sync up?
		// really, the re-org hasn't been proven; if the remote node
		// provides us with a new block we'll ask again.
		if reorgHeight == -1 {
			logging.Errorf("Header error: %s\n", err.Error())
			return false, nil
		}
		// some other error
		return false, err
	}

	// truncate header file if reorg happens
	if reorgHeight != 0 {
		fileHeight := reorgHeight - s.Param.StartHeight
		err = s.headerFile.Truncate(int64(fileHeight) * 80)
		if err != nil {
			return false, err
		}

		s.syncHeight = reorgHeight
	}

	// a header message is all or nothing; if we think there's something
	// wrong with it, we don't take any of their headers
	for _, resphdr := range m.Headers {
		// write to end of file
		err = resphdr.Serialize(s.headerFile)
		if err != nil {
			return false, err
		}
	}
	logging.Infof("Added %d headers OK.", len(m.Headers))
	return true, nil
}

// AskForHeaders ...
func (s *SPVCon) AskForHeaders() error {
	s.Synced = false
	ghdr := wire.NewMsgGetHeaders()
	ghdr.ProtocolVersion = s.localVersion

	tipheight := s.GetHeaderTipHeight()
	logging.Infof("got header tip height %d\n", tipheight)
	// get tip header, as well as a few older ones (inefficient...?)
	// yes, inefficient; really we should use "getheaders" and skip some of this

	tipheader, err := s.GetHeaderAtHeight(tipheight)
	if err != nil {
		logging.Errorf("AskForHeaders GetHeaderAtHeight error\n")
		return err
	}

	tHash := tipheader.BlockHash()
	err = ghdr.AddBlockLocatorHash(&tHash)
	if err != nil {
		return err
	}

	backnum := int32(1)

	// add more blockhashes in there if we're high enough
	for tipheight > s.Param.StartHeight+backnum {
		backhdr, err := s.GetHeaderAtHeight(tipheight - backnum)
		if err != nil {
			return err
		}
		backhash := backhdr.BlockHash()

		err = ghdr.AddBlockLocatorHash(&backhash)
		if err != nil {
			return err
		}

		// send the most recent 10 blockhashes, then get sparse
		if backnum > 10 {
			backnum <<= 2
		} else {
			backnum++
		}
	}

	logging.Infof("get headers message has %d header hashes, first one is %s\n",
		len(ghdr.BlockLocatorHashes), ghdr.BlockLocatorHashes[0].String())

	s.outMsgQueue <- ghdr
	return nil
}

// AskForBlocks requests blocks from current to last
// right now this asks for 1 block per getData message.
// Maybe it's faster to ask for many in each message?
func (s *SPVCon) AskForBlocks() error {
	var hdr wire.BlockHeader

	s.headerMutex.Lock() // lock just to check filesize
	stat, err := os.Stat(s.headerFile.Name())
	if err != nil {
		return err
	}
	s.headerMutex.Unlock() // checked, unlock
	endPos := stat.Size()

	// move back 1 header length to read
	headerTip := int32(endPos/80) + (s.headerStartHeight - 1)

	logging.Infof("blockTip to %d headerTip %d\n", s.syncHeight, headerTip)
	if s.syncHeight > headerTip {
		return fmt.Errorf("error- db longer than headers! shouldn't happen.")
	}
	if s.syncHeight == headerTip {
		// nothing to ask for; set wait state and return
		logging.Infof("no blocks to request, entering wait state\n")
		logging.Infof("%d bytes received\n", s.RBytes)
		s.inWaitState <- true

		// check if we can grab outputs
		// Do this on wallit level instead
		//		err = s.GrabAll()
		//		if err != nil {
		//			return err
		//		}
		// also advertise any unconfirmed txs here
		//		s.Rebroadcast()
		// ask for mempool each time...?  put something in to only ask the
		// first time we sync...?
		//		if !s.Ironman {
		//			s.AskForMempool()
		//		}
		return nil
	}

	logging.Debugf("will request blocks %d to %d\n", s.syncHeight+1, headerTip)
	reqHeight := s.syncHeight

	// loop through all heights where we want merkleblocks.
	for reqHeight < headerTip {
		reqHeight++ // we're requesting the next header

		// load header from file
		s.headerMutex.Lock() // seek to header we need
		_, err = s.headerFile.Seek(
			int64((reqHeight-s.headerStartHeight)*80), os.SEEK_SET)
		if err != nil {
			return err
		}
		err = hdr.Deserialize(s.headerFile) // read header, done w/ file for now
		s.headerMutex.Unlock()              // unlock after reading 1 header
		if err != nil {
			logging.Errorf("header deserialize error!\n")
			return err
		}

		bHash := hdr.BlockHash()
		// create inventory we're asking for
		var iv1 *wire.InvVect
		// if hardmode, ask for legit blocks, none of this ralphy stuff
		// I don't think you can have a queue for SPV.  You miss stuff.
		// also ask if someone wants rawblocks, like the watchtower

		iv1 = wire.NewInvVect(wire.InvTypeFilteredBlock, &bHash)
		gdataMsg := wire.NewMsgGetData()
		// add inventory
		err = gdataMsg.AddInvVect(iv1)
		if err != nil {
			return err
		}

		hah := NewRootAndHeight(hdr.BlockHash(), reqHeight)
		if reqHeight == headerTip { // if this is the last block, indicate finality
			hah.final = true
		}
		// waits here most of the time for the queue to empty out
		s.blockQueue <- hah // push height and mroot of requested block on queue
		s.outMsgQueue <- gdataMsg
	}
	return nil
}
