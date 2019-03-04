package server

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mit-dci/go-bverify/mpt"
)

type Server struct {
	// Tracks the pubkeys for LogIDs
	logIDToPubKey sync.Map

	// Tracks the last log index per log
	logIDIndex sync.Map

	// The full MPT tracking all client logs
	fullmpt *mpt.FullMPT

	// The last state of the MPT when we last committed
	LastCommitMpt *mpt.FullMPT

	// Lock guarding the MPT
	mptLock sync.Mutex

	// Cache of the last root committed to the blockchain
	lastCommitment [32]byte

	// Address (port) to run the server on
	addr string

	// Processing threads
	processors     []LogProcessor
	processorsLock sync.Mutex

	// channel to stop
	stop  chan bool
	ready chan bool

	listener *net.TCPListener

	AutoCommit         bool
	KeepCommitmentTree bool
}

func NewServer(addr string) (*Server, error) {
	srv := new(Server)
	srv.AutoCommit = true
	srv.KeepCommitmentTree = true
	srv.addr = addr

	if srv.addr == "" {
		srv.addr = ":9100"
	}

	srv.fullmpt, _ = mpt.NewFullMPT()
	srv.mptLock = sync.Mutex{}

	srv.lastCommitment = [32]byte{}

	srv.processors = make([]LogProcessor, 0)
	srv.processorsLock = sync.Mutex{}

	srv.stop = make(chan bool, 1)
	srv.ready = make(chan bool, 0)

	return srv, nil
}

func (srv *Server) RegisterLogID(logID [32]byte, controllingKey [33]byte) error {
	_, ok := srv.logIDToPubKey.Load(logID)
	if ok {
		return fmt.Errorf("Duplicate log ID created: [%x]", logID)
	}
	srv.logIDToPubKey.Store(logID, controllingKey)
	return nil
}

func (srv *Server) GetPubKeyForLogID(logID [32]byte) ([33]byte, error) {
	pk, ok := srv.logIDToPubKey.Load(logID)
	if !ok {
		return [33]byte{}, fmt.Errorf("LogID not found")
	}
	return pk.([33]byte), nil
}

func (srv *Server) RegisterLogStatement(logID [32]byte, index uint64, statement []byte) error {
	idx, ok := srv.logIDIndex.Load(logID)
	if !ok && index != uint64(0) {
		return fmt.Errorf("Unexpected log index %d - expected 0", index)
	} else if ok && index != (idx.(uint64))+1 {
		return fmt.Errorf("Unexpected log index %d - expected %d", index, (idx.(uint64))+1)
	}
	srv.logIDIndex.Store(logID, index)

	srv.mptLock.Lock()
	srv.fullmpt.Insert(logID[:], statement)
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

	commitTicker := time.NewTicker(time.Second * 5)
	go func(s *Server) {
		for range commitTicker.C {
			if s.AutoCommit {
				s.Commit()
			}
		}
	}(srv)

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
		go proc.Process()
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
	srv.stop <- true
	srv.listener.Close()
}

func (srv *Server) Commit() error {
	srv.mptLock.Lock()
	defer srv.mptLock.Unlock()
	commitment := srv.fullmpt.Commitment()
	if bytes.Equal(srv.lastCommitment[:], commitment[:]) {
		commitment = nil
		return nil
	}

	var err error
	// Retain the full MPT at the time of commitment to be able to serve
	// proofs
	copy(srv.lastCommitment[:], commitment[:])

	if srv.KeepCommitmentTree {
		srv.LastCommitMpt, err = mpt.NewFullMPTFromBytes(srv.fullmpt.Bytes())
		if err != nil {
			return err
		}
	}

	delta, _ := mpt.NewDeltaMPT(srv.fullmpt)
	srv.processorsLock.Lock()
	var wg sync.WaitGroup
	for _, pr := range srv.processors {
		wg.Add(1)
		go func(proc LogProcessor) {
			proc.SendProofs(delta)
			wg.Done()
		}(pr)
	}
	wg.Wait()

	srv.processorsLock.Unlock()

	srv.fullmpt.Reset()

	commitment = nil
	return nil
}

func (srv *Server) GetProofForKeys(keys [][]byte) (*mpt.PartialMPT, error) {
	return mpt.NewPartialMPTIncludingKeys(srv.fullmpt, keys)
}

func (srv *Server) TreeSize() int {
	return srv.fullmpt.ByteSize()
}

func (srv *Server) TreeGraph() []byte {
	return srv.fullmpt.Graph()
}
