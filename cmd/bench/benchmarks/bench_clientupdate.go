package benchmarks

import (
	"crypto/rand"
	"fmt"
	"sync"

	mathrand "math/rand"
	"net"
	"os"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/server"
)

const (
	CLIENTUPDATE_TOTALLOGS            = 10000000
	CLIENTUPDATE_UPDATESIZE_INCREMENT = 200000
	CLIENTUPDATE_SAMPLES              = 5
)

type receivedProofUpdate struct {
	forClient   *client.Client
	proofUpdate []byte
}

type clientUpdateBenchResult struct {
	trackingLogCount    int
	averageUpdateSizeKB float64
	averageUpdateTimeMS float64
}

type clientUpdateBenchResultCollection struct {
	numberOfChangingLogs int
	result               []clientUpdateBenchResult
}

func newDummyClientAndProcessor(srv *server.Server) (net.Conn, server.LogProcessor) {
	s, c := net.Pipe()
	p := server.NewLogProcessor(s, srv)
	go p.Process()
	return c, p
}

func newDummyClient(srv *server.Server) net.Conn {
	c, _ := newDummyClientAndProcessor(srv)
	return c
}

func RunClientUpdateBench() {
	// Create server
	srv, err := server.NewServer("", 0)
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false
	srv.KeepCommitmentTree = false

	// Create client to receive proof updates
	var key [32]byte
	rand.Read(key[:])
	clt, _ := client.NewClientWithConnection(key[:], newDummyClient(srv))
	clt.OnError = func(err error, c *client.Client) {
		panic(err)
	}
	err = clt.SubscribeProofUpdates()
	if err != nil {
		panic(err)
	}

	// Create dummy logs, which returns the logIDs
	logIds := makeDummyLogs(srv, CLIENTUPDATE_TOTALLOGS-1) // Our clients log will be the final one.
	logging.Debugf("Dummy logs created, committing the tree...")
	err = srv.Commit()
	if err != nil {
		panic(err)
	}
	logging.Debugf("Commit done.")

	statement := make([]byte, 32)
	rand.Read(statement)
	clt.StartLog(statement)

	logging.Debugf("Shuffling dummy logs...")

	logIdIdx := make([]uint64, CLIENTUPDATE_TOTALLOGS-1)
	logIdxsToChange := make([]int, CLIENTUPDATE_TOTALLOGS-1)
	for i := range logIdxsToChange {
		logIdIdx[i] = 0
		logIdxsToChange[i] = i
	}
	mathrand.Seed(time.Now().UnixNano())
	mathrand.Shuffle(len(logIdxsToChange), func(i, j int) { logIdxsToChange[i], logIdxsToChange[j] = logIdxsToChange[j], logIdxsToChange[i] })

	var subset = logIdxsToChange[:]
	logId := [32]byte{}

	updateSizes := make([]int64, CLIENTUPDATE_TOTALLOGS/CLIENTUPDATE_UPDATESIZE_INCREMENT)

	var wgProofUpdates sync.WaitGroup

	numsChangeLogs := []int{10, 100, 1000, 10000, 100000}
	for i := 200000; i < CLIENTUPDATE_TOTALLOGS; i += CLIENTUPDATE_UPDATESIZE_INCREMENT {
		numsChangeLogs = append(numsChangeLogs, i)
	}

	for loop, numChangeLogs := range numsChangeLogs {
		for i := 0; i < CLIENTUPDATE_SAMPLES; i++ {
			// pick a changing random subset of logs every run
			if CLIENTUPDATE_TOTALLOGS > numChangeLogs {
				startIdx := mathrand.Intn(CLIENTUPDATE_TOTALLOGS - numChangeLogs)
				subset = logIdxsToChange[startIdx : startIdx+numChangeLogs]
			}

			for j := 0; j < numChangeLogs; j++ {
				logIdIdx[subset[j]]++
				copy(logId[:], logIds[subset[j]*32:subset[j]*32+32])
				statement := make([]byte, 32)
				rand.Read(statement)

				err = srv.RegisterLogStatement(logId, logIdIdx[subset[j]], statement)
				if err != nil {
					panic(err)
				}
			}
			logging.Debugf("[%d/%d] [%d/%d] Committing server [%d]", loop, CLIENTUPDATE_TOTALLOGS/CLIENTUPDATE_UPDATESIZE_INCREMENT, i+1, CLIENTUPDATE_SAMPLES, len(logIdxsToChange))
			wgProofUpdates.Add(1)
			clt.OnProofUpdate = func(proofUpdate []byte, client *client.Client) {
				atomic.AddInt64(&updateSizes[loop], int64(len(proofUpdate)))
				wgProofUpdates.Done()
			}
			err := srv.Commit()
			if err != nil {
				panic(err)
			}
			wgProofUpdates.Wait()
		}
	}

	graph, _ := os.Create("graph_clientupdate_size.tex")
	graph.Write([]byte("\\begin{figure}\n\t\\begin{tikzpicture}\n\t\t\\begin{axis}[\n"))
	graph.Write([]byte("\t\t\txlabel=Number of changed logs out of $10^7$,\n\t\tylabel=Size of proof updates per day for 1 log (in KB)]\n"))
	graph.Write([]byte("\n\t\t\t\\addplot[color=red,mark=x] coordinates {\n"))
	for i, us := range updateSizes {
		averageUpdateSizeKBDay := float64(us) / float64(CLIENTUPDATE_SAMPLES) / float64(1024) * float64(144)
		graph.Write([]byte(fmt.Sprintf("\t\t\t\t(%d,%.4f)\n", i*CLIENTUPDATE_UPDATESIZE_INCREMENT, averageUpdateSizeKBDay)))
	}
	graph.Write([]byte("\t\t};"))
	graph.Write([]byte("\n\t\t\\end{axis}\n\t\\end{tikzpicture}\n"))
	graph.Write([]byte("\t\\caption{Proof size per day}\n"))
	graph.Write([]byte("\t\\label{graph_clientupdate}\n"))
	graph.Write([]byte("\\end{figure}\n"))
	graph.Close()

	logging.Debugf("Done")

	srv.Stop()

}
