package main

import (
	"flag"
	"fmt"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/logging"
)

func main() {
	var err error
	hostName := flag.String("host", "localhost", "Host to connect to")
	hostPort := flag.Int("port", 9100, "Port to connect to")
	resync := flag.Bool("resync", false, "Resynchronize commitments on startup")
	flag.Parse()

	// This is a fixed hash that the server will commit to first before even
	// becoming available to clients. This to ensure we always get the entire
	// chain when requesting commitments.
	logging.Debugf("Starting new client and connecting to %s:%d...", *hostName, *hostPort)
	cli, err := client.NewClient([]byte{}, fmt.Sprintf("%s:%d", *hostName, *hostPort))
	if err != nil {
		panic(err)
	}

	err = cli.Run(*resync)
	if err != nil {
		panic(err)
	}
}
