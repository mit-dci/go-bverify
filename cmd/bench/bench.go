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

	flag.Parse()
	if *runProofSize || *runAll {
		benchmarks.RunProofSizeBench()
	}

	if *runClientUpdate || *runAll {
		benchmarks.RunClientUpdateBench()
	}

	if *runProofSizePerLog || *runAll {
		benchmarks.RunProofSizePerLogBench()
	}
}
