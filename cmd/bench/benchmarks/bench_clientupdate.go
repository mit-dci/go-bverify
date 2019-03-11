package benchmarks

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/mpt"
	"github.com/mit-dci/go-bverify/server"
)

const (
	CLIENTUPDATE_TOTALLOGS = 1000000
	CLIENTUPDATE_SAMPLES   = 20
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
	// We run the same benchmark for various numbers of updated logs for
	// each sample. That way we can make multiple tables: one showing a fixed number
	// of updates and a varying number of logs being tracked
	// the other shows a fixed number of logs tracked and a varying number
	// of updates.
	trackingLogSizes := []int{1, 10, 100, 1000}
	runUpdateSizes := []int{1000, 10000, 100000, 1000000}

	results := make([]clientUpdateBenchResultCollection, len(runUpdateSizes))
	for i, rus := range runUpdateSizes {
		fmt.Printf("Running client update bench for %d updating logs...\n", rus)
		results[i] = clientUpdateBenchResultCollection{numberOfChangingLogs: rus, result: runClientUpdateBench(CLIENTUPDATE_TOTALLOGS, rus, trackingLogSizes)}
	}

	for _, rr := range results {
		table, _ := os.Create(fmt.Sprintf("table_clientupdate_size_and_time_%d_updates.tex", rr.numberOfChangingLogs))
		table.Write([]byte("\\begin{table*}[t]\n"))
		table.Write([]byte("\\centering\n"))
		table.Write([]byte("\\begin{tabular}{ |c||c|c| }\n"))
		table.Write([]byte(" \\hline\n"))
		table.Write([]byte(" Number of logs & Data per update & Time per update \\\\\n"))
		table.Write([]byte(" \\hline \\hline\n"))

		for _, r := range rr.result {
			table.Write([]byte(fmt.Sprintf("  %d & %.2f KB & %.2f ms \\\\\n", r.trackingLogCount, r.averageUpdateSizeKB, r.averageUpdateTimeMS)))
		}

		table.Write([]byte("\\hline\n"))
		table.Write([]byte("\\end{tabular}\n"))
		table.Write([]byte(fmt.Sprintf("\\caption{The average size and processing time of one update for various number of logs tracked (over %d samples) when the \\sys server is maintaining $%d$ logs and updating a random selection of $%d$ logs. The measured clients do not update their log, to show the cost of being idle.}\n", CLIENTUPDATE_SAMPLES, CLIENTUPDATE_TOTALLOGS, rr.numberOfChangingLogs)))
		table.Write([]byte(fmt.Sprintf("\\label{table:clientupdate_size_and_time_%d_updates}\n", rr.numberOfChangingLogs)))
		table.Write([]byte("\\end{table*}	\n"))
		table.Close()
	}

	for _, pl := range trackingLogSizes {
		table, _ := os.Create(fmt.Sprintf("table_clientupdate_size_and_time_%d_logs.tex", pl))
		table.Write([]byte("\\begin{table*}[t]\n"))
		table.Write([]byte("\\centering\n"))
		table.Write([]byte("\\begin{tabular}{ |c||c|c| }\n"))
		table.Write([]byte(" \\hline\n"))
		table.Write([]byte(" Number of updates & Data per update & Time per update \\\\\n"))
		table.Write([]byte(" \\hline \\hline\n"))
		for i, rus := range runUpdateSizes {
			for _, r := range results[i].result {
				if r.trackingLogCount == pl {
					table.Write([]byte(fmt.Sprintf("  %d & %.2f KB & %.2f ms \\\\\n", rus, r.averageUpdateSizeKB, r.averageUpdateTimeMS)))
				}
			}

		}
		table.Write([]byte("\\hline\n"))
		table.Write([]byte("\\end{tabular}\n"))

		s := "s"
		if pl == 1 {
			s = ""
		}
		logs := fmt.Sprintf("%d log%s", pl, s)

		table.Write([]byte(fmt.Sprintf("\\caption{The average size and processing time of one update for %s (over %d samples) when the \\sys server is maintaining $%d$ logs and updating an increasing selection of logs. The measured clients does not update its log, to show the cost of being idle.}\n", logs, CLIENTUPDATE_SAMPLES, CLIENTUPDATE_TOTALLOGS)))
		table.Write([]byte(fmt.Sprintf("\\label{table:clientupdate_size_and_time_%d_logs}\n", pl)))
		table.Write([]byte("\\end{table*}	\n"))
		table.Close()
	}

	for _, pl := range trackingLogSizes {
		table, _ := os.Create(fmt.Sprintf("table_clientupdate_sizes_%d_logs.tex", pl))
		table.Write([]byte("\\begin{table*}[t]\n"))
		table.Write([]byte("\\centering\n"))
		table.Write([]byte("\\begin{tabular}{ |c||c|c|c| }\n"))
		table.Write([]byte(" \\hline\n"))
		table.Write([]byte(" Number of updates & Data per day & Data per month & Data per year \\\\\n"))
		table.Write([]byte(" \\hline \\hline\n"))
		for i, rus := range runUpdateSizes {
			for _, r := range results[i].result {
				if r.trackingLogCount == pl {
					table.Write([]byte(fmt.Sprintf("  %d & %.2f KB & %.2f KB & %.2f MB \\\\\n", rus, r.averageUpdateSizeKB*144, r.averageUpdateSizeKB*144*365/12, r.averageUpdateSizeKB*144*365/1024)))
				}
			}

		}
		table.Write([]byte("\\hline\n"))
		table.Write([]byte("\\end{tabular}\n"))

		s := "s"
		if pl == 1 {
			s = ""
		}
		logs := fmt.Sprintf("%d log%s", pl, s)

		table.Write([]byte(fmt.Sprintf("\\caption{The average size of updates per day, month and year for %s (over %d samples) when the \\sys server is maintaining $%d$ logs and updating an increasing selection of logs. The measured client does not update its log, to show the cost of being idle.}\n", logs, CLIENTUPDATE_SAMPLES, CLIENTUPDATE_TOTALLOGS)))
		table.Write([]byte(fmt.Sprintf("\\label{table:clientupdate_sizes_%d_logs}\n", pl)))
		table.Write([]byte("\\end{table*}	\n"))
		table.Close()
	}

	for _, pl := range trackingLogSizes {
		table, _ := os.Create(fmt.Sprintf("table_clientupdate_times_%d_logs.tex", pl))
		table.Write([]byte("\\begin{table*}[t]\n"))
		table.Write([]byte("\\centering\n"))
		table.Write([]byte("\\begin{tabular}{ |c||c| }\n"))
		table.Write([]byte(" \\hline\n"))
		table.Write([]byte(" Number of updates & Time to process updates \\\\\n"))
		table.Write([]byte(" \\hline \\hline\n"))
		for i, rus := range runUpdateSizes {
			for _, r := range results[i].result {
				if r.trackingLogCount == pl {
					table.Write([]byte(fmt.Sprintf("  %d & %.2f ms \\\\\n", rus, r.averageUpdateTimeMS)))
				}
			}

		}
		table.Write([]byte("\\hline\n"))
		table.Write([]byte("\\end{tabular}\n"))

		s := "s"
		if pl == 1 {
			s = ""
		}
		logs := fmt.Sprintf("%d log%s", pl, s)

		table.Write([]byte(fmt.Sprintf("\\caption{The average time to process updates to the proofs for %s (over %d samples) when the \\sys server is maintaining $%d$ logs and updating an increasing selection of logs. The measured client does not update its log, to show the cost of being idle.}\n", logs, CLIENTUPDATE_SAMPLES, CLIENTUPDATE_TOTALLOGS)))
		table.Write([]byte(fmt.Sprintf("\\label{table:table_clientupdate_times_%d_logs}\n", pl)))
		table.Write([]byte("\\end{table*}	\n"))
		table.Close()
	}

}

// RunClientUpdateBench will create 1M logs and then
// measure the time and size of a proof update for clients
// whose logs have not been changed (while there were other updates)
func runClientUpdateBench(totalLogs, numChangeLogs int, proofLogs []int) []clientUpdateBenchResult {
	srv, err := server.NewServer("")
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Create clients
	clientLogIds := make([][][32]byte, len(proofLogs))
	clients := make([]*client.Client, len(proofLogs))
	keys := make([][32]byte, len(proofLogs))
	proofUpdatesChan := make(chan receivedProofUpdate, len(proofLogs)*(CLIENTUPDATE_SAMPLES+1))

	// Create dummy logs, which returns the logIDs
	logIds := makeDummyLogs(srv, totalLogs)
	srv.Commit()
	fmt.Printf("\n")
	results := make([]clientUpdateBenchResult, len(proofLogs))
	updates := make([]int64, len(proofLogs))
	updateSizes := make([]int64, len(proofLogs))
	updateTimes := make([]int64, len(proofLogs))
	partialMpts := make([]*mpt.PartialMPT, len(proofLogs))
	partialMptLocks := make([]sync.Mutex, len(proofLogs))

	var wg sync.WaitGroup
	wg.Add(runtime.NumCPU())
	for iThreads := 0; iThreads < runtime.NumCPU(); iThreads++ {
		go func() {
			for pu := range proofUpdatesChan {
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
				partialMptLocks[clientIdx].Lock()
				if partialMpts[clientIdx] == nil {
					// If we don't have a partial MPT for this client, then we
					// never received *any* proofs. A delta will  not help us as
					// it will not include the state of the tree prior to us joining
					// the log. So we need to fetch the PartialMPT first

					partialMpts[clientIdx], err = clients[clientIdx].RequestProof([][32]byte{})
					if err != nil {
						fmt.Printf("Error while fetching initial proof: %s\n", err.Error())
					}
				}

				// Process the client proof and measure the time it took to update

				start := time.Now()
				partialMpts[clientIdx].ProcessUpdatesFromBytes(pu.proofUpdate)
				atomic.AddInt64(&(updateTimes[clientIdx]), time.Since(start).Nanoseconds())
				partialMptLocks[clientIdx].Unlock()

				atomic.AddInt64(&(updates[clientIdx]), 1)
				atomic.AddInt64(&updateSizes[clientIdx], int64(len(pu.proofUpdate)))
			}

			for i, pl := range proofLogs {
				updateSizeKB := float64(updateSizes[i]) / float64(updates[i]) / float64(1000)
				updateTimeMS := float64(updateTimes[i]) / float64(updates[i]) / float64(100000)

				results[i] = clientUpdateBenchResult{trackingLogCount: pl, averageUpdateSizeKB: updateSizeKB, averageUpdateTimeMS: updateTimeMS}
			}
			wg.Done()
		}()
	}

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

	logIdIdx := make([]uint64, totalLogs)
	logIdxsToChange := make([]int, totalLogs)
	for i := range logIdxsToChange {
		logIdIdx[i] = 0
		logIdxsToChange[i] = i
	}
	mathrand.Seed(time.Now().UnixNano())
	mathrand.Shuffle(len(logIdxsToChange), func(i, j int) { logIdxsToChange[i], logIdxsToChange[j] = logIdxsToChange[j], logIdxsToChange[i] })

	var subset = logIdxsToChange[:]
	logId := [32]byte{}
	for i := 0; i < CLIENTUPDATE_SAMPLES; i++ {
		// pick a changing random subset of logs every run
		if totalLogs > numChangeLogs {
			startIdx := mathrand.Intn(totalLogs - numChangeLogs)
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
		fmt.Printf("\r[%d/%d] Committing server [%d]                    ", i+1, CLIENTUPDATE_SAMPLES, len(logIdxsToChange))
		srv.Commit()
	}

	close(proofUpdatesChan)
	wg.Wait()
	fmt.Printf("\nDone\n")

	return results
}
