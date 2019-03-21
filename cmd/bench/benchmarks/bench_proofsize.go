package benchmarks

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/server"
)

const (
	PROOFSIZE_TOTALLOGS   = 10000000
	PROOFSIZE_INCREMENTS  = 100000
	PROOFSIZE_SAMPLELIMIT = 1000
)

// runProofSizeBench will add 10k logs each run, and
// report the average proof size for 1, 10, 100, 1000 logs
func RunProofSizeBench() {
	// These are the different # logs we sample proof sizes for.
	// We output a graph per number of logs
	proofLogs := [4]int{1, 10, 100, 1000}

	srv, err := server.NewServer("", 0)
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Output a TEX graph
	graphs := make([]*os.File, len(proofLogs))
	for i, pl := range proofLogs {
		graphs[i], _ = os.Create(fmt.Sprintf("graph_proofsize_%d.tex", pl))
		graphs[i].Write([]byte("\\begin{figure}\n\t\\begin{tikzpicture}\n\t\t\\begin{axis}[\n"))
		graphs[i].Write([]byte(fmt.Sprintf("\t\t\txlabel=Number of server logs,\n\t\tylabel=Proof size for %d logs (bytes)]\n", pl)))
		graphs[i].Write([]byte("\n\t\t\t\\addplot[color=red,mark=x] coordinates {\n"))
		graphs[i].Write([]byte("\t\t\t\t(0,0)\n"))
		defer graphs[i].Close()
	}

	// Store the log IDs into one big byteslice
	logIds := make([]byte, 32*PROOFSIZE_TOTALLOGS)

	// We need total / increments number of runs
	runCount := PROOFSIZE_TOTALLOGS / PROOFSIZE_INCREMENTS

	for runIdx := 0; runIdx < runCount; runIdx++ {
		fmt.Printf("\rProof Size Run [%d/%d] (%.2f %%) - Tree size: %d bytes", runIdx+1, runCount, float64(runIdx+1)/float64(runCount)*float64(100), srv.TreeSize())

		var wg sync.WaitGroup
		// Since we're not actually verifying the statements, we can just
		// use random pubkeys, logIDs and witnesses
		pub33 := [33]byte{}
		_, err := rand.Read(pub33[:])
		if err != nil {
			panic(err)
		}

		for logIdx := 0; logIdx < PROOFSIZE_INCREMENTS; logIdx++ {
			wg.Add(1)
			go func(run, idx int) {

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
				startIdx := (run * PROOFSIZE_INCREMENTS * 32) + idx*32
				// cache the generated LogID into the big array
				copy(logIds[startIdx:], logId[:])

				witness = nil

				wg.Done()
			}(runIdx, logIdx)
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
		maxLogId := ((runIdx + 1) * PROOFSIZE_INCREMENTS)

		wg = sync.WaitGroup{}
		// Get proof sizes for all requested # logs by getting all possible
		// samples from the full tree
		for i := 0; i < maxLogId; i++ {
			wg.Add(1)
			go func(pIdx int) {
				// Take random logIDs from the known logIDs the size of the
				// desired number of proofs
				logIdSets := make([][][]byte, len(proofLogs))
				maxProofLogs := 0
				for idx, pl := range proofLogs {
					numLogs := atomic.LoadInt64(&receivedProofs[idx])
					// Only try the slice if it still fits within the bounds of our logs collection
					// Don't use overlapping slices and cap it on ~PROOFSIZE_SAMPLELIMIT samples.
					// Because this function is executed in parallel there will be _at least_ PROOFSIZE_SAMPLELIMIT
					// samples per run. Could be slightly more
					if pIdx+pl < maxLogId && (pIdx%pl == 0) && numLogs < PROOFSIZE_SAMPLELIMIT {
						logIdSets[idx] = make([][]byte, pl)
						if pl > maxProofLogs {
							maxProofLogs = pl
						}
					}
				}

				// Fill the LogID sets with random logIDs
				for logId := 0; logId < maxProofLogs; logId++ {
					for idx, pl := range proofLogs {

						if logId < pl && len(logIdSets[idx]) > 0 {
							offset := (pIdx + logId) * 32
							logIdSets[idx][logId] = logIds[offset : offset+32]
						}
					}
				}

				// Get the proof for the keys in each of the sets and then
				// calculate their size
				var wg2 sync.WaitGroup
				for ipL := range proofLogs {
					if len(logIdSets[ipL]) > 0 {
						wg2.Add(1)
						go func(idx int) {
							partialMPT, _ := srv.GetProofForKeys(logIdSets[idx])
							atomic.AddInt64(&(receivedProofs[idx]), int64(1))
							atomic.AddInt64(&(receivedProofSizes[idx]), int64(partialMPT.ByteSize()))
							wg2.Done()
						}(ipL)
					}
				}

				wg2.Wait()
				logIdSets = nil
				wg.Done()
			}(i)
		}

		wg.Wait()

		// Write the average sampled sizes to the TEX files
		for idx := range proofLogs {
			numLogs := atomic.LoadInt64(&receivedProofs[idx])
			// Only write the graph point if we took more than 0 samples
			if numLogs > 0 {
				graphs[idx].Write([]byte(fmt.Sprintf("\t\t\t\t(%d,%d)\n", (runIdx+1)*PROOFSIZE_INCREMENTS, atomic.LoadInt64(&receivedProofSizes[idx])/numLogs)))
			}
		}
	}

	// Write end markers to the tex files and we're done.
	for idx, pl := range proofLogs {
		var s = "s"
		if pl == 1 {
			s = ""
		}
		graphs[idx].Write([]byte("\t\t};"))
		graphs[idx].Write([]byte("\n\t\t\\end{axis}\n\t\\end{tikzpicture}\n"))
		graphs[idx].Write([]byte(fmt.Sprintf("\t\\caption{Proof size for %d log%s}\n", pl, s)))
		graphs[idx].Write([]byte(fmt.Sprintf("\t\\label{graph_proofsize_%d}\n", pl)))
		graphs[idx].Write([]byte("\\end{figure}\n"))
	}
}
