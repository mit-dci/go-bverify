package benchmarks

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/mit-dci/go-bverify/client"

	"github.com/gonum/stat"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/server"
)

const (
	MICROBENCH_TOTALLOGS  = 10000000
	MICROBENCH_UPDATELOGS = 100000 // 1%
	MICROBENCH_NUMCLIENTS = 100
	MICROBENCH_RUNS       = 10
)

func RunMicroBench() {
	srv, err := server.NewServer("", 0)
	if err != nil {
		panic(err)
	}
	srv.AutoCommit = false

	// Create all logs first
	// Store the log IDs into one big byteslice
	logIds := make([]byte, 32*MICROBENCH_TOTALLOGS)
	clients := make([]*client.Client, MICROBENCH_NUMCLIENTS)
	processors := make([]server.LogProcessor, MICROBENCH_NUMCLIENTS)
	keys := make([][32]byte, MICROBENCH_NUMCLIENTS)

	logging.Debugf("Microbench: Creating clients       ")

	// Create the clients
	for i := 0; i < MICROBENCH_NUMCLIENTS; i++ {
		keys[i] = [32]byte{}
		rand.Read(keys[i][:])
		c, p := newDummyClientAndProcessor(srv)
		processors[i] = p
		clients[i], _ = client.NewClientWithConnection(keys[i][:], c)
	}

	createLogs := make(chan int, MICROBENCH_TOTALLOGS)
	// Create the logs
	for i := 0; i < MICROBENCH_TOTALLOGS; i++ {
		createLogs <- i
	}

	// When first filling the tree we don't need to sign/verify. This saves a lot
	// of time.
	for i := 0; i < MICROBENCH_NUMCLIENTS; i++ {
		clients[i].DummySignatures = true
	}
	srv.CheckSignatures = false

	close(createLogs)

	var wg sync.WaitGroup
	for iThread := 0; iThread < runtime.NumCPU(); iThread++ {
		wg.Add(1)
		go func() {
			for i := range createLogs {
				if i%MICROBENCH_UPDATELOGS == 0 {
					logging.Debugf("Microbench: Creating logs [%d/%d]", i+1, MICROBENCH_TOTALLOGS)
				}
				client := clients[i%MICROBENCH_NUMCLIENTS]
				logId, err := client.StartLog([]byte(fmt.Sprintf("Hello world %d", i)))
				if err != nil {
					panic(err)
				}
				copy(logIds[i*32:], logId[:])
			}
			wg.Done()
		}()
	}

	wg.Wait()
	srv.Commit()

	// Switch on signature creation and validation
	srv.CheckSignatures = true
	for i := 0; i < MICROBENCH_NUMCLIENTS; i++ {
		clients[i].DummySignatures = false
	}

	runTimesSigCheck := make([]float64, MICROBENCH_UPDATELOGS)
	runTimesMPTUpdate := make([]float64, MICROBENCH_UPDATELOGS)
	runTimesCommit := make([]float64, MICROBENCH_RUNS)

	logging.Debugf("Microbench: Shuffling log IDs       ")

	logIdIdx := make([]uint64, MICROBENCH_TOTALLOGS)
	logIdxsToChange := make([]int, MICROBENCH_TOTALLOGS)
	for i := range logIdxsToChange {
		logIdIdx[i] = 0
		logIdxsToChange[i] = i
	}
	mathrand.Seed(time.Now().UnixNano())
	mathrand.Shuffle(len(logIdxsToChange), func(i, j int) { logIdxsToChange[i], logIdxsToChange[j] = logIdxsToChange[j], logIdxsToChange[i] })

	totalNodes := 0
	runTimesDeltaProof := make([]float64, MICROBENCH_RUNS)
	avgHashes := 0

	// Now we're ready to run the first benchmark, that benchmarks signature verification, MPT update and commitment

	numHashes := float64(0)

	for iRun := 0; iRun < MICROBENCH_RUNS; iRun++ {
		logging.Debugf("Microbench: Running benchmark 1/3 [%d/%d]      ", iRun+1, MICROBENCH_RUNS)

		startIdx := mathrand.Intn(MICROBENCH_TOTALLOGS - MICROBENCH_UPDATELOGS)
		subset := logIdxsToChange[startIdx : startIdx+MICROBENCH_UPDATELOGS]
		statement := make([]byte, 32)
		rand.Read(statement)
		var logId [32]byte
		for j := 0; j < MICROBENCH_UPDATELOGS; j++ {
			logIdIdx[subset[j]]++
			copy(logId[:], logIds[subset[j]*32:subset[j]*32+32])
			client := clients[subset[j]%MICROBENCH_NUMCLIENTS]
			processor := processors[subset[j]%MICROBENCH_NUMCLIENTS]

			client.DummySignatures = !(iRun == 0)
			sls, err := client.SignedAppendLog(logIdIdx[subset[j]], logId, statement)
			if err != nil {
				panic(err)
			}

			// The two benchmarked operations (check signature and MPT update)
			if iRun == 0 {
				statsIdx := j + iRun*MICROBENCH_UPDATELOGS
				start := time.Now()
				processor.(*server.ServerLogProcessor).VerifyAppendLog(sls)
				runTimesSigCheck[statsIdx] = float64(time.Since(start).Nanoseconds())

				start = time.Now()
				processor.(*server.ServerLogProcessor).CommitAppendLog(sls)
				runTimesMPTUpdate[statsIdx] = float64(time.Since(start).Nanoseconds())
			} else {
				// No need to verify in the other runs as that's not what we're benchmarking
				processor.(*server.ServerLogProcessor).CommitAppendLog(sls)
			}
		}

		if iRun == 0 {
			totalNodes = srv.CountNodes()
		}

		numHashes += float64(srv.CountRecalculations())

		start := time.Now()
		srv.Commitment()
		runTimesCommit[iRun] = float64(time.Since(start).Nanoseconds())

		// Commit handles other stuff such as pushing delta updates to connected clients. We only
		// want to benchmark the commitment itself.
		srv.Commit()
	}

	avgHashes = int32(numHashes / float64(MICROBENCH_RUNS))
	// Print raw values

	// Add a new client, subscribe to a single log ID. Then repeatedly add statements to 1% of the other logs
	// and measure the time to generate a proof update for the single log

	for iRun := 0; iRun < MICROBENCH_RUNS; iRun++ {
		// Make the subset one bigger to ensure we can skip the randLogId
		startIdx := mathrand.Intn(MICROBENCH_TOTALLOGS - MICROBENCH_UPDATELOGS - 1)
		subset := logIdxsToChange[startIdx : startIdx+MICROBENCH_UPDATELOGS+1]
		statement := make([]byte, 32)
		rand.Read(statement)
		randLogId := mathrand.Intn(MICROBENCH_TOTALLOGS)

		logging.Debugf("Microbench: Running benchmark 2/3 [%d/%d]      ", iRun+1, MICROBENCH_RUNS)
		var logId [32]byte
		for j := 0; j < MICROBENCH_UPDATELOGS; j++ {
			logIdIdx[subset[j]]++
			copy(logId[:], logIds[subset[j]*32:subset[j]*32+32])
			srv.RegisterLogStatement(logId, logIdIdx[subset[j]], statement)
		}

		srv.Commit()
		start := time.Now()
		p, err := srv.GetDeltaProofForKeys([][]byte{logIds[randLogId*32 : randLogId*32+32]})
		logging.Debugf("Retrieved delta proof of size %d", p.ByteSize())
		runTimesDeltaProof[iRun] = float64(time.Since(start).Nanoseconds())
		if err != nil {
			panic(err)
		}

	}

	runTimesFullProof := make([]float64, MICROBENCH_RUNS)

	for iRun := 0; iRun < MICROBENCH_RUNS; iRun++ {
		// Make the subset one bigger to ensure we can skip the randLogId
		startIdx := mathrand.Intn(MICROBENCH_TOTALLOGS - (MICROBENCH_UPDATELOGS * 10) - 1)
		subset := logIdxsToChange[startIdx : startIdx+(MICROBENCH_UPDATELOGS*10)+1]
		statement := make([]byte, 32)
		rand.Read(statement)
		randLogId := mathrand.Intn(MICROBENCH_TOTALLOGS)

		logging.Debugf("Microbench: Running benchmark 3/3 [%d/%d]      ", iRun+1, MICROBENCH_RUNS)
		var logId [32]byte
		for j := 0; j < MICROBENCH_UPDATELOGS*10; j++ {
			logIdIdx[subset[j]]++
			copy(logId[:], logIds[subset[j]*32:subset[j]*32+32])
			srv.RegisterLogStatement(logId, logIdIdx[subset[j]], statement)
		}

		srv.Commit()
		start := time.Now()
		p, err := srv.GetProofForKeys([][]byte{logIds[randLogId*32 : randLogId*32+32]})
		logging.Debugf("Retrieved full proof of size %d", p.ByteSize())
		runTimesFullProof[iRun] = float64(time.Since(start).Nanoseconds())
		if err != nil {
			panic(err)
		}

	}

	logging.Debugf("Microbench: Writing output                ")

	table, _ := os.Create("table_microbench.tex")
	table.Write([]byte("\\begin{table*}[t]\n"))
	table.Write([]byte("\\centering\n"))
	table.Write([]byte("\\begin{tabular}{ |c||c|c| }\n"))
	table.Write([]byte(" \\hline\n"))
	table.Write([]byte(" Operation & Time (ms) & std \\\\\n"))
	table.Write([]byte(" \\hline \\hline\n"))

	table.Write([]byte(fmt.Sprintf("  Check signature & %.3f & %.3f \\\\\n", average(runTimesSigCheck)/float64(1000000), stat.StdDev(runTimesSigCheck, nil)/float64(1000000))))
	table.Write([]byte(fmt.Sprintf("  Single MPT Update & %.3f & %.3f \\\\\n", average(runTimesMPTUpdate)/float64(1000000), stat.StdDev(runTimesMPTUpdate, nil)/float64(1000000))))
	table.Write([]byte(fmt.Sprintf("  Batch Commitment & %.3f & %.3f \\\\\n", average(runTimesCommit)/float64(1000000), stat.StdDev(runTimesCommit, nil)/float64(1000000))))

	table.Write([]byte("\\hline\n"))

	table.Write([]byte(fmt.Sprintf("  Proof Updates Generation & %.4f & %.4f \\\\\n", average(runTimesDeltaProof)/float64(1000000), stat.StdDev(runTimesDeltaProof, nil)/float64(1000000))))
	table.Write([]byte(fmt.Sprintf("  Full Proof Generation & %.4f & %.4f \\\\\n", average(runTimesFullProof)/float64(1000000), stat.StdDev(runTimesFullProof, nil)/float64(1000000))))

	table.Write([]byte("\\hline\n"))
	table.Write([]byte("\\end{tabular}\n"))
	table.Write([]byte("\\caption{Micro benchmarks of the commitment server. The first group of operations are the steps to commit a new log statement and the next two groups are generations of proofs, which are there own operations.}\n"))
	table.Write([]byte("\\label{table:microbench}\n"))
	table.Write([]byte("\\end{table*}	\n"))
	table.Close()

	table, _ = os.Create("table_microbench.raw")
	table.Write([]byte("Raw values for signature check:\n----\n"))
	for _, m := range runTimesSigCheck {
		table.Write([]byte(fmt.Sprintf("%f\n", m)))
	}
	table.Write([]byte("\n\nRaw values for mpt update:\n----\n"))
	for _, m := range runTimesMPTUpdate {
		table.Write([]byte(fmt.Sprintf("%f\n", m)))
	}
	table.Write([]byte("\n\nRaw values for commit:\n----\n"))
	for _, m := range runTimesCommit {
		table.Write([]byte(fmt.Sprintf("%f\n", m)))
	}
	table.Write([]byte("\n\nRaw values for delta proof:\n----\n"))
	for _, m := range runTimesDeltaProof {
		table.Write([]byte(fmt.Sprintf("%f\n", m)))
	}
	table.Write([]byte("\n\nRaw values for full proof:\n----\n"))
	for _, m := range runTimesFullProof {
		table.Write([]byte(fmt.Sprintf("%f\n", m)))
	}

	table.Write([]byte(fmt.Sprintf("\n\nTotal nodes: %d - Avg update hashes:%d\n----\n", totalNodes, avgHashes)))

	table.Close()
	logging.Debugf("Microbench: Done.                                  \n")

}

func average(xs []float64) float64 {
	total := 0.0
	for _, v := range xs {
		total += v
	}
	return total / float64(len(xs))
}
