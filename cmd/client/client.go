package main

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
)

type TestClient struct {
	cli   *client.Client
	logID [32]byte
}

func main() {
	// Connect 250 clients
	var err error

	cli := make([]*TestClient, 2500)

	for i := 0; i < 2500; i++ {
		fmt.Printf("Constructing client %d\n", i)
		key := make([]byte, 32)
		n, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		if n != 32 {
			panic("No 32 byte key could be read from random")
		}
		fmt.Printf("Created key for client %d\n", i)
		cl, err := client.NewClient(key)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Assigning cli[%d]\n", i)
		cli[i] = &TestClient{cli: cl}
		fmt.Printf("Assigned cli[%d]\n", i)
	}

	fmt.Printf("Created 2500 clients. Starting their logs")

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

	fmt.Printf("Started 2500 logs. Adding 10k statements per log")
	for i := uint64(1); i < 10000; i++ {
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
}
