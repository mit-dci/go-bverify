package uspv

import (
	"path"

	"github.com/mit-dci/go-bverify/bitcoin/coinparam"
	"github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/utils"
)

func (s *SPVCon) Start(params *coinparam.Params) error {

	s.Param = params

	s.inMsgQueue = make(chan wire.Message)
	s.outMsgQueue = make(chan wire.Message)
	s.syncHeight = 0

	headerFilePath := path.Join(utils.ClientDataDirectory(), "header.bin")
	// open header file
	err := s.openHeaderFile(headerFilePath)
	if err != nil {
		return err
	}

	err = s.Connect()
	if err != nil {
		logging.Errorf("Can't connect to host\n")
		return err
	} else {
		logging.Debugf("Connected to host\n")
	}

	return nil
}
