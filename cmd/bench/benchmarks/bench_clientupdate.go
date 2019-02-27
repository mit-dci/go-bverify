package benchmarks

import (
	"net"

	"github.com/mit-dci/go-bverify/client"

	"github.com/mit-dci/go-bverify/server"
	"github.com/mit-dci/go-bverify/wire"
)

const (
	CLIENTUPDATE_TOTALLOGS = 10000000
)

type receivedProofUpdate struct {
	forClient 	*client.Client
	proofUpdate    []byte
}

func newDummyClient(srv *server.Server) *wire.Connection {
	s, c := net.Pipe()
	p := server.NewLogProcessor(s, srv)
	go p.Process()
	return wire.NewConnection(c)
}

// RunClientUpdateBench will create 1M logs and then
// measure the time and size of a proof update for clients
// whose logs have not been changed (while there were other updates)
func RunClientUpdateBench() {
	srv, err := server.NewServer("")
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Create dummy logs, which returns the logIDs
	logIds := makeDummyLogs(srv, CLIENTUPDATE_TOTALLOGS)

	// Create clients
	proofLogs := []int{1, 10, 100, 1000}
	logIds := make([][][]byte, len(proofLogs))
	clients := make([]*client.Client, len(proofLogs))
	keys := make([][32]byte, len(proofLogs))
	var proofUpdatesChan := make(chan receivedProofUpdate, 100)

	onProofUpdate := func(proofUpdate []byte, client *client.Client) {
		proofUpdatesChan <- receivedProofUpdate{forClient: client, proofUpdate:proofUpdate}
	}

	for i := 0; i < len(proofLogs); i++ {
		keys[i] := [32]byte{}
		rand.Read(keys[i][:])
		clients[i], _ = client.NewClientWithConnection(keys[i], newDummyClient(srv))
		clients[i].OnProofUpdate = onProofUpdate
	}

	for i, pl := range proofLogs {
		logIds[i] = make([][]byte, pl)
		for j := 0; j < pl; j++ {
			statement := make([]byte, 32)
			rand.Read(statement)
			logIds[i][j] = clients[i].StartLog(statement)
		}
	}
	
	srv.Commit()
	
	
}
