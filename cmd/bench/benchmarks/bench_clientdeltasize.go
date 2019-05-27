package benchmarks

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"os"
	"sync"
	"time"

	"github.com/gonum/stat"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/server"
)

const (
	CLIENTDELTASIZE_TOTALLOGS = 100000
	CLIENTDELTASIZE_SAMPLES   = 25
)

// RunClientDeltaSizeBench will create 1M logs and then
// measure the size of a proof update (Delta) for a client
// maintaining a single log that does not change
func RunClientDeltaSizeBench() {
	srv, err := server.NewServer("", 0)
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Output a TEX graph
	graph, _ := os.Create("graph_clientdeltasize.tex")
	graph.Write([]byte("\\begin{figure}\n\t\\begin{tikzpicture}\n\t\\begin{axis}[\n"))
	graph.Write([]byte("\t\txlabel=Number of updated logs out of $10^7$,\n\t\tylabel=Delta update size (bytes)]\n"))
	graph.Write([]byte("\n\t\t\\addplot[color=red,mark=x,error bars/.cd,x dir=both,x explicit] coordinates {\n"))
	graph.Write([]byte("\t\t\t(0,0)\n"))
	defer graph.Close()

	// Create dummy logs, which returns the logIDs
	logIds := makeDummyLogs(srv, CLIENTDELTASIZE_TOTALLOGS)
	srv.Commit()

	// Create client
	key := [32]byte{}
	rand.Read(key[:])
	c, _ := client.NewClientWithConnection(key[:], newDummyClient(srv))
	statement := make([]byte, 32)
	rand.Read(statement)
	c.StartLog(statement)
	srv.Commit()

	var wg sync.WaitGroup
	updateSizes := []float64{}
	updateSizesLock := sync.Mutex{}

	c.OnProofUpdate = func(proofUpdate []byte, client *client.Client) {
		logging.Debugf("Got proof update len %d", len(proofUpdate))
		updateSizesLock.Lock()
		updateSizes = append(updateSizes, float64(len(proofUpdate)))
		updateSizesLock.Unlock()

		wg.Done()
	}

	c.SubscribeProofUpdates()

	logIdIdx := make([]uint64, CLIENTDELTASIZE_TOTALLOGS)
	logIdxsToChange := make([]int, CLIENTDELTASIZE_TOTALLOGS)
	for i := range logIdxsToChange {
		logIdIdx[i] = 0
		logIdxsToChange[i] = i
	}

	logId := [32]byte{}

	for numChangeLogs := CLIENTDELTASIZE_TOTALLOGS / 20; numChangeLogs <= CLIENTDELTASIZE_TOTALLOGS; numChangeLogs += CLIENTDELTASIZE_TOTALLOGS / 20 {
		for i := 0; i < CLIENTDELTASIZE_SAMPLES; i++ {
			mathrand.Seed(time.Now().UnixNano())
			mathrand.Shuffle(len(logIdxsToChange), func(i, j int) { logIdxsToChange[i], logIdxsToChange[j] = logIdxsToChange[j], logIdxsToChange[i] })

			logging.Debugf("Measuring delta size with [%d/%d] updates, sample [%d/%d]", numChangeLogs, CLIENTDELTASIZE_TOTALLOGS, i+1, CLIENTDELTASIZE_SAMPLES)
			// pick a changing random subset of logs every run
			var subset = logIdxsToChange[:]
			if CLIENTDELTASIZE_TOTALLOGS > numChangeLogs {
				startIdx := mathrand.Intn(CLIENTDELTASIZE_TOTALLOGS - numChangeLogs)
				subset = logIdxsToChange[startIdx : startIdx+numChangeLogs]
			}

			statement := make([]byte, 32)
			rand.Read(statement)
			for j := 0; j < numChangeLogs; j++ {
				logIdIdx[subset[j]]++
				copy(logId[:], logIds[subset[j]*32:subset[j]*32+32])

				err = srv.RegisterLogStatement(logId, logIdIdx[subset[j]], statement)
				if err != nil {
					panic(err)
				}
			}

			wg.Add(1)
			srv.Commit()
			wg.Wait()
		}

		// We now know the average size of the proof updates for this number of updates.

		graph.Write([]byte(fmt.Sprintf("\t\t\t(%d,%f) +- (%f,0)\n", numChangeLogs, average(updateSizes), stat.StdDev(updateSizes, nil))))
		logging.Debugf("RAW Update Sizes: %v", updateSizes)
		updateSizes = []float64{}
	}

	// Write end markers to the tex file and we're done.
	graph.Write([]byte("\t\t};"))
	graph.Write([]byte("\n\t\t\\end{axis}\n\\end{tikzpicture}"))
	graph.Write([]byte("\t\\caption{Client delta size for 1 log}\n"))
	graph.Write([]byte("\t\\label{graph_clientdeltasize}\n"))
	graph.Write([]byte("\\end{figure}\n"))

	logging.Debugf("Done")
}
