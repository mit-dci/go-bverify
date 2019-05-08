package uspv

import (
	"github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/logging"
)

func (s *SPVCon) incomingMessageHandler() {
	for {
		n, xm, _, err := wire.ReadMessageWithEncodingN(s.con, s.localVersion,
			wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)
		if err != nil {
			logging.Infof("ReadMessageWithEncodingN error.  Disconnecting from given peer. %s\n", err.Error())
			s.con.Close() // close the connection to prevent spam messages from crashing lit.
			s.Connect()
			return
		}
		s.RBytes += uint64(n)
		//		logging.Infof("Got %d byte %s message\n", n, xm.Command())
		switch m := xm.(type) {
		case *wire.MsgVersion:
			logging.Infof("Got version message.  Agent %s, version %d, at height %d\n",
				m.UserAgent, m.ProtocolVersion, m.LastBlock)
			s.remoteVersion = uint32(m.ProtocolVersion) // weird cast! bug?
		case *wire.MsgVerAck:
			logging.Infof("Got verack.  Whatever.\n")
		case *wire.MsgAddr:
			logging.Infof("got %d addresses.\n", len(m.AddrList))
		case *wire.MsgPing:
			// logging.Infof("Got a ping message.  We should pong back or they will kick us off.")
			go s.PongBack(m.Nonce)
		case *wire.MsgPong:
			logging.Infof("Got a pong response. OK.\n")
		case *wire.MsgHeaders: // concurrent because we keep asking for blocks
			go s.HeaderHandler(m)
		case *wire.MsgReject:
			logging.Infof("Rejected! cmd: %s code: %s tx: %s reason: %s",
				m.Cmd, m.Code.String(), m.Hash.String(), m.Reason)
		case *wire.MsgNotFound:
			logging.Infof("Got not found response from remote:")
			for i, thing := range m.InvList {
				logging.Infof("\t%d) %s: %s", i, thing.Type, thing.Hash)
			}

		default:
			if m == nil {
				logging.Errorf("Got nil message")
			}
		}
	}
}

// this one seems kindof pointless?  could get ridf of it and let
// functions call WriteMessageWithEncodingN themselves...
func (s *SPVCon) outgoingMessageHandler() {
	for {
		msg := <-s.outMsgQueue
		if msg == nil {
			logging.Errorf("ERROR: nil message to outgoingMessageHandler\n")
			continue
		}
		n, err := wire.WriteMessageWithEncodingN(s.con, msg, s.localVersion,
			wire.BitcoinNet(s.Param.NetMagicBytes), wire.LatestEncoding)

		if err != nil {
			logging.Errorf("Write message error: %s", err.Error())
		}
		s.WBytes += uint64(n)
	}
}

// REORG TODO: how to detect reorgs and send them up to wallet layer

// HeaderHandler ...
func (s *SPVCon) HeaderHandler(m *wire.MsgHeaders) {
	moar, err := s.IngestHeaders(m)
	if err != nil {
		logging.Errorf("Header error: %s\n", err.Error())
	}
	// more to get? if so, ask for them and return
	if moar {
		err = s.AskForHeaders()
		if err != nil {
			logging.Errorf("AskForHeaders error: %s", err.Error())
		}
	}
}

// InvHandler ...
func (s *SPVCon) InvHandler(m *wire.MsgInv) {
	logging.Infof("got inv.  Contains:\n")
	for i, thing := range m.InvList {
		logging.Infof("\t%d)%s : %s",
			i, thing.Type.String(), thing.Hash.String())
		if thing.Type == wire.InvTypeBlock { // new block what to do?
			select {
			case <-s.inWaitState:
				// start getting headers
				logging.Infof("asking for headers due to inv block\n")
				err := s.AskForHeaders()
				if err != nil {
					logging.Errorf("AskForHeaders error: %s", err.Error())
				}
			default:
				// drop it as if its component particles had high thermal energies
				logging.Infof("inv block but ignoring; not synced\n")
			}
		}
	}
}

// PongBack ...
func (s *SPVCon) PongBack(nonce uint64) {
	mpong := wire.NewMsgPong(nonce)

	s.outMsgQueue <- mpong
	return
}
