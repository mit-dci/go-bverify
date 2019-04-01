package wallet

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/mit-dci/go-bverify/bitcoin/bech32"
	"github.com/mit-dci/go-bverify/bitcoin/btcutil"
	"github.com/mit-dci/go-bverify/bitcoin/chaincfg"
	"github.com/mit-dci/go-bverify/bitcoin/chainhash"
	"github.com/mit-dci/go-bverify/bitcoin/rpcclient"
	"github.com/mit-dci/go-bverify/bitcoin/txscript"
	"github.com/mit-dci/go-bverify/bitcoin/wire"
	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/utils"
	"github.com/tidwall/buntdb"
)

const MINOUTPUT uint64 = 1000

type Wallet struct {
	// The key to commit to the blockchain with
	privateKey         *btcec.PrivateKey
	pubKey             *btcec.PublicKey
	pubKeyHash         [20]byte
	utxos              []Utxo
	db                 *buntdb.DB
	rpcClient          *rpcclient.Client
	activeChain        ChainIndex
	newBlockListeners  []chan *wire.MsgBlock
	params             *chaincfg.Params
	synced             bool
	lastCommitmentTxId []byte
}

func NewWallet(params *chaincfg.Params, rescanBlocks int) (*Wallet, error) {
	var err error
	w := new(Wallet)
	w.params = params
	w.activeChain = ChainIndex{w.params.GenesisHash}
	keyFile := path.Join(utils.DataDirectory(), "privkey.hex")
	key32 := [32]byte{}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		rand.Read(key32[:])
		ioutil.WriteFile(keyFile, key32[:], 0600)
	} else if err != nil {
		return nil, err
	} else {
		key, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		copy(key32[:], key)
	}

	w.privateKey, w.pubKey = btcec.PrivKeyFromBytes(btcec.S256(), key32[:])
	copy(w.pubKeyHash[:], btcutil.Hash160(w.pubKey.SerializeCompressed()))

	w.db, err = buntdb.Open(path.Join(utils.DataDirectory(), "wallet.db"))
	if err != nil {
		return nil, err
	}

	w.loadStuff()
	w.readChainState()

	if rescanBlocks > 0 {
		if len(w.activeChain) < rescanBlocks {
			rescanBlocks = len(w.activeChain)
		}
		w.activeChain = w.activeChain[:len(w.activeChain)-rescanBlocks]
	}

	w.newBlockListeners = make([]chan *wire.MsgBlock, 0)
	logging.Debugf("Wallet initialized. At height %d - Balance %d - Address is: %s\n", len(w.activeChain), w.Balance(), w.address())
	w.synced = false
	go w.BlockLoop()

	// TODO: make this configurable
	connCfg := &rpcclient.ConnConfig{
		Host:         utils.GetEnvOrDefault("BITCOINRPC", "localhost:18443"),
		User:         utils.GetEnvOrDefault("BITCOINRPCUSER", "bverify"),
		Pass:         utils.GetEnvOrDefault("BITCOINRPCPASSWORD", "bverify"),
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	w.rpcClient, err = rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Wallet) AddNewBlockListener(blockChan chan *wire.MsgBlock) {
	w.newBlockListeners = append(w.newBlockListeners, blockChan)
}

func (w *Wallet) address() string {
	adr, _ := bech32.SegWitV0Encode(w.params.Bech32Prefix, w.pubKeyHash[:])
	return adr
}

func (w *Wallet) loadStuff() error {
	err := w.db.View(func(tx *buntdb.Tx) error {
		tx.AscendRange("", "utxo-", "utxp-", func(key, value string) bool {
			w.utxos = append(w.utxos, UtxoFromBytes([]byte(value)))
			return true
		})

		txidString, err := tx.Get("lastcommit-txid", false)
		if err != nil {
			return err
		}
		w.lastCommitmentTxId = []byte(txidString)
		return nil
	})
	return err
}

func (w *Wallet) HeightOfBlock(blockHash chainhash.Hash) int {
	return w.activeChain.FindBlock(&blockHash)
}

func (w *Wallet) BlockLoop() {
	for {
		time.Sleep(time.Second * 5)
		w.synced = false
		bestHash, err := w.rpcClient.GetBestBlockHash()
		if err != nil {
			logging.Errorf("Error getting best blockhash: %s\n", err.Error())
			continue
		}

		if bestHash.IsEqual(w.activeChain[len(w.activeChain)-1]) {
			w.synced = true
			continue
		}

		logging.Debugf("Found new best hash, trying to attach to known chain")

		hash, _ := chainhash.NewHash(bestHash.CloneBytes())
		pendingBlockHashes := make([]*chainhash.Hash, 0)
		startIndex := 0
		for {
			header, err := w.rpcClient.GetBlockHeader(hash)
			if err != nil {
				logging.Errorf("Error getting block header: %s\n", err.Error())
				continue
			}

			newHash, _ := chainhash.NewHash(hash.CloneBytes())
			pendingBlockHashes = append([]*chainhash.Hash{newHash}, pendingBlockHashes...)
			hash = &header.PrevBlock
			idx := w.activeChain.FindBlock(&header.PrevBlock)
			if idx > -1 {
				idx++
				// We found a way to connect to our activeChain
				// Remove all blocks after idx, if any
				newChain := w.activeChain[:idx]
				newChain = append(newChain, pendingBlockHashes...)
				w.activeChain = newChain
				startIndex = idx
				break
			}
			if len(pendingBlockHashes)%1000 == 0 {
				logging.Debugf("Pending hashes: %d", len(pendingBlockHashes))
			}
		}

		for _, hash := range w.activeChain[startIndex:] {
			block, err := w.rpcClient.GetBlock(hash)
			if err != nil {
				logging.Errorf("Error getting block: %s\n", err.Error())
				continue
			}

			err = w.processBlock(block)
			if err != nil {
				logging.Errorf("Error processing block: %s\n", err.Error())
				continue
			}
		}

		w.synced = true
		w.persistChainState()
	}
}

func (w *Wallet) IsSynced() bool {
	return w.synced
}

func (w *Wallet) AddInputsAndChange(tx *wire.MsgTx, totalValueNeeded uint64) error {
	valueAdded := uint64(0)
	utxosToAdd := []Utxo{}
	// This may seem weird, but we're adding _all_ inputs always. Why? Because we
	// want to prevent polluting the UTXO set. If someone donates to our server's
	// wallet (or we do ourselves), we immediately consolidate the coins in the
	// next commitment. The OP_RETURN will still be the 0th output, and the previous
	// commitment should be the first txin to ensure we form a chain.
	//
	// So, first find the index of the last commitment's TX output 1

	lastOutputIdx := -1
	for i, utxo := range w.utxos {
		if utxo.Outpoint == 1 && bytes.Equal(utxo.TxHash[:], w.lastCommitmentTxId) {
			lastOutputIdx = i
			break
		}
	}

	if lastOutputIdx == -1 {
		logging.Warnf("Did not find last commitment's output in the UTXOs. This is fine when we are a fresh server. Otherwise, there's something wrong")
	} else {
		valueAdded += w.utxos[lastOutputIdx].Value
		utxosToAdd = append(utxosToAdd, w.utxos[lastOutputIdx])
	}

	for i, utxo := range w.utxos {
		if i != lastOutputIdx {
			valueAdded += utxo.Value
			utxosToAdd = append(utxosToAdd, utxo)
		}
	}

	if valueAdded < totalValueNeeded {
		return fmt.Errorf("Insufficient balance")
	}

	for _, utxo := range utxosToAdd {
		tx.AddTxIn(wire.NewTxIn(&wire.OutPoint{utxo.TxHash, utxo.Outpoint}, nil, nil))
	}

	// Add change output when there's more than dust left, otherwise give to miners
	if valueAdded-totalValueNeeded > MINOUTPUT {
		tx.AddTxOut(wire.NewTxOut(int64(valueAdded-totalValueNeeded), utils.DirectWPKHScriptFromPKH(w.pubKeyHash)))
	}

	return nil
}

func (w *Wallet) Balance() uint64 {
	value := uint64(0)
	for _, u := range w.utxos {
		value += u.Value
	}
	return value
}

func (w *Wallet) processBlock(block *wire.MsgBlock) error {
	for i, nbl := range w.newBlockListeners {
		select {
		case nbl <- block:

		default:

		}
	}

	balBefore := w.Balance()
	for _, tx := range block.Transactions {
		w.processTransaction(tx)
	}
	balAfter := w.Balance()
	if balAfter != balBefore {
		logging.Debugf("Our balance is now %d", w.Balance())
	}
	return nil
}

func (w *Wallet) Height() int {
	return len(w.activeChain) - 1
}

func (w *Wallet) processTransaction(tx *wire.MsgTx) {
	for i, out := range tx.TxOut {
		keyHash := utils.KeyHashFromPkScript(out.PkScript)
		if bytes.Equal(keyHash, w.pubKeyHash[:]) {
			w.registerUtxo(Utxo{
				TxHash:   tx.TxHash(),
				Outpoint: uint32(i),
				Value:    uint64(out.Value),
				PkScript: out.PkScript,
			})
		}
	}

	w.markTxInputsAsSpent(tx)
}

func (w *Wallet) markTxInputsAsSpent(tx *wire.MsgTx) {
	for _, in := range tx.TxIn {
		removeIndex := -1
		for j, out := range w.utxos {
			if in.PreviousOutPoint.Hash.IsEqual(&out.TxHash) && in.PreviousOutPoint.Index == out.Outpoint {
				// Spent!
				removeIndex = j
				break
			}
		}
		if removeIndex >= 0 {
			w.db.Update(func(dtx *buntdb.Tx) error {
				key := fmt.Sprintf("utxo-%s-%d", w.utxos[removeIndex].TxHash.String(), w.utxos[removeIndex].Outpoint)
				_, err := dtx.Delete(key)
				return err
			})
			w.utxos = append(w.utxos[:removeIndex], w.utxos[removeIndex+1:]...)

		}
	}
}

func (w *Wallet) registerUtxo(utxo Utxo) {
	alreadyAtIdx := -1
	for i, u := range w.utxos {
		if utxo.TxHash.IsEqual(&u.TxHash) && utxo.Outpoint == u.Outpoint {
			alreadyAtIdx = i
		}
	}

	if alreadyAtIdx >= 0 {
		w.utxos[alreadyAtIdx] = utxo
	} else {
		w.utxos = append(w.utxos, utxo)
	}

	err := w.db.Update(func(dtx *buntdb.Tx) error {
		key := fmt.Sprintf("utxo-%s-%d", utxo.TxHash.String(), utxo.Outpoint)
		_, _, err := dtx.Set(key, string(utxo.Bytes()), nil)
		return err
	})
	if err != nil {
		logging.Errorf("[Wallet] Error registering utxo: %s", err.Error())
	}
}

func (w *Wallet) persistChainState() {
	var buf bytes.Buffer
	for _, h := range w.activeChain {
		buf.Write(h[:])
	}
	ioutil.WriteFile(path.Join(utils.DataDirectory(), "chainstate.hex"), buf.Bytes(), 0644)
}

func (w *Wallet) readChainState() {
	b, err := ioutil.ReadFile(path.Join(utils.DataDirectory(), "chainstate.hex"))
	if err != nil {
		return
	}
	readIndex := ChainIndex{}
	buf := bytes.NewBuffer(b)
	hash := make([]byte, 32)
	for {
		i, err := buf.Read(hash)
		if i == 32 && err == nil {
			ch, err := chainhash.NewHash(hash)
			if err == nil {
				readIndex = append(readIndex, ch)
			} else {
				break
			}
		} else {
			break
		}
	}
	w.activeChain = readIndex
}

func (w *Wallet) FindUtxoFromTxIn(txi *wire.TxIn) (Utxo, error) {
	for _, out := range w.utxos {
		if txi.PreviousOutPoint.Hash.IsEqual(&out.TxHash) && txi.PreviousOutPoint.Index == out.Outpoint {
			return out, nil
		}
	}
	return Utxo{}, fmt.Errorf("Utxo not found")
}

func (w *Wallet) SignMyInputs(tx *wire.MsgTx) error {
	// generate tx-wide hashCache for segwit stuff
	// might not be needed (non-witness) but make it anyway
	hCache := txscript.NewTxSigHashes(tx)
	witStash := make([][][]byte, len(tx.TxIn))
	for i, txi := range tx.TxIn {
		utxo, err := w.FindUtxoFromTxIn(txi)
		if err != nil {
			continue
		}

		logging.Debugf("Signing input [%s / %d] with script [%x] and value [%d]", utxo.TxHash.String(), utxo.Outpoint, utxo.PkScript, utxo.Value)

		witStash[i], err = txscript.WitnessSignature(tx, hCache, i,
			int64(utxo.Value), utxo.PkScript, txscript.SigHashAll, w.privateKey, true)
		if err != nil {
			return err
		}
	}

	// swap sigs into sigScripts in txins
	for i, txin := range tx.TxIn {
		if witStash[i] != nil {
			txin.Witness = witStash[i]
			txin.SignatureScript = nil
		}
	}

	return nil
}

func (w *Wallet) Commit(commitment []byte) (*chainhash.Hash, []byte, error) {
	tx := wire.NewMsgTx(1)
	neededInputs := uint64(1000) // minfee

	tx.AddTxOut(wire.NewTxOut(0, append([]byte{0x6A, byte(len(commitment))}, commitment...)))

	err := w.AddInputsAndChange(tx, neededInputs)
	if err != nil {
		return nil, nil, err
	}

	err = w.SignMyInputs(tx)
	if err != nil {
		return nil, nil, err
	}

	txid, err := w.rpcClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, nil, err
	}
	var buf bytes.Buffer
	tx.Serialize(&buf)

	w.lastCommitmentTxId = txid[:]
	err = w.db.Update(func(dtx *buntdb.Tx) error {
		_, _, err := dtx.Set("lastcommit-txid", string(txid[:]), nil)
		return err
	})
	if err != nil {
		logging.Errorf("[Wallet] Error saving lastcommit txid: %s", err.Error())
	}

	return txid, buf.Bytes(), nil
}
