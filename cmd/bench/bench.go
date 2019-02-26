package main

import (
	"flag"

	"github.com/mit-dci/go-bverify/cmd/bench/benchmarks"
)

func main() {
	runAll := flag.Bool("all", false, "Run all benchmarks")
	runProofSize := flag.Bool("proofsize", false, "Run proof size benchmark")
	runProofSizePerLog := flag.Bool("proofsizeperlog", false, "Run proof size benchmark")

	flag.Parse()
	if *runProofSize || *runAll {
		benchmarks.RunProofSizeBench()
	}

	if *runProofSizePerLog || *runAll {
		benchmarks.RunProofSizePerLogBench()
	}
}
