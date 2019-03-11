package benchmarks

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand"
	"os"
	"time"

	"github.com/mit-dci/go-bverify/client"

	"github.com/gonum/stat"
	"github.com/mit-dci/go-bverify/server"
)

const (
	MICROBENCH_TOTALLOGS  = 100000
	MICROBENCH_UPDATELOGS = 1000 // 1%
	MICROBENCH_NUMCLIENTS = 100
	MICROBENCH_RUNS       = 100
)

func RunMicroBench() {
	srv, err := server.NewServer("")
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

	fmt.Printf("\nMicrobench: Creating clients       ")

	// Create the clients
	for i := 0; i < MICROBENCH_NUMCLIENTS; i++ {
		keys[i] = [32]byte{}
		rand.Read(keys[i][:])
		c, p := newDummyClientAndProcessor(srv)
		processors[i] = p
		clients[i], _ = client.NewClientWithConnection(keys[i][:], c)
	}

	// Create the logs
	for i := 0; i < MICROBENCH_TOTALLOGS; i++ {
		if i%10000 == 0 {
			fmt.Printf("\rMicrobench: Creating logs [%d/%d]      ", i+1, MICROBENCH_TOTALLOGS)
		}
		client := clients[i%MICROBENCH_NUMCLIENTS]
		logId, err := client.StartLog([]byte(fmt.Sprintf("Hello world %d", i)))
		if err != nil {
			panic(err)
		}
		copy(logIds[i*32:], logId[:])
	}

	runTimesSigCheck := make([]float64, MICROBENCH_RUNS*MICROBENCH_UPDATELOGS)
	runTimesMPTUpdate := make([]float64, MICROBENCH_RUNS*MICROBENCH_UPDATELOGS)
	runTimesCommit := make([]float64, MICROBENCH_RUNS)

	fmt.Printf("\rMicrobench: Shuffling log IDs       ")

	logIdIdx := make([]uint64, MICROBENCH_TOTALLOGS)
	logIdxsToChange := make([]int, MICROBENCH_TOTALLOGS)
	for i := range logIdxsToChange {
		logIdIdx[i] = 0
		logIdxsToChange[i] = i
	}
	mathrand.Seed(time.Now().UnixNano())
	mathrand.Shuffle(len(logIdxsToChange), func(i, j int) { logIdxsToChange[i], logIdxsToChange[j] = logIdxsToChange[j], logIdxsToChange[i] })

	// Now we're ready to run the first benchmark, that benchmarks signature verification, MPT update and commitment
	for iRun := 0; iRun < MICROBENCH_RUNS; iRun++ {
		fmt.Printf("\rMicrobench: Running benchmark 1/3 [%d/%d]      ", iRun+1, MICROBENCH_RUNS)

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

			sls, err := client.SignedAppendLog(logIdIdx[subset[j]], logId, statement)
			if err != nil {
				panic(err)
			}

			// The two benchmarked operations (check signature and MPT update)
			statsIdx := j + iRun*MICROBENCH_UPDATELOGS

			start := time.Now()
			processor.(*server.ServerLogProcessor).VerifyAppendLog(sls)
			runTimesSigCheck[statsIdx] = float64(time.Since(start).Nanoseconds()) / float64(1000000)

			start = time.Now()
			processor.(*server.ServerLogProcessor).CommitAppendLog(sls)
			runTimesMPTUpdate[statsIdx] = float64(time.Since(start).Nanoseconds()) / float64(1000000)
		}

		start := time.Now()
		srv.Commitment()
		runTimesCommit[iRun] = float64(time.Since(start).Nanoseconds()) / float64(1000000)

		// Commit handles other stuff such as pushing delta updates to connected clients. We only
		// want to benchmark the commitment itself.
		srv.Commit()
	}

	// Add a new client, subscribe to a single log ID. Then repeatedly add statements to 1% of the other logs
	// and measure the time to generate a proof update for the single log

	runTimesDeltaProof := make([]float64, MICROBENCH_RUNS)

	c, _ := newDummyClientAndProcessor(srv)
	deltaTestKey := [32]byte{}
	rand.Read(deltaTestKey[:])
	deltaTestClient, _ := client.NewClientWithConnection(deltaTestKey[:], c)
	deltaTestLogId, err := deltaTestClient.StartLog([]byte("Hello World Delta"))
	if err != nil {
		panic(err)
	}
	for iRun := 0; iRun < MICROBENCH_RUNS; iRun++ {
		startIdx := mathrand.Intn(MICROBENCH_TOTALLOGS - MICROBENCH_UPDATELOGS)
		subset := logIdxsToChange[startIdx : startIdx+MICROBENCH_UPDATELOGS]
		statement := make([]byte, 32)
		rand.Read(statement)

		fmt.Printf("\rMicrobench: Running benchmark 2/3 [%d/%d]      ", iRun+1, MICROBENCH_RUNS)
		var logId [32]byte
		for j := 0; j < MICROBENCH_UPDATELOGS; j++ {
			logIdIdx[subset[j]]++
			copy(logId[:], logIds[subset[j]*32:subset[j]*32+32])
			srv.RegisterLogStatement(logId, logIdIdx[subset[j]], statement)
		}

		srv.Commit()
		start := time.Now()
		srv.GetDeltaProofForKeys([][]byte{deltaTestLogId[:]})
		runTimesDeltaProof[iRun] = float64(time.Since(start).Nanoseconds()) / float64(1000000)
	}

	fmt.Printf("\rMicrobench: Writing output                ")

	table, _ := os.Create("table_microbench.tex")
	table.Write([]byte("\\begin{table*}[t]\n"))
	table.Write([]byte("\\centering\n"))
	table.Write([]byte("\\begin{tabular}{ |c||c|c| }\n"))
	table.Write([]byte(" \\hline\n"))
	table.Write([]byte(" Operation & Time (ms) & std \\\\\n"))
	table.Write([]byte(" \\hline \\hline\n"))

	table.Write([]byte(fmt.Sprintf("  Check signature & %.4f & %.4f \\\\\n", average(runTimesSigCheck), stat.StdDev(runTimesSigCheck, nil))))
	table.Write([]byte(fmt.Sprintf("  Single MPT Update & %.4f & %.4f \\\\\n", average(runTimesMPTUpdate), stat.StdDev(runTimesMPTUpdate, nil))))
	table.Write([]byte(fmt.Sprintf("  Batch Commitment & %.4f & %.4f \\\\\n", average(runTimesCommit), stat.StdDev(runTimesCommit, nil))))

	table.Write([]byte("\\hline\n"))

	table.Write([]byte(fmt.Sprintf("  Proof Updates Generation & %.4f & %.4f \\\\\n", average(runTimesDeltaProof), stat.StdDev(runTimesDeltaProof, nil))))
	table.Write([]byte(fmt.Sprintf("  Full Proof Generation & %.4f & %.4f \\\\\n", 0.0, 0.0)))

	table.Write([]byte("\\hline\n"))
	table.Write([]byte("\\end{tabular}\n"))
	table.Write([]byte("\\caption{Micro benchmarks of the commitment server. The first group of operations are the steps to commit a new log statement and the next two groups are generations of proofs, which are there own operations.}\n"))
	table.Write([]byte("\\label{table:microbench}\n"))
	table.Write([]byte("\\end{table*}	\n"))
	table.Close()

	fmt.Printf("\rMicrobench: Done.                                  \n")

}

func average(xs []float64) float64 {
	total := 0.0
	for _, v := range xs {
		total += v
	}
	return total / float64(len(xs))
}
