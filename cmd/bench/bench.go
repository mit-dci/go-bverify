package main

import (
	"flag"

	"github.com/mit-dci/go-bverify/cmd/bench/benchmarks"
)

func main() {
	runAll := flag.Bool("all", false, "Run all benchmarks")
	runProofSize := flag.Bool("proofsize", false, "Run proof size benchmark")
	runProofSizePerLog := flag.Bool("proofsizeperlog", false, "Run proof size benchmark")
	runClientUpdate := flag.Bool("clientupdate", false, "Run client update benchmark")
	runClientDeltaSize := flag.Bool("clientdelta", false, "Run client delta size benchmark")

	flag.Parse()
	if *runProofSize || *runAll {
		benchmarks.RunProofSizeBench()
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
}
