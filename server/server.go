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
	logIDToPubKey map[[32]byte][33]byte
	// Lock for R/W of logIDToPubKey
	logIDLock sync.RWMutex

	// Tracks the last log index per log
	logIDIndex map[[32]byte]uint64
	// Lock for R/W of logIDIndex
	logIDIndexLock sync.RWMutex

	// The full MPT tracking all client logs
	fullmpt *mpt.FullMPT
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
}

func NewServer(addr string) (*Server, error) {
	srv := new(Server)
	srv.addr = addr

	if srv.addr == "" {
		srv.addr = ":9100"
	}

	srv.fullmpt, _ = mpt.NewFullMPT()
	srv.mptLock = sync.Mutex{}

	srv.lastCommitment = [32]byte{}

	srv.logIDToPubKey = map[[32]byte][33]byte{}
	srv.logIDIndex = map[[32]byte]uint64{}

	srv.logIDIndexLock = sync.RWMutex{}
	srv.logIDLock = sync.RWMutex{}

	srv.processors = make([]LogProcessor, 0)
	srv.processorsLock = sync.Mutex{}

	srv.stop = make(chan bool, 1)
	srv.ready = make(chan bool, 0)

	return srv, nil
}

func (srv *Server) RegisterLogID(logID [32]byte, controllingKey [33]byte) error {
	srv.logIDLock.RLock()
	_, ok := srv.logIDToPubKey[logID]
	srv.logIDLock.RUnlock()
	if ok {
		return fmt.Errorf("Duplicate log ID created: [%x]", logID)
	}
	srv.logIDLock.Lock()
	insert := [33]byte{}
	copy(insert[:], controllingKey[:])
	srv.logIDToPubKey[logID] = insert
	srv.logIDLock.Unlock()
	return nil
}

func (srv *Server) GetPubKeyForLogID(logID [32]byte) ([33]byte, error) {
	srv.logIDLock.RLock()
	pk, ok := srv.logIDToPubKey[logID]
	srv.logIDLock.RUnlock()
	if !ok {
		return [33]byte{}, fmt.Errorf("LogID not found")
	}
	// return a copy
	returnKey := [33]byte{}
	copy(returnKey[:], pk[:])
	return returnKey, nil
}

func (srv *Server) RegisterLogStatement(logID [32]byte, index uint64, statement []byte) error {
	srv.logIDIndexLock.RLock()
	idx, ok := srv.logIDIndex[logID]
	srv.logIDIndexLock.RUnlock()
	if !ok && index != uint64(0) {
		return fmt.Errorf("Unexpected log index %d - expected 0", index)
	} else if ok && index != idx+1 {
		return fmt.Errorf("Unexpected log index %d - expected %d", index, idx+1)
	}
	srv.logIDIndexLock.Lock()
	srv.logIDIndex[logID] = index
	srv.logIDIndexLock.Unlock()

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
			s.Commit()
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
		return nil
	}
	copy(srv.lastCommitment[:], commitment[:])
	delta, _ := mpt.NewDeltaMPT(srv.fullmpt)
	srv.processorsLock.Lock()
	for _, pr := range srv.processors {
		pr.SendProofs(delta)
	}
	srv.processorsLock.Unlock()

	srv.fullmpt.Reset()
	return nil
}
