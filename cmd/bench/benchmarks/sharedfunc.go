package benchmarks

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/mit-dci/go-bverify/server"
)

func makeDummyLogs(srv *server.Server, numLogs int) []byte {

	// Store the log IDs into one big byteslice
	logIds := make([]byte, 32*numLogs)

	// Since we're not actually verifying the statements, we can just
	// use random pubkeys, logIDs and witnesses
	pub33 := [33]byte{}
	_, err := rand.Read(pub33[:])
	if err != nil {
		panic(err)
	}

	c := make(chan int, numLogs)
	for logIdx := 0; logIdx < numLogs; logIdx++ {
		c <- logIdx
	}
	close(c)
	makeLogs := func(wg *sync.WaitGroup, idxChan <-chan int) {
		wg.Add(1)
		go func(idxChan <-chan int) {
			for idx := range idxChan {
				// Read a random witness and log ID
				witness := make([]byte, 32)
				logId := [32]byte{}
				rand.Read(logId[:])
				rand.Read(witness[:])

				// Create the log and write the first statement
				srv.RegisterLogID(logId, pub33)
				srv.RegisterLogStatement(logId, 0, witness[:])

				// startIdx determines the start position of the LogID in the
				// large byteslice we use to cache them
				startIdx := idx * 32
				// cache the generated LogID into the big array
				copy(logIds[startIdx:], logId[:])

				witness = nil
			}
			wg.Done()
		}(idxChan)
	}

	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		makeLogs(&wg, c)
	}
	go func() {
		fmt.Printf("\n")
		for {
			fmt.Printf("\rCreated %d/%d dummy logs", numLogs-len(c), numLogs)
			if len(c) == 0 {
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
	// Wait for all logs to be finished in the goroutines
	wg.Wait()

	return logIds
}
