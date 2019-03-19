package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/utils"
)

func main() {
	var err error

	hostName := flag.String("host", "localhost", "Host to connect to")
	hostPort := flag.Int("port", 9100, "Port to connect to")
	flag.Parse()

	os.MkdirAll(utils.ClientDataDirectory(), 0700)

	logging.SetLogLevel(int(logging.LogLevelDebug))

	logFilePath := path.Join(utils.ClientDataDirectory(), "b_verify_client.log")
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logFile.Close()
	logging.SetLogFile(logFile)

	// generate key
	keyFile := path.Join(utils.ClientDataDirectory(), "privkey.hex")
	key32 := [32]byte{}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		rand.Read(key32[:])
		ioutil.WriteFile(keyFile, key32[:], 0600)
	} else if err != nil {
		panic(err)
	} else {
		key, err := ioutil.ReadFile(keyFile)
		if err != nil {
			panic(err)
		}
		copy(key32[:], key)
	}

	logging.Debugf("Starting new client and connecting to %s:%d...", *hostName, *hostPort)
	cli, err := client.NewClient(key32[:], fmt.Sprintf("%s:%d", *hostName, *hostPort))
	if err != nil {
		panic(err)
	}

	logging.Debugf("Connected, starting SPV connection")
	err = cli.StartSPV()
	if err != nil {
		panic(err)
	}

	logging.Debugf("Starting receive loop")
	cli.ReceiveLoop()
}
