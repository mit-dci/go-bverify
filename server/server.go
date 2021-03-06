package server

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"

	"github.com/tidwall/buntdb"

	"github.com/mit-dci/go-bverify/bitcoin/blockchain"
	"github.com/mit-dci/go-bverify/bitcoin/btcutil"
	"github.com/mit-dci/go-bverify/bitcoin/chaincfg"
	btcwire "github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/mpt"
	"github.com/mit-dci/go-bverify/utils"
	"github.com/mit-dci/go-bverify/wallet"
	"github.com/mit-dci/go-bverify/wire"
)

type ServerState struct {
	LastCommitmentTree          []byte
	LastConfirmedCommitmentTree []byte
}

type Server struct {
	// Tracks the pubkeys for LogIDs
	logIDToPubKey map[[32]byte][33]byte

	// Guards the logIDToPubKey map
	logIDToPubKeyLock sync.Mutex

	// Tracks the pubkeys for LogIDs
	logIDIndex map[[32]byte]uint64

	// Guards the logIDIndex map
	logIDIndexLock sync.Mutex

	// The full MPT tracking all client logs
	fullmpt *mpt.FullMPT

	// The last state of the MPT when we last committed
	LastCommitMpt *mpt.FullMPT

	// The state of the MPT upon confirmation from the chain (mined commitment
	// transaction)
	LastConfirmedCommitMpt *mpt.FullMPT

	// Lock guarding the MPT
	mptLock sync.Mutex

	// Cache of the last root committed to the blockchain
	lastCommitment [32]byte

	// Cache of the last delta for on-demand proof deltas
	lastDelta *mpt.DeltaMPT

	// Address (port) to run the server on
	addr string

	// Processing threads
	allProcessors  []LogProcessor
	processors     []LogProcessor
	processorsLock sync.Mutex

	// Wallet for keeping the funds used to commit to the chain
	wallet *wallet.Wallet

	// Database for keeping commitment history
	commitmentDb *buntdb.DB

	// In-memory array for keeping commitment history
	commitments []*wire.Commitment

	// channel to stop
	stop chan bool

	// channel to indicate the server is up and running
	ready   chan bool
	isReady bool

	// Listener for clients
	listener *net.TCPListener

	// The commitment server automatically commits every time a block has been
	// found. This can be turned off by switching this boolean off.
	AutoCommit bool

	// This can be used to switch off keeping a copy of the tree at the commitment
	// point (used to generate client-side proofs) - this is needed for some tests
	// and benchmarks where this is not needed to save time
	KeepCommitmentTree bool

	// Commit to the blockchain every N blocks (if AutoCommit enabled)
	CommitEveryNBlocks int

	// Last block that a commitment was initiated
	LastCommitHeight int

	// Blocks to rescan on startup
	RescanBlocks int

	// Full server also runs an actual Bitcoin wallet and commits to the actual chain
	Full bool

	// Allow disable signature verification to save needless processing  time when
	// initializing benchmarks
	CheckSignatures bool
}

func NewServer(addr string, rescanBlocks int) (*Server, error) {
	logging.SetLogLevel(int(logging.LogLevelDebug))

	srv := new(Server)
	srv.RescanBlocks = rescanBlocks
	srv.CheckSignatures = true
	srv.AutoCommit = true
	srv.KeepCommitmentTree = true
	srv.CommitEveryNBlocks = 1 // every hour (well, on bitcoin at least)
	srv.addr = addr

	if srv.addr == "" {
		srv.addr = ":9100"
	}

	srv.fullmpt, _ = mpt.NewFullMPT()
	srv.mptLock = sync.Mutex{}
	srv.logIDToPubKey = map[[32]byte][33]byte{}
	srv.logIDToPubKeyLock = sync.Mutex{}
	srv.logIDIndex = map[[32]byte]uint64{}
	srv.logIDIndexLock = sync.Mutex{}

	srv.lastCommitment = [32]byte{}
	srv.allProcessors = make([]LogProcessor, 0)
	srv.processors = make([]LogProcessor, 0)
	srv.processorsLock = sync.Mutex{}

	srv.stop = make(chan bool, 1)
	srv.ready = make(chan bool, 0)

	return srv, nil
}

func (srv *Server) RegisterLogID(logID [32]byte, controllingKey [33]byte) error {
	srv.logIDToPubKeyLock.Lock()
	_, ok := srv.logIDToPubKey[logID]
	srv.logIDToPubKeyLock.Unlock()

	if ok {
		return fmt.Errorf("Duplicate log ID created: [%x]", logID)
	}
	srv.logIDToPubKeyLock.Lock()
	srv.logIDToPubKey[logID] = controllingKey
	srv.logIDToPubKeyLock.Unlock()

	if srv.Full {
		// Persist the log ID and its controlling key
		err := srv.commitmentDb.Update(func(tx *buntdb.Tx) error {
			_, _, err := tx.Set(fmt.Sprintf("key-%x", logID), string(controllingKey[:]), nil)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (srv *Server) GetPubKeyForLogID(logID [32]byte) ([33]byte, error) {
	srv.logIDToPubKeyLock.Lock()
	pk, ok := srv.logIDToPubKey[logID]
	srv.logIDToPubKeyLock.Unlock()

	if !ok {
		return [33]byte{}, fmt.Errorf("LogID not found")
	}
	return pk, nil
}
func (srv *Server) GetNextLogIndex(logID [32]byte) uint64 {
	srv.logIDIndexLock.Lock()
	idx, ok := srv.logIDIndex[logID]
	srv.logIDIndexLock.Unlock()
	if !ok {
		return uint64(0)
	}
	return idx + 1
}

func (srv *Server) RegisterLogStatement(logID [32]byte, index uint64, statement []byte) error {
	srv.logIDIndexLock.Lock()
	idx, ok := srv.logIDIndex[logID]
	srv.logIDIndexLock.Unlock()
	if !ok && index != uint64(0) {
		return fmt.Errorf("Unexpected log index %d - expected 0", index)
	} else if ok && index != idx+1 {
		return fmt.Errorf("Unexpected log index %d - expected %d", index, idx+1)
	}
	srv.logIDIndexLock.Lock()
	srv.logIDIndex[logID] = index
	srv.logIDIndexLock.Unlock()

	if srv.Full {
		// Persist the index
		err := srv.commitmentDb.Update(func(tx *buntdb.Tx) error {
			_, _, err := tx.Set(fmt.Sprintf("idx-%x", logID), fmt.Sprintf("%d", index), nil)
			return err
		})
		if err != nil {
			return err
		}
	}

	srv.mptLock.Lock()
	logIdClean := make([]byte, 32)
	copy(logIdClean, logID[:])
	srv.fullmpt.Insert(logIdClean, statement)
	logIdClean = nil
	srv.mptLock.Unlock()

	return nil
}

func (srv *Server) Run() error {
	addr, err := net.ResolveTCPAddr("tcp", srv.addr)
	if err != nil {
		return err
	}

	srv.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	if srv.Full {
		os.MkdirAll(utils.DataDirectory(), 0700)

		logFilePath := path.Join(utils.DataDirectory(), "b_verify.log")
		logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		defer logFile.Close()
		logging.SetLogFile(logFile)

		params := &chaincfg.RegressionNetParams
		if utils.GetEnvOrDefault("BITCOINNET", "regtest") == "testnet" {
			params = &chaincfg.TestNet3Params
		} else if utils.GetEnvOrDefault("BITCOINNET", "regtest") == "mainnet" {
			params = &chaincfg.MainNetParams
		}

		srv.wallet, err = wallet.NewWallet(params, srv.RescanBlocks)
		if err != nil {
			return err
		}

		srv.commitmentDb, err = buntdb.Open(path.Join(utils.DataDirectory(), "commitment.db"))
		if err != nil {
			return err
		}

		newBlockChan := make(chan *btcwire.MsgBlock, 100)
		srv.wallet.AddNewBlockListener(newBlockChan)
		go srv.blockWatcher(newBlockChan)
		srv.loadState()
		srv.loadCommitments()
		srv.loadLogs()

		// After everything is loaded, we should touch our "special log" to trigger new
		// commitments, even if clients don't change anything. We did that after the last
		// commit before the server got shutdown, but that is only in memory.

		trigger := make([]byte, 32)
		rand.Read(trigger[:])

		srv.RegisterLogStatement([32]byte{}, srv.GetNextLogIndex([32]byte{}), trigger)
	}

	logging.Debugf("Server ready. Commitment: %x - Last committed at height: %d", srv.lastCommitment, srv.LastCommitHeight)
	// When we're starting a new server, we need to commit a fixed value to the
	// chain, to indicidate the starting of our commitment server.
	// Otherwise, how would you prove there hasn't been any previous commitments?
	// So if len(commitments) == 0 then ignore this clause, forcing the commitment
	// even if it's empty.
	if len(srv.commitments) == 0 {
		logging.Debugf("This is a fresh server. Before opening our doors, we'll have to do our maiden commitment.")
		if srv.Full {
			loggedWarning := false
			for {
				if srv.wallet.IsSynced() {
					break
				}
				if !loggedWarning {
					logging.Warnf("Waiting for wallet to be synced")
					loggedWarning = true
				}

				time.Sleep(1 * time.Second)
			}

			loggedWarning = false
			for {
				if srv.wallet.Balance() > 5000 {
					break
				}
				if !loggedWarning {
					logging.Warnf("We need at least 5000 satoshi balance in our wallet to be able to commit. Deposit that into your wallet to kick things off.")
					loggedWarning = true
				}

				time.Sleep(1 * time.Second)
			}
		}
		// Okay we have enough money now, so register a fixed log that will always
		// result in the same commitment hash. That way we can recognize that as the
		// "maiden hash" in each chain of commitments.
		srv.RegisterLogID([32]byte{}, [33]byte{})
		logHash := fastsha256.Sum256([]byte("Maiden commitment for b_verify"))
		srv.RegisterLogStatement([32]byte{}, 0, logHash[:])
		err := srv.Commit()
		if err != nil {
			panic(err)
		}
	}

	srv.isReady = true

	select {
	case srv.ready <- true:
	default:
	}

	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			select {
			case <-srv.stop:
				return nil
			default:
				continue
			}
		}
		proc := NewLogProcessor(conn, srv)
		srv.allProcessors = append(srv.allProcessors, proc)
		go proc.Process()
	}

	return nil
}

func (srv *Server) blockWatcher(wc chan *btcwire.MsgBlock) {
	for {
		block := <-wc

		logging.Debugf("Received new block in blockwatcher")

		// check if our last commit is in here
		err := srv.processMerkleProofs(block)
		if err != nil {
			logging.Errorf("Error getting merkle proofs from block: %s", err.Error())
		}

		blocksSince := srv.wallet.Height() - srv.LastCommitHeight
		if blocksSince >= srv.CommitEveryNBlocks {
			logging.Debugf("Reached commit threshold. Committing to chain")
			pending := srv.getPendingCommitments()
			if len(pending) == 0 {
				if srv.isReady {
					err := srv.Commit()
					if err != nil {
						logging.Errorf("Error while committing to chain: %s", err.Error())
					}
				} else {
					logging.Warnf("Server not ready, not committing")
				}
			} else {
				logging.Debugf("We still have a pending commitment, waiting for it to be mined before committing again")
			}
		} else {
			logging.Debugf("Got new block, %d since last commit (commit every %d) - waiting", blocksSince, srv.CommitEveryNBlocks)
		}
	}
}

func (srv *Server) loadCommitments() {
	srv.commitments = make([]*wire.Commitment, 0)
	err := srv.commitmentDb.View(func(tx *buntdb.Tx) error {
		tx.AscendRange("", "commitment-", "commitmenu-", func(key, value string) bool {
			srv.commitments = append(srv.commitments, wire.CommitmentFromBytes([]byte(value)))
			return true
		})
		return nil
	})
	if err != nil {
		logging.Errorf("[Server] Error loading commitments: %s", err.Error())
		return
	}

	logging.Debugf("Loaded %d previous commitments", len(srv.commitments))

	// Sort in right order
	commitments := make([]*wire.Commitment, 0)
	for {
		if len(commitments) == len(srv.commitments) {
			break
		}
		lenBefore := len(commitments)
		if len(commitments) == 0 {
			// Find maiden commitment and add that
			for _, c := range srv.commitments {
				if bytes.Equal(c.Commitment[:], utils.MaidenHash()) {
					commitments = append(commitments, c)
					break
				}
			}
		} else {
			// Find commitment that spends last commitment's outpoint 1
			for _, c := range srv.commitments {
				tx := btcwire.NewMsgTx(1)
				err := tx.Deserialize(bytes.NewBuffer(c.RawTx))
				if err != nil {
					err = fmt.Errorf("Commitment %x has unparseable rawTX: %s", c.Commitment, err.Error())
					panic(err)
				}
				if tx.TxIn[0].PreviousOutPoint.Index == 1 && tx.TxIn[0].PreviousOutPoint.Hash.IsEqual(commitments[len(commitments)-1].TxHash) {
					commitments = append(commitments, c)
					break
				}
			}
		}

		if len(commitments) == lenBefore {
			err = fmt.Errorf("Could not reconstruct commitment chain from disk")
			panic(err)
		}
	}

	srv.commitments = commitments
	if len(srv.commitments) > 0 {
		srv.LastCommitHeight = srv.commitments[len(srv.commitments)-1].TriggeredAtBlockHeight
	}
}

func (srv *Server) loadLogs() {
	srv.logIDToPubKeyLock.Lock()
	srv.logIDIndexLock.Lock()
	err := srv.commitmentDb.View(func(tx *buntdb.Tx) error {
		tx.AscendRange("", "key-", "key.", func(key, value string) bool {
			logID, _ := hex.DecodeString(key[4:])
			logID32 := [32]byte{}
			copy(logID32[:], logID)
			controllingKey := [33]byte{}
			copy(controllingKey[:], []byte(value))

			srv.logIDToPubKey[logID32] = controllingKey
			return true
		})

		tx.AscendRange("", "idx-", "idx.", func(key, value string) bool {
			logID, _ := hex.DecodeString(key[4:])
			logID32 := [32]byte{}
			copy(logID32[:], logID)
			idx, _ := strconv.ParseUint(value, 10, 64)

			srv.logIDIndex[logID32] = idx
			return true
		})
		return nil
	})

	srv.logIDToPubKeyLock.Unlock()
	srv.logIDIndexLock.Unlock()

	if err != nil {
		logging.Errorf("[Server] Error loading logs: %s", err.Error())
		return
	}
}

func (srv *Server) saveCommitment(c *wire.Commitment) {
	alreadyAtIdx := -1
	for i, sc := range srv.commitments {
		if bytes.Equal(c.Commitment[:], sc.Commitment[:]) {
			alreadyAtIdx = i
		}
	}

	if c.TriggeredAtBlockHeight > srv.LastCommitHeight {
		srv.LastCommitHeight = c.TriggeredAtBlockHeight
	}

	if alreadyAtIdx > -1 {
		srv.commitments[alreadyAtIdx] = c
	} else {
		srv.commitments = append(srv.commitments, c)
	}
	err := srv.commitmentDb.Update(func(dtx *buntdb.Tx) error {
		key := fmt.Sprintf("commitment-%x", c.Commitment)
		_, _, err := dtx.Set(key, string(c.Bytes()), nil)
		return err
	})
	if err != nil {
		logging.Errorf("[Server] Error saving commitment: %s", err.Error())
	}
}

func (srv *Server) getPendingCommitments() []*wire.Commitment {
	r := make([]*wire.Commitment, 0)
	for _, c := range srv.commitments {
		if c.IncludedInBlock == nil {
			r = append(r, c)
		}
	}
	return r
}

func (srv *Server) processMerkleProofs(block *btcwire.MsgBlock) error {
	logging.Debugf("Processing block %s for commitments", block.BlockHash().String())
	pending := srv.getPendingCommitments()

	for _, c := range srv.commitments { // Scan all commitments. Commitments can reorg.
		commitmentInBlock := false
		for _, tx := range block.Transactions {
			hash := tx.TxHash()
			if hash.IsEqual(c.TxHash) {
				commitmentInBlock = true
				break
			}
		}

		if commitmentInBlock {
			blockHash := block.BlockHash()
			if c.IncludedInBlock != nil && !c.IncludedInBlock.IsEqual(&blockHash) {
				logging.Debugf("Commitment %x was in block %s, now %s", c.Commitment, c.IncludedInBlock.String(), blockHash.String())
			} else {
				logging.Debugf("Commitment %x is in block %s", c.Commitment, blockHash.String())
			}
			merkleRoot := block.Header.MerkleRoot
			txs := make([]*btcutil.Tx, len(block.Transactions))
			for i, tx := range block.Transactions {
				txs[i] = btcutil.NewTx(tx)
			}
			hashes := blockchain.BuildMerkleTreeStore(txs, false)

			// next, find the index of our txid
			hashIdx := -1
			for i, h := range hashes {
				if bytes.Equal(h.CloneBytes(), c.TxHash[:]) {
					hashIdx = i
					break
				}
			}

			proof := utils.NewMerkleProof(hashes, uint64(hashIdx))

			// sanity check
			if !proof.Check(c.TxHash, &merkleRoot) {
				panic(fmt.Errorf("Merkle root doesn't match"))
			}

			c.MerkleProof = proof
			c.IncludedInBlock = &blockHash

			if bytes.Equal(srv.lastCommitment[:], c.Commitment[:]) {
				srv.LastConfirmedCommitMpt, _ = srv.LastCommitMpt.Copy()
				srv.commitState()
			}

			srv.saveCommitment(c)
		}
	}

	pending = srv.getPendingCommitments()
	if len(pending) > 0 {
		logging.Debugf("We still have %d pending commitments:", len(pending))
	}

	return nil
}

func (srv *Server) registerProcessor(p LogProcessor) {
	srv.processorsLock.Lock()
	srv.processors = append(srv.processors, p)
	srv.processorsLock.Unlock()
}

func (srv *Server) unregisterProcessor(p LogProcessor) {
	srv.processorsLock.Lock()
	removeIdx := -1
	for i, pp := range srv.processors {
		if pp == p {
			removeIdx = i
		}
	}
	if removeIdx != -1 {
		srv.processors = append(srv.processors[:removeIdx], srv.processors[removeIdx+1:]...)
	}
	srv.processorsLock.Unlock()
}

func (srv *Server) Stop() {
	for _, p := range srv.allProcessors {
		p.Stop()
	}
	srv.stop <- true
	srv.listener.Close()
}

func (srv *Server) Commitment() []byte {
	srv.mptLock.Lock()
	defer srv.mptLock.Unlock()
	return srv.fullmpt.Commitment()
}

func (srv *Server) CountNodes() int {
	return srv.fullmpt.CountNodes()
}

func (srv *Server) CountRecalculations() int {
	return srv.fullmpt.CountRecalculations()
}

func (srv *Server) Commit() error {
	srv.mptLock.Lock()
	commitment := srv.fullmpt.Commitment()
	if bytes.Equal(srv.lastCommitment[:], commitment[:]) {
		commitment = nil
		logging.Debugf("No changes to commit")
		srv.mptLock.Unlock()
		return nil
	}

	var err error
	// Retain the full MPT at the time of commitment to be able to serve
	// proofs
	copy(srv.lastCommitment[:], commitment[:])

	if srv.KeepCommitmentTree {
		if srv.LastCommitMpt != nil {
			srv.LastCommitMpt.Dispose()
		}
		srv.LastCommitMpt, err = srv.fullmpt.Copy()
		if err != nil {
			srv.mptLock.Unlock()
			return err
		}
	}
	if srv.lastDelta != nil {
		srv.lastDelta.Dispose()
	}

	srv.lastDelta, _ = mpt.NewDeltaMPT(srv.fullmpt)

	srv.processorsLock.Lock()
	var wg sync.WaitGroup
	for _, pr := range srv.processors {
		wg.Add(1)
		go func(proc LogProcessor) {
			proc.SendProofs(srv.lastDelta)
			wg.Done()
		}(pr)
	}
	wg.Wait()

	srv.processorsLock.Unlock()

	srv.fullmpt.Reset()

	srv.mptLock.Unlock()

	if srv.Full {
		txID, rawTx, err := srv.wallet.Commit(commitment[:])
		if err != nil {
			return err
		}

		comm32 := [32]byte{}
		copy(comm32[:], commitment)
		c := wire.NewCommitment(comm32, txID, rawTx, srv.wallet.Height())
		srv.saveCommitment(c)
		logging.Debugf("Committed to chain: %s", txID.String())

		srv.commitState()

		// change something in the tree to force a commitment next time around
		nextIdx := srv.GetNextLogIndex([32]byte{})
		srv.RegisterLogStatement([32]byte{}, nextIdx, commitment)
	}
	commitment = nil
	return nil
}

func (srv *Server) commitState() error {
	commitState := ServerState{}
	commitState.LastCommitmentTree = srv.LastCommitMpt.Bytes()
	if srv.LastConfirmedCommitMpt != nil {
		commitState.LastConfirmedCommitmentTree = srv.LastConfirmedCommitMpt.Bytes()
	}
	stateBytes, err := json.Marshal(commitState)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path.Join(utils.DataDirectory(), "serverstate.hex"), stateBytes, 0600)
}

func (srv *Server) loadState() error {
	b, err := ioutil.ReadFile(path.Join(utils.DataDirectory(), "serverstate.hex"))
	if err != nil {
		if err != os.ErrNotExist {
			return err
		} else {
			return nil
		}
	}

	commitState := ServerState{}
	json.Unmarshal(b, &commitState)

	lastCommitmentBuffer := bytes.NewBuffer(commitState.LastCommitmentTree)

	srv.LastCommitMpt, err = mpt.DeserializeNewFullMPT(lastCommitmentBuffer)
	if err != nil {
		return err
	}

	if commitState.LastConfirmedCommitmentTree != nil {
		lastConfirmedCommitmentBuffer := bytes.NewBuffer(commitState.LastConfirmedCommitmentTree)
		srv.LastConfirmedCommitMpt, err = mpt.DeserializeNewFullMPT(lastConfirmedCommitmentBuffer)
		if err != nil {
			return err
		}
	}

	srv.fullmpt, err = srv.LastCommitMpt.Copy()
	if err != nil {
		return err
	}

	copy(srv.lastCommitment[:], srv.LastCommitMpt.Commitment())

	return nil
}

func (srv *Server) GetProofForKeys(keys [][]byte) (*mpt.PartialMPT, error) {
	if srv.Full {
		if srv.LastConfirmedCommitMpt == nil {
			return nil, fmt.Errorf("There has not yet been a confirmed commitment, please try again later")
		}
		return mpt.NewPartialMPTIncludingKeys(srv.LastConfirmedCommitMpt, keys)
	} else {
		if srv.LastCommitMpt == nil {
			return nil, fmt.Errorf("There has not yet been a confirmed commitment, please try again later")
		}
		return mpt.NewPartialMPTIncludingKeys(srv.LastCommitMpt, keys)
	}
}

func (srv *Server) GetDeltaProofForKeys(keys [][]byte) (*mpt.DeltaMPT, error) {
	return srv.lastDelta.GetUpdatesForKeys(keys)
}

func (srv *Server) GetCommitmentDetails(commitment [32]byte) (*wire.Commitment, error) {
	null := [32]byte{}
	if bytes.Equal(null[:], commitment[:]) {
		return srv.commitments[len(srv.commitments)-1], nil
	}

	for _, c := range srv.commitments {
		if bytes.Equal(c.Commitment[:], commitment[:]) {
			// Return a clone
			return wire.CommitmentFromBytes(c.Bytes()), nil
		}
	}
	return nil, fmt.Errorf("Commitment not found")
}

func (srv *Server) GetCommitmentHistory(sinceCommitment [32]byte) []*wire.Commitment {
	logging.Debugf("Fetching commit history since %x", sinceCommitment)
	startIdx := int(-1)
	null := [32]byte{}
	if !bytes.Equal(null[:], sinceCommitment[:]) {
		for i, c := range srv.commitments {
			if bytes.Equal(c.Commitment[:], sinceCommitment[:]) {
				startIdx = i
				break
			}
		}
	}

	logging.Debugf("Start index is %d", startIdx)

	// Create a new array that we're gonna return. It'll
	// contain all commitments we have.
	commitments := make([]*wire.Commitment, 0)
	if len(srv.commitments) > startIdx {
		for _, c := range srv.commitments[startIdx+1:] {
			// Clone each commitment into the array we return
			// We don't want the caller to mess anything up to our
			// in-memory array.
			comm := wire.CommitmentFromBytes(c.Bytes())
			if comm.IncludedInBlock != nil {
				// Only include mined commitments
				commitments = append(commitments, comm)
			} else {
				logging.Debugf("Skipping commitment %x since it's not included in a block yet", comm.Commitment)
			}

		}
	}
	return commitments
}

func (srv *Server) TreeSize() int {
	return srv.fullmpt.ByteSize()
}

func (srv *Server) TreeGraph() []byte {
	return srv.fullmpt.Graph()
}
