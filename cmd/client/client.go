package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
)

type TestClient struct {
	cli   *client.Client
	logID [32]byte
}

func main() {
	var err error

	hostName := flag.String("host", "localhost", "Host to connect to")
	hostPort := flag.Int("port", 9100, "Port to connect to")
	clients := flag.Int("clients", 250, "Number of clients to start")
	logs := flag.Int("logs", 1000, "Number of logs to write per client")
	flag.Parse()

	cli := make([]*TestClient, *clients)

	for i := 0; i < *clients; i++ {
		key := make([]byte, 32)
		n, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		if n != 32 {
			panic("No 32 byte key could be read from random")
		}
		cl, err := client.NewClient(key, fmt.Sprintf("%s:%d", *hostName, *hostPort))
		if err != nil {
			panic(err)
		}

		cli[i] = &TestClient{cli: cl}
	}

	fmt.Printf("\nCreated %d clients. Starting their logs", *clients)

	var wg sync.WaitGroup
	for _, c := range cli {
		wg.Add(1)
		go func(c *TestClient) {
			defer wg.Done()
			logHash := fastsha256.Sum256([]byte("Hello world"))
			c.logID, err = c.cli.StartLog(logHash[:])
		}(c)
	}
	wg.Wait()

	fmt.Printf("\nStarted %d logs. Adding %d statements per log", *clients, *logs)
	for i := uint64(1); i < uint64(*logs); i++ {
		logHash := fastsha256.Sum256([]byte(fmt.Sprintf("Hello world %d", i)))
		for _, c := range cli {
			wg.Add(1)
			go func(c *TestClient) {
				defer wg.Done()
				err = c.cli.AppendLog(i, c.logID, logHash[:])
				if err != nil {
					panic(err)
				}
			}(c)
		}
		wg.Wait()
	}

	fmt.Printf("\nAdded all statements, waiting for the server to commit")

	time.Sleep(time.Second * 15) // Wait for the server to process the commitments

	fmt.Printf("\nRequesting proofs")
	for _, c := range cli {
		wg.Add(1)
		go func(c *TestClient) {
			defer wg.Done()
			_, err := c.cli.RequestProof([][32]byte{})
			if err != nil {
				panic(err)
			}
		}(c)
	}
	wg.Wait()
	fmt.Printf("Done!")
}
