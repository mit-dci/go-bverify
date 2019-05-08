package mobile

import (
	"fmt"

	"github.com/mit-dci/go-bverify/utils"

	"github.com/mit-dci/go-bverify/client"
	"github.com/mit-dci/go-bverify/logging"
)

var overrideDataDir string

func SetDataDir(dataDir string) {
	logging.Debugf("Setting datadir to %s", dataDir)
	utils.SetOverrideClientDataDirectory(dataDir)
}

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

	logging.Debugf("Connected, running client")

	go func() {
		err = cli.Run(false)
		if err != nil {
			logging.Errorf("Error running client: %s", err.Error())
		}
	}()

	return nil
}

func init() {
	logging.SetLogLevel(int(logging.LogLevelDebug))

}
