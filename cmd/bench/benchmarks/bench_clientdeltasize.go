package benchmarks

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/server"
)

const (
	CLIENTDELTASIZE_TOTALLOGS       = 1000000
	CLIENTDELTASIZE_CHANGEINCREMENT = 1000
	CLIENTDELTASIZE_SAMPLES         = 20
)

// RunClientDeltaSizeBench will create 1M logs and then
// measure the size of a proof update (Delta) for a client
// maintaining a single log that does not change
func RunClientDeltaSizeBench() {
	srv, err := server.NewServer("")
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Output a TEX graph
	graph, _ := os.Create("graph_clientdeltasize.tex")
	graph.Write([]byte("\\begin{figure}\n\t\\begin{tikzpicture}\n\t\\begin{axis}[\n"))
	graph.Write([]byte("\t\txlabel=Number of updated logs out of 1M,\n\t\tylabel=Delta update size (bytes)]\n"))
	graph.Write([]byte("\n\t\t\\addplot[color=red,mark=x] coordinates {\n"))
	graph.Write([]byte("\t\t\t(0,0)\n"))
	defer graph.Close()

	// Create dummy logs, which returns the logIDs
	logIds := makeDummyLogs(srv, CLIENTDELTASIZE_TOTALLOGS)
	srv.Commit()
	fmt.Printf("\n")

	// Create client
	key := [32]byte{}
	rand.Read(key[:])
	c, _ := client.NewClientWithConnection(key[:], newDummyClient(srv))
	statement := make([]byte, 32)
	rand.Read(statement)
	c.StartLog(statement)
	srv.Commit()

	var wg sync.WaitGroup
	var updates, updateSizes int64

	c.OnProofUpdate = func(proofUpdate []byte, client *client.Client) {
		atomic.AddInt64(&updates, 1)
		atomic.AddInt64(&updateSizes, int64(len(proofUpdate)))

		wg.Done()
	}

	c.SubscribeProofUpdates()

	logIdIdx := make([]uint64, CLIENTDELTASIZE_TOTALLOGS)
	logIdxsToChange := make([]int, CLIENTDELTASIZE_TOTALLOGS)
	for i := range logIdxsToChange {
		logIdIdx[i] = 0
		logIdxsToChange[i] = i
	}
	mathrand.Seed(time.Now().UnixNano())
	mathrand.Shuffle(len(logIdxsToChange), func(i, j int) { logIdxsToChange[i], logIdxsToChange[j] = logIdxsToChange[j], logIdxsToChange[i] })

	var subset = logIdxsToChange[:]
	logId := [32]byte{}
	for numChangeLogs := CLIENTDELTASIZE_CHANGEINCREMENT; numChangeLogs < CLIENTDELTASIZE_TOTALLOGS; numChangeLogs += CLIENTDELTASIZE_CHANGEINCREMENT {
		for i := 0; i < CLIENTUPDATE_SAMPLES; i++ {
			// pick a changing random subset of logs every run
			if CLIENTDELTASIZE_TOTALLOGS > numChangeLogs {
				startIdx := mathrand.Intn(CLIENTDELTASIZE_TOTALLOGS - numChangeLogs)
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
			fmt.Printf("\r[%d/%d] [%d/%d] Committing server [%d]                    ", numChangeLogs, CLIENTDELTASIZE_TOTALLOGS, CLIENTUPDATE_SAMPLES, len(logIdxsToChange))

			wg.Add(1)
			srv.Commit()
		}
		wg.Wait()

		// We now know the average size of the proof updates for this number of updates.
		graph.Write([]byte(fmt.Sprintf("\t\t\t(%d,%d)\n", numChangeLogs, updateSizes/updates)))

		updates, updateSizes = 0, 0
	}

	// Write end markers to the tex file and we're done.
	graph.Write([]byte("\t\t};"))
	graph.Write([]byte("\n\t\t\\end{axis}\n\\end{tikzpicture}"))
	graph.Write([]byte("\t\\caption{Client delta size for 1 log}\n"))
	graph.Write([]byte("\t\\label{graph_clientdeltasize}\n"))
	graph.Write([]byte("\\end{figure}\n"))

	fmt.Printf("\nDone\n")
}
