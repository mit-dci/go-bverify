package benchmarks

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"

	"github.com/mit-dci/go-bverify/server"
)

const (
	PROOFSIZEPERLOG_TOTALLOGS   = 10000000
	PROOFSIZEPERLOG_INCREMENTS  = 10
	PROOFSIZEPERLOG_MAXLOGCOUNT = 10000
	PROOFSIZEPERLOG_SAMPLELIMIT = 1000
)

// RunProofSizePerLogBench will add 10M logs and
// report the average proof size per log for tracking
// PROOFSIZEPERLOG_INCREMENTS up to PROOFSIZEPERLOG_MAXLOGCOUNT
// logs in total (in steps of PROOFSIZEPERLOG_INCREMENTS)
func RunProofSizePerLogBench() {
	fmt.Printf("\n\rRunning proof size per log benchmark")
	srv, err := server.NewServer("")
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Output a TEX graph
	graph, _ := os.Create("graph_proofsizeperlog.tex")
	graph.Write([]byte("\\begin{tikzpicture}\n\t\\begin{axis}[\n"))
	graph.Write([]byte("\t\txlabel=Logs included in proof,\n\t\tylabel=Proof size per log (bytes)]\n"))
	graph.Write([]byte("\n\t\t\\addplot[color=red,mark=x] coordinates {\n"))
	graph.Write([]byte("\t\t\t(0,0)\n"))
	defer graph.Close()

	// Store the log IDs into one big byteslice
	logIds := make([]byte, 32*PROOFSIZEPERLOG_TOTALLOGS)

	var wg sync.WaitGroup
	// Since we're not actually verifying the statements, we can just
	// use random pubkeys, logIDs and witnesses
	pub33 := [33]byte{}
	_, err = rand.Read(pub33[:])
	if err != nil {
		panic(err)
	}

	fmt.Printf("\rRunning proof size per log benchmark: [Adding %d logs...]", PROOFSIZEPERLOG_TOTALLOGS)

	for logIdx := 0; logIdx < PROOFSIZEPERLOG_TOTALLOGS; logIdx++ {
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
			startIdx := idx * 32
			// cache the generated LogID into the big array
			copy(logIds[startIdx:], logId[:])

			witness = nil

			wg.Done()
		}(logIdx)
	}
	// Wait for all logs to be finished in the goroutines
	wg.Wait()

	fmt.Printf("\rRunning proof size per log benchmark: [Committing the log]                  ")

	// Now make the server commit the tree
	err = srv.Commit()
	if err != nil {
		panic(err)
	}

	fmt.Printf("\rRunning proof size per log benchmark: [Committed the log, generating proofs]                  ")

	// Make an arrays for keeping the total size of received
	// proofs, one element per proofLogs element
	receivedProofs := make([]int64, PROOFSIZEPERLOG_MAXLOGCOUNT/PROOFSIZEPERLOG_INCREMENTS)
	receivedProofSizes := make([]int64, PROOFSIZEPERLOG_MAXLOGCOUNT/PROOFSIZEPERLOG_INCREMENTS)

	wg = sync.WaitGroup{}
	// Get proof sizes for all requested # logs by getting all possible
	// samples from the full tree
	for i := 0; i < PROOFSIZEPERLOG_TOTALLOGS; i++ {
		wg.Add(1)
		go func(pIdx int) {
			// Take random logIDs from the known logIDs the size of the
			// desired number of proofs
			logIdSets := make([][][]byte, len(receivedProofs))
			maxProofLogs := 0
			for idx := range receivedProofs {
				pl := ((idx + 1) * PROOFSIZEPERLOG_INCREMENTS)
				numLogs := atomic.LoadInt64(&receivedProofs[idx])
				// Only try the slice if it still fits within the bounds of our logs collection
				// Don't use overlapping slices and cap it on ~PROOFSIZEPERLOG_SAMPLELIMIT samples.
				// Because this function is executed in parallel there will be _at least_ PROOFSIZEPERLOG_SAMPLELIMIT
				// samples per run. Could be slightly more
				if pIdx+pl < PROOFSIZEPERLOG_TOTALLOGS && (pIdx%pl == 0) && numLogs < PROOFSIZEPERLOG_SAMPLELIMIT {
					logIdSets[idx] = make([][]byte, pl)
					if pl > maxProofLogs {
						maxProofLogs = pl
					}
				}
			}

			// Fill the LogID sets with random logIDs
			for logId := 0; logId < PROOFSIZEPERLOG_MAXLOGCOUNT; logId++ {
				for idx := range receivedProofs {
					pl := ((idx + 1) * PROOFSIZEPERLOG_INCREMENTS)
					if logId < pl && len(logIdSets[idx]) > 0 {
						offset := (pIdx + logId) * 32
						logIdSets[idx][logId] = logIds[offset : offset+32]
					}
				}
			}

			// Get the proof for the keys in each of the sets and then
			// calculate their size
			var wg2 sync.WaitGroup
			for ipL := range receivedProofs {
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

	// Write the average sampled sizes to the TEX file
	for idx := range receivedProofs {
		avgProofSize := atomic.LoadInt64(&receivedProofSizes[idx]) / atomic.LoadInt64(&receivedProofs[idx])
		avgProofSizePerLog := avgProofSize / int64((idx+1)*PROOFSIZEPERLOG_INCREMENTS)
		graph.Write([]byte(fmt.Sprintf("\t\t\t(%d,%d)\n", (idx+1)*PROOFSIZEPERLOG_INCREMENTS, avgProofSizePerLog)))
	}

	// Write end markers to the tex file and we're done.
	graph.Write([]byte("\t\t};"))
	graph.Write([]byte("\n\t\t\\end{axis}\n\\end{tikzpicture}"))

}
