package benchmarks

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/mit-dci/go-bverify/wire"

	"github.com/mit-dci/go-bverify/server"
)

var operationTimes []int64
var operationCounts []int64
var firstOperation []time.Time
var operationRunningTime []int64

func BenchmarkProxy(conn net.Conn, srv *server.Server) {
	c, s := net.Pipe()
	proc := server.NewLogProcessor(s, srv)
	go proc.Process()
	client := wire.NewConnection(conn)
	server := wire.NewConnection(c)

	// Intercept messages from client, forward to server - measure processing time
	for {
		t, p, err := client.ReadNextMessage()
		if err != nil {
			return
		}

		if firstOperation[int(t)-1].Year() == 2000 {
			firstOperation[int(t)-1] = time.Now()
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

		atomic.AddInt64(&(operationTimes[int(t)-1]), time.Since(start).Nanoseconds())
		atomic.AddInt64(&(operationCounts[int(t)-1]), 1)

		operationRunningTime[int(t)-1] = time.Since(firstOperation[int(t)-1]).Nanoseconds()
		client.WriteMessage(t2, p)
	}
}

func RunServerBench(port int) {
	stop := make(chan bool, 1)

	firstOperation = make([]time.Time, 11)
	for i := range firstOperation {
		firstOperation[i] = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	operationTimes = make([]int64, 11)       // Total number of messages
	operationCounts = make([]int64, 11)      // Total number of messages
	operationRunningTime = make([]int64, 11) // Total number of messages

	srv, err := server.NewServer("")
	if err != nil {
		panic(err)
	}

	commitTicker := time.NewTicker(time.Second * 5)
	go func(s *server.Server) {
		for range commitTicker.C {
			s.Commit()
		}
	}(srv)

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			stop <- true
			listener.Close()
		}
	}()

	fmt.Printf("\nStarting server benchmark. Press ^C to end the benchmark and produce results.\n\nServer benchmark: Server is listening for requests. ")

	for {
		breakOut := false
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-stop:
				breakOut = true
			default:
				continue
			}
		}

		if breakOut {
			break
		}
		go BenchmarkProxy(conn, srv)
	}

	operationNames := []string{"Create new log", "Add log statement", "", "", "", "Request Full Proof", "", "", "", "", "", ""}

	fmt.Printf("\nServer benchmark: Writing output                                 ")

	table, _ := os.Create("table_serversimulation.tex")
	table.Write([]byte("\\begin{table*}[t]\n"))
	table.Write([]byte("\\centering\n"))
	table.Write([]byte("\\begin{tabular}{ |c||c|c| }\n"))
	table.Write([]byte(" \\hline\n"))
	table.Write([]byte(" Operation & Response time (ms) & Average Throughput (Operations/Second) \\\\\n"))
	table.Write([]byte(" \\hline \\hline\n"))

	for i := range operationTimes {
		totalTime := atomic.LoadInt64(&(operationTimes[i]))
		totalCount := atomic.LoadInt64(&(operationCounts[i]))
		if totalCount > 0 && operationNames[i] != "" {
			avgTime := float64(totalTime) / float64(totalCount) / float64(1000000)
			opsSec := float64(totalCount) / (float64(operationRunningTime[i]) / float64(1000000000)) //server running time is in nanosec, convert to sec first
			table.Write([]byte(fmt.Sprintf("  %s & %.3f & %.3f \\\\\n", operationNames[i], avgTime, opsSec)))
		}
	}

	table.Write([]byte("\\hline\n"))
	table.Write([]byte("\\end{tabular}\n"))
	table.Write([]byte("\\caption{Simulation of commitment server facing heavy load. In these simulations large number of clients request operations on the server simultaneously. The test measures the amount of time required to respond to client requests, and the average throughput of the commitment server in performing these operations.}\n"))
	table.Write([]byte("\\label{table:simulation}\n"))
	table.Write([]byte("\\end{table*}	\n"))
	table.Close()

	fmt.Printf("\r Server bench : completed                                  \n")
}
