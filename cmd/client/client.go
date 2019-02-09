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

	cli := make([]*TestClient, 250)

	for i := 0; i < 250; i++ {
		key := make([]byte, 32)
		n, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		if n != 32 {
			panic("No 32 byte key could be read from random")
		}
		cl, err := client.NewClient(key)
		if err != nil {
			panic(err)
		}

		cli[i] = &TestClient{cli: cl}

	}

	var wg sync.WaitGroup
	for _, c := range cli {
		wg.Add(1)
		go func(c *TestClient) {
			logHash := fastsha256.Sum256([]byte("Hello world"))
			c.logID, err = c.cli.StartLog(logHash[:])
			wg.Add(-1)
		}(c)
	}
	wg.Wait()

	for i := uint64(1); i < 10000; i++ {
		logHash := fastsha256.Sum256([]byte(fmt.Sprintf("Hello world %d", i)))
		for _, c := range cli {
			wg.Add(1)
			go func(c *TestClient) {
				err = c.cli.AppendLog(i, c.logID, logHash[:])
				if err != nil {
					panic(err)
				}
				wg.Add(-1)
			}(c)
		}
		wg.Wait()
	}
}
