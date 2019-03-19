package uspv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/logging"
	"golang.org/x/net/proxy"
)

// IP4 ...
func IP4(ipAddress string) bool {
	parseIP := net.ParseIP(ipAddress)
	if parseIP.To4() == nil {
		return false
	}
	return true
}

func (s *SPVCon) parseRemoteNode(remoteNode string) (string, string, error) {
	colonCount := strings.Count(remoteNode, ":")
	var conMode string
	if colonCount <= 1 {
		if colonCount == 0 {
			remoteNode = remoteNode + ":" + s.Param.DefaultPort
		}
		return remoteNode, "tcp4", nil
	} else if colonCount >= 5 {
		// ipv6 without remote port
		// assume users don't give ports with ipv6 nodes
		if !strings.Contains(remoteNode, "[") && !strings.Contains(remoteNode, "]") {
			remoteNode = "[" + remoteNode + "]" + ":" + s.Param.DefaultPort
		}
		conMode = "tcp6"
		return remoteNode, conMode, nil
	} else {
		return "", "", fmt.Errorf("Invalid ip")
	}
}

// GetListOfNodes contacts all DNSSeeds for the coin specified and then contacts
// each one of them in order to receive a list of ips and then returns a combined
// list
func (s *SPVCon) GetListOfNodes() ([]string, error) {
	var listOfNodes []string // slice of IP addrs returned from the DNS seed
	logging.Infof("Attempting to retrieve peers to connect to based on DNS Seed\n")

	for _, seed := range s.Param.DNSSeeds {
		temp, err := net.LookupHost(seed)
		// need this temp in order to capture the error from net.LookupHost
		// also need this to report the number of IPs we get from a seed
		if err != nil {
			logging.Infof("Have difficulty trying to connect to %s. Going to the next seed", seed)
			continue
		}
		listOfNodes = append(listOfNodes, temp...)
		logging.Infof("Got %d IPs from DNS seed %s\n", len(temp), seed)
	}
	if len(listOfNodes) == 0 {
		return nil, fmt.Errorf("No peers found connected to DNS Seeds. Please provide a host to connect to.")
	}
	logging.Info(listOfNodes)
	return listOfNodes, nil
}

// DialNode receives a list of node ips and then tries to connect to them one by one.
func (s *SPVCon) DialNode(listOfNodes []string) error {

	// now have some IPs, go through and try to connect to one.
	var err error
	for i, ip := range listOfNodes {
		// try to connect to all nodes in this range
		var conString, conMode string
		// need to check whether conString is ipv4 or ipv6
		conString, conMode, err = s.parseRemoteNode(ip)
		if err != nil {
			logging.Infof("parse error for node (skipped): %s", err)
			continue
		}
		logging.Infof("Attempting connection to node at %s\n",
			conString)

		if s.ProxyURL != "" {
			logging.Infof("Attempting to connect via proxy %s", s.ProxyURL)
			d, err := proxy.SOCKS5("tcp", s.ProxyURL, nil, proxy.Direct)
			if err != nil {
				return err
			}
			s.con, err = d.Dial(conMode, conString)
		} else {
			d := net.Dialer{Timeout: 2 * time.Second}
			s.con, err = d.Dial(conMode, conString)
		}

		if err != nil {
			if i != len(listOfNodes)-1 {
				logging.Warn(err.Error())
				continue
			} else if i == len(listOfNodes)-1 {
				logging.Error(err)
				// all nodes have been exhausted, we move on to the next one, if any.
				return fmt.Errorf(" Tried to connect to all available node Addresses. Failed")
			}
		}
		break
	}

	if s.con == nil {
		return fmt.Errorf("Failed to connect to a coin daemon")
	}

	return nil
}

// Handshake ...
func (s *SPVCon) Handshake(listOfNodes []string) error {
	// assign version bits for local node
	s.localVersion = VERSION
	myMsgVer, err := wire.NewMsgVersionFromConn(s.con, 0, 0)
	if err != nil {
		return err
	}
	err = myMsgVer.AddUserAgent("lit", "v0.1")
	if err != nil {
		return err
	}
	// set this to enable segWit
	myMsgVer.AddService(wire.SFNodeWitness)
	// this actually sends
	n, err := wire.WriteMessageWithEncodingN(
		s.con, myMsgVer, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	logging.Infof("wrote %d byte version message to %s\n",
		n, s.con.RemoteAddr().String())
	n, m, b, err := wire.ReadMessageWithEncodingN(
		s.con, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		logging.Error(err)
		return err
	}
	s.RBytes += uint64(n)
	logging.Infof("got %d byte response %x\n command: %s\n", n, b, m.Command())

	mv, ok := m.(*wire.MsgVersion)
	if !ok {
		return fmt.Errorf("Cast to MsgVersion failed.")
	}

	logging.Infof("connected to %s", mv.UserAgent)

	if mv.ProtocolVersion < 70013 {
		//70014 -> core v0.13.1, so we should be fine
		return fmt.Errorf("Remote node version: %x too old, disconnecting.", mv.ProtocolVersion)
	}

	if !((strings.Contains(s.Param.Name, "lite") && strings.Contains(mv.UserAgent, "LitecoinCore")) || strings.Contains(mv.UserAgent, "Satoshi") || strings.Contains(mv.UserAgent, "btcd")) && (len(listOfNodes) != 0) {
		// TODO: improve this filtering criterion
		return fmt.Errorf("Couldn't connect to this node. Returning!")
	}

	logging.Infof("remote reports version %x (dec %d)\n",
		mv.ProtocolVersion, mv.ProtocolVersion)

	// set remote height
	s.remoteHeight = mv.LastBlock
	// set remote version
	s.remoteVersion = uint32(mv.ProtocolVersion)

	mva := wire.NewMsgVerAck()
	n, err = wire.WriteMessageWithEncodingN(
		s.con, mva, s.localVersion,
		wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
	if err != nil {
		return err
	}
	s.WBytes += uint64(n)
	return nil
}

var listOfNodes []string

// Connect dials out and connects to full nodes. Calls GetListOfNodes to get the
// list of nodes if the user has specified a YupString. Else, moves on to dial
// the node to see if its up and establishes a connection followed by Handshake()
// which sends out wire messages, checks for version string to prevent spam, etc.
func (s *SPVCon) Connect() error {
	var err error

	if len(listOfNodes) == 0 {
		listOfNodes, err = s.GetListOfNodes()
		if err != nil {
			logging.Error(err)
			return err
			// automatically quit if there are no other hosts to connect to.
		}
	}

	handShakeFailed := false // need to be in this scope to access it here
	connEstablished := false
	for len(listOfNodes) != 0 && !connEstablished {
		err = s.DialNode(listOfNodes)
		if err != nil {
			logging.Error(err)
			logging.Infof("Couldn't dial node %s, Moving on", listOfNodes[0])
			listOfNodes = listOfNodes[1:]
			continue
		}
		err = s.Handshake(listOfNodes)
		if err != nil {
			// spam node or some other problem. Delete node from list and try again
			handShakeFailed = true
			logging.Infof("Handshake with %s failed. Moving on. Error: %s", listOfNodes[0], err.Error())
			if len(listOfNodes) == 1 { // this is the last node, error out
				return fmt.Errorf("Couldn't establish connection with any remote node. Exiting.")
			}
			logging.Error("Couldn't establish connection with node. Proceeding to the next one")
			listOfNodes = listOfNodes[1:]
			connEstablished = false
		} else {
			connEstablished = true
			listOfNodes = listOfNodes[1:]
		}
	}

	if !handShakeFailed && !connEstablished {
		// this case happens when user provided node fails to connect
		return fmt.Errorf("Couldn't establish connection with node. Exiting.")
	}
	if handShakeFailed && !connEstablished {
		// this case is when the last node fails and we continue, only to exit the
		// loop and execute below code, which is unnecessary.
		return fmt.Errorf("Couldn't establish connection with any remote node after an instance of handshake. Exiting.")
	}
	go s.incomingMessageHandler()
	go s.outgoingMessageHandler()

	err = s.AskForHeaders()
	if err != nil {
		return err
	}
	return nil
}

/*
Truncated header files
Like a regular header but the first 80 bytes is mostly empty.
The very first 4 bytes (big endian) says what height the empty 80 bytes
replace.  The next header, starting at offset 80, needs to be valid.
*/
func (s *SPVCon) openHeaderFile(hfn string) error {
	_, err := os.Stat(hfn)
	if err != nil {
		if os.IsNotExist(err) {
			var b bytes.Buffer
			// if StartHeader is defined, start with hardcoded height
			if s.Param.StartHeight != 0 {
				hdr := s.Param.StartHeader
				_, err := b.Write(hdr[:])
				if err != nil {
					return err
				}
			} else {
				err = s.Param.GenesisBlock.Header.Serialize(&b)
				if err != nil {
					return err
				}
			}
			err = ioutil.WriteFile(hfn, b.Bytes(), 0600)
			if err != nil {
				return err
			}
			logging.Infof("made genesis header %x\n", b.Bytes())
			logging.Infof("made genesis hash %s\n", s.Param.GenesisHash.String())
			logging.Infof("created hardcoded genesis header at %s\n", hfn)
		}
	}

	if s.Param.StartHeight != 0 {
		s.headerStartHeight = s.Param.StartHeight
	}

	s.headerFile, err = os.OpenFile(hfn, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	logging.Infof("opened header file %s\n", s.headerFile.Name())
	return nil
}
