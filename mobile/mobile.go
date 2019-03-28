package mobile

import (
	"fmt"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/logging"
)

func RunBVerifyClient(hostName string, hostPort int) error {
	var err error

	// This is a fixed hash that the server will commit to first before even
	// becoming available to clients. This to ensure we always get the entire
	// chain when requesting commitments.
	logging.Debugf("Starting new client and connecting to %s:%d...", hostName, hostPort)
	cli, err := client.NewClient([]byte{}, fmt.Sprintf("%s:%d", hostName, hostPort))
	if err != nil {
		return err
	}

	go cli.Run(false)

	return nil
}
