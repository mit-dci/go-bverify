package benchmarks

import (
	"crypto/rand"
	"os/signal"

	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/wire"
)

var clientOperationTimes []int64
var clientOperationCounts []int64
var clientFirstOperation []time.Time
var clientOperationRunningTime []int64

type TestClient struct {
	cli    *client.Client
	logIDs [][32]byte
}

func BenchmarkProxyClient(conn net.Conn) net.Conn {
	c, s := net.Pipe()
	server := wire.NewConnection(conn)
	client := wire.NewConnection(s)

	go func(client, server *wire.Connection) {
		// Intercept messages from client, forward to server - measure processing time
		for {
			t, p, err := client.ReadNextMessage()
			if err != nil {
				return
			}

			if clientFirstOperation[int(t)-1].Year() == 2000 {
				clientFirstOperation[int(t)-1] = time.Now()
			}

			start := time.Now()
			err = server.WriteMessage(t, p)
			if err != nil {
				return
			}

			t2, p, err := server.ReadNextMessage()
			if err != nil {
				return
			}

			atomic.AddInt64(&(clientOperationTimes[int(t)-1]), time.Since(start).Nanoseconds())
			atomic.AddInt64(&(clientOperationCounts[int(t)-1]), 1)

			clientOperationRunningTime[int(t)-1] = time.Since(clientFirstOperation[int(t)-1]).Nanoseconds()
			client.WriteMessage(t2, p)
		}
	}(client, server)

	return c
}

func RunClientBench(host string, port, numClients, numLogs, numStatements int) {
	var err error
	cli := make([]*TestClient, numClients)
	clientFirstOperation = make([]time.Time, 11)
	for i := range clientFirstOperation {
		clientFirstOperation[i] = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	clientOperationTimes = make([]int64, 11)       // Total number of messages
	clientOperationCounts = make([]int64, 11)      // Total number of messages
	clientOperationRunningTime = make([]int64, 11) // Total number of messages

	for i := 0; i < numClients; i++ {
		key := make([]byte, 32)
		n, err := rand.Read(key)
		if err != nil {
			panic(err)
		}
		if n != 32 {
			panic("No 32 byte key could be read from random")
		}

		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			panic(err)
		}

		c := BenchmarkProxyClient(conn)

		cl, err := client.NewClientWithConnection(key, c)
		if err != nil {
			panic(err)
		}
		cl.AckTimeout = time.Minute * 10   // increase to ridiculous height, we're measuring it but we don't want it to time out
		cl.ProofTimeout = time.Minute * 10 // increase to ridiculous height, we're measuring it but we don't want it to time out

		cli[i] = &TestClient{cli: cl, logIDs: make([][32]byte, numLogs)}
	}

	logging.Debugf("Created %d clients. Starting their logs", numClients)

	var wg sync.WaitGroup
	for _, c := range cli {
		wg.Add(1)
		go func(c *TestClient) {
			defer wg.Done()
			for i := 0; i < numLogs; i++ {
				logHash := fastsha256.Sum256([]byte(fmt.Sprintf("Hello world %d", i)))
				c.logIDs[i], err = c.cli.StartLog(logHash[:])
			}
		}(c)
	}
	wg.Wait()

	logging.Debugf("Started %d logs. Adding %d statements per log", numClients*numLogs, numStatements)
	for i := uint64(1); i < uint64(numStatements); i++ {
		for _, c := range cli {
			wg.Add(1)
			go func(c *TestClient) {
				defer wg.Done()
				for j := 0; j < numLogs; j++ {
					logHash := fastsha256.Sum256([]byte(fmt.Sprintf("Hello world %d %d", i, j)))
					err = c.cli.AppendLog(i, c.logIDs[j], logHash[:])
					if err != nil {
						panic(err)
					}
				}
			}(c)
		}
		wg.Wait()
	}

	logging.Debugf("Added all statements, press ^C when the server has committed")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	logging.Debugf("Requesting proofs")
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
	logging.Debugf("Server committed")

	operationNames := []string{"Create new log", "Add log statement", "", "", "", "Request Full Proof", "", "", "", "", "", ""}

	logging.Debugf("Writing output")

	table, _ := os.Create("table_clientsimulation.tex")
	table.Write([]byte("\\begin{table*}[t]\n"))
	table.Write([]byte("\\centering\n"))
	table.Write([]byte("\\begin{tabular}{ |c||c|c| }\n"))
	table.Write([]byte(" \\hline\n"))
	table.Write([]byte(" Operation & Response time (ms) & Average Throughput (Operations/Second) \\\\\n"))
	table.Write([]byte(" \\hline \\hline\n"))

	for i := range clientOperationTimes {
		totalTime := atomic.LoadInt64(&(clientOperationTimes[i]))
		totalCount := atomic.LoadInt64(&(clientOperationCounts[i]))
		if totalCount > 0 && operationNames[i] != "" {
			avgTime := float64(totalTime) / float64(totalCount) / float64(1000000)
			opsSec := float64(totalCount) / (float64(clientOperationRunningTime[i]) / float64(1000000000)) //server running time is in nanosec, convert to sec first
			table.Write([]byte(fmt.Sprintf("  %s & %.3f & %.3f \\\\\n", operationNames[i], avgTime, opsSec)))
		}
	}

	table.Write([]byte("\\hline\n"))
	table.Write([]byte("\\end{tabular}\n"))
	table.Write([]byte("\\caption{Client-side measurements of the server simulation.}\n"))
	table.Write([]byte("\\label{table:clientsimulation}\n"))
	table.Write([]byte("\\end{table*}	\n"))
	table.Close()

	logging.Debugf("Completed")
}
