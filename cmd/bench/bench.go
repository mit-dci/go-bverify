package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/server"
)

func main() {

	runProofSizeBench(100000, 10000000)

	return
}

// runProofSizeBench will add 10k logs each run, and
// report the average proof size for 1, 10, 100, 1000 logs
func runProofSizeBench(increments, maxTotalLogs int) {
	// These are the different # logs we sample proof sizes for.
	// We output a graph per number of logs
	proofLogs := []int{1, 10, 100, 1000}

	srv, err := server.NewServer("")
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Output a TEX graph
	graphs := make([]*os.File, len(proofLogs))
	for i, pl := range proofLogs {
		graphs[i], _ = os.Create(fmt.Sprintf("graph_proofsize_%d.tex", pl))
		graphs[i].Write([]byte("\\begin{tikzpicture}\n\t\\begin{axis}[\n"))
		graphs[i].Write([]byte(fmt.Sprintf("\t\txlabel=Number of server logs,\n\t\tylabel=Proof size for %d logs (bytes)]\n", pl)))
		graphs[i].Write([]byte("\n\t\t\\addplot[color=red,mark=x] coordinates {\n"))
		defer graphs[i].Close()
	}

	// Store the log IDs into one big byteslice
	logIds := make([]byte, 32*maxTotalLogs)

	// We need total / increments number of runs
	runs := maxTotalLogs / increments

	for iRun := 0; iRun < runs; iRun++ {
		fmt.Printf("\rProof Size Run [%d/%d] (%.2f %%)", iRun+1, runs, float64(iRun+1)/float64(runs)*float64(100))

		var wg sync.WaitGroup
		// Since we're not actually verifying the statements, we can just
		// use random pubkeys, logIDs and witnesses
		pub33 := [33]byte{}
		_, err := rand.Read(pub33[:])
		if err != nil {
			panic(err)
		}

		for iLog := 0; iLog < increments; iLog++ {
			wg.Add(1)
			go func(idx int) {

				// Read a random witness and log ID
				witness := make([]byte, 32)
				logId := [32]byte{}
				rand.Read(logId[:])
				rand.Read(witness[:])

				// Create the log and write the first statement
				srv.RegisterLogID(logId, pub33)
				srv.RegisterLogStatement(logId, 0, witness[:])

				// startIdx determines the start position of the LogID in the
				// large byteslice we use to cache them
				startIdx := (iRun * increments * 32) + iLog*32
				// cache the generated LogID into the big array
				copy(logIds[startIdx:], logId[:])

				witness = nil

				wg.Done()
			}(iLog)
		}
		// Wait for all logs to be finished in the goroutines
		wg.Wait()

		// Now make the server commit the tree
		err = srv.Commit()
		if err != nil {
			panic(err)
		}

		// Make an arrays for keeping the total size of received
		// proofs, one element per proofLogs element
		receivedProofs := make([]int64, len(proofLogs))
		receivedProofSizes := make([]int64, len(proofLogs))
		rand.Seed(time.Now().UnixNano())

		// This determines the range in which we can look for LogIDs
		// in the cache array
		maxLogId := ((iRun + 1) * increments)

		// Get proof sizes for all requested # logs by getting all possible
		// samples from the full tree
		for i := 0; i < maxLogId-1000; i++ {
			wg.Add(1)
			go func(pIdx int) {
				// Take random logIDs from the known logIDs the size of the
				// desired number of proofs
				logIdSets := make([][][]byte, len(proofLogs))
				maxProofLogs := 0
				for idx, pl := range proofLogs {
					logIdSets[idx] = make([][]byte, pl)
					if pl > maxProofLogs {
						maxProofLogs = pl
					}
				}

				// Fill the LogID sets with random logIDs
				for logId := 0; logId < maxProofLogs; logId++ {
					for idx, pl := range proofLogs {
						if logId < pl {
							offset := (pIdx + logId) * 32
							logIdSets[idx][logId] = logIds[offset : offset+32]
						}
					}
				}

				// Get the proof for the keys in each of the sets and then
				// calculate their size
				var wg2 sync.WaitGroup
				for i := range proofLogs {
					wg2.Add(1)
					go func(idx int) {
						partialMPT, _ := srv.GetProofForKeys(logIdSets[idx])
						atomic.AddInt64(&(receivedProofs[idx]), int64(1))
						atomic.AddInt64(&(receivedProofSizes[idx]), int64(partialMPT.ByteSize()))
						wg2.Done()
					}(i)
				}

				wg2.Wait()
				logIdSets = nil
				wg.Done()
			}(i)
		}

		wg.Wait()

		// Write the average sampled sizes to the TEX files
		for idx := range proofLogs {
			graphs[idx].Write([]byte(fmt.Sprintf("\t\t\t(%d,%d)\n", iRun*increments, atomic.LoadInt64(&receivedProofSizes[idx])/atomic.LoadInt64(&receivedProofs[idx]))))
		}
	}

	// Write end markers to the tex files and we're done.
	for idx := range proofLogs {
		graphs[idx].Write([]byte("\t\t};"))
		graphs[idx].Write([]byte("\n\t\t\\end{axis}\n\\end{tikzpicture}"))
	}

}
