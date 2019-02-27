package benchmarks

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"

	//"github.com/mit-dci/go-bverify/mpt"

	"github.com/mit-dci/go-bverify/client"

	"github.com/mit-dci/go-bverify/server"
)

const (
	CLIENTUPDATE_TOTALLOGS     = 100000
	CLIENTUPDATE_SAMPLES       = 100
	CLIENTUPDATE_NUMCHANGELOGS = 1000 // The number of logs that are changed with each commitment
)

type receivedProofUpdate struct {
	forClient   *client.Client
	proofUpdate []byte
}

func newDummyClient(srv *server.Server) net.Conn {
	s, c := net.Pipe()
	p := server.NewLogProcessor(s, srv)
	go p.Process()
	return c
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
	clientLogIds := make([][][32]byte, len(proofLogs))
	clients := make([]*client.Client, len(proofLogs))
	keys := make([][32]byte, len(proofLogs))
	proofUpdatesChan := make(chan receivedProofUpdate, len(proofLogs)*(CLIENTUPDATE_SAMPLES+1))

	//partialMpts := make([]mpt.PartialMPT, len(proofLogs))

	go func() {
		for {
			pu := <-proofUpdatesChan
			clientIdx := -1
			for i := 0; i < len(clients); i++ {
				if clients[i] == pu.forClient {
					clientIdx = i
					break
				}
			}
			if clientIdx == -1 {
				continue
			}

		}
	}()

	onProofUpdate := func(proofUpdate []byte, client *client.Client) {
		proofUpdatesChan <- receivedProofUpdate{forClient: client, proofUpdate: proofUpdate}
	}

	onError := func(err error, client *client.Client) {
		fmt.Printf("Error occured: %s", err.Error())
	}

	for i := 0; i < len(proofLogs); i++ {
		keys[i] = [32]byte{}
		rand.Read(keys[i][:])
		clients[i], _ = client.NewClientWithConnection(keys[i][:], newDummyClient(srv))
		clients[i].OnProofUpdate = onProofUpdate
		clients[i].OnError = onError
		clients[i].SubscribeProofUpdates()
	}

	for i, pl := range proofLogs {
		clientLogIds[i] = make([][32]byte, pl)
		for j := 0; j < pl; j++ {
			statement := make([]byte, 32)
			rand.Read(statement)
			clientLogIds[i][j], _ = clients[i].StartLog(statement)
		}
	}

	logIdsToChange := make([][32]byte, CLIENTUPDATE_NUMCHANGELOGS)
	for j := 0; j < CLIENTUPDATE_NUMCHANGELOGS; j++ {
		statement := make([]byte, 4)
		rand.Read(statement)
		logIdx := (int(binary.BigEndian.Uint16(statement[0:4])) % len(logIds)) * 32
		copy(logIdsToChange[j][:], logIds[logIdx:logIdx+32])
	}

	for i := 0; i < CLIENTUPDATE_SAMPLES; i++ {
		for _, logId := range logIdsToChange {
			statement := make([]byte, 32)
			rand.Read(statement)
			srv.RegisterLogStatement(logId, uint64(i), statement)
		}
		fmt.Printf("\rCommitting [%d/%d]", i, CLIENTUPDATE_SAMPLES)
		srv.Commit()
	}

}
