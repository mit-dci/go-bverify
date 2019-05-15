package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/mit-dci/go-bverify/cmd/bench/benchmarks"
	"github.com/mit-dci/go-bverify/logging"
)

func main() {
	runAll := flag.Bool("all", false, "Run all benchmarks")
	runProofSize := flag.Bool("proofsize", false, "Run proof size benchmark")
	runProofSizePerLog := flag.Bool("proofsizeperlog", false, "Run proof size benchmark")
	runClientUpdate := flag.Bool("clientupdate", false, "Run client update benchmark")
	runClientDeltaSize := flag.Bool("clientdelta", false, "Run client delta size benchmark")
	runMicroBench := flag.Bool("microbench", false, "Run commitment server microbenchmark")
	runServerBench := flag.Bool("serverbench", false, "Run commitment server in benchmark mode")
	serverBenchPort := flag.Int("serverbenchport", 9100, "Port to run the benchmark server on")
	runClientBench := flag.Bool("clientbench", false, "Run commitment server in benchmark mode")
	clientBenchHost := flag.String("clientbenchhost", "localhost", "Host running the benchmark server")
	clientBenchPort := flag.Int("clientbenchport", 9100, "Port to connect to the benchmark server on")
	clientBenchClients := flag.Int("clientbenchclients", 1000, "Number of clients to emulate")
	clientBenchLogs := flag.Int("clientbenchlogs", 1000, "Number of logs to write per client")
	clientBenchStatements := flag.Int("clientbenchstatements", 100, "Number of statements to write to each log")
	profileServer := flag.Bool("profileserver", false, "Run a live profiling server")
	memProfile := flag.String("memprofile", "mem.pprof", "Write a memory profile")
	cpuProfile := flag.String("cpuprofile", "cpu.pprof", "Write a cpu profile")

	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			logging.Errorf("could not create CPU profile: ", err)
			return
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			logging.Errorf("could not start CPU profile: ", err)
			return
		}
		defer pprof.StopCPUProfile()
	}

	logging.SetLogLevel(int(logging.LogLevelDebug))

	if *profileServer {

		go func() {
			log.Println("Profiling!")
			log.Println(http.ListenAndServe(":6060", nil))
		}()
	}

	if *runProofSize || *runAll {
		benchmarks.RunProofSizeBench()
	}

	if *runClientBench || *runAll {
		benchmarks.RunClientBench(*clientBenchHost, *clientBenchPort, *clientBenchClients, *clientBenchLogs, *clientBenchStatements)
	}

	if *runServerBench || *runAll {
		benchmarks.RunServerBench(*serverBenchPort)
	}

	if *runMicroBench || *runAll {
		benchmarks.RunMicroBench()
	}

	if *runClientUpdate || *runAll {
		benchmarks.RunClientUpdateBench()
	}

	if *runClientDeltaSize || *runAll {
		benchmarks.RunClientDeltaSizeBench()
	}

	if *runProofSizePerLog || *runAll {
		benchmarks.RunProofSizePerLogBench()
	}

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			logging.Errorf("could not create memory profile: ", err)
			return
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			logging.Errorf("could not write memory profile: ", err)
		}
	}
}
