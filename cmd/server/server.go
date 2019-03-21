package main

import (
	"flag"

	"github.com/mit-dci/go-bverify/server"
)

func main() {
	rescanBlocks := flag.Int("rescan", 0, "Rescan this number of blocks on startup")
	flag.Parse()

	srv, _ := server.NewServer(":9100", *rescanBlocks)
	srv.Full = true
	srv.Run()
}
