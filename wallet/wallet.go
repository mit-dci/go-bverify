package wallet

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/adiabat/bech32"

	"github.com/btcsuite/btcd/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"github.com/tidwall/buntdb"

	"github.com/btcsuite/btcutil"

	"github.com/mit-dci/go-bverify/utils"

	"path"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/rpcclient"
)

type Wallet struct {
	// The key to commit to the blockchain with
	privateKey  *btcec.PrivateKey
	pubKey      *btcec.PublicKey
	pubKeyHash  [20]byte
	utxos       []Utxo
	db          *buntdb.DB
	rpcClient   *rpcclient.Client
	activeChain ChainIndex
}

func NewWallet() (*Wallet, error) {
	var err error
	w := new(Wallet)
	genesis, _ := chainhash.NewHashFromStr("0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206")
	w.activeChain = ChainIndex{genesis}
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

	fmt.Printf("Wallet initialized. At height %d - Balance %d - Address is: %s\n", len(w.activeChain), w.Balance(), w.address())

	go w.BlockLoop()

	// TODO: make this configurable
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18443",
		User:         "bverify",
		Pass:         "bverify",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	w.rpcClient, err = rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Wallet) address() string {
	adr, _ := bech32.SegWitV0Encode("bcrt", w.pubKeyHash[:])
	return adr
}

func (w *Wallet) loadStuff() {
	w.db.View(func(tx *buntdb.Tx) error {
		tx.AscendRange("", "utxo-", "utxp-", func(key, value string) bool {
			w.utxos = append(w.utxos, UtxoFromBytes([]byte(value)))
			return true
		})
		return nil
	})
}

func (w *Wallet) BlockLoop() {
	for {
		time.Sleep(time.Second * 5)
		bestHash, err := w.rpcClient.GetBestBlockHash()
		if err != nil {
			fmt.Printf("Error getting best blockhash: %s\n", err.Error())
			continue
		}

		if bestHash.IsEqual(w.activeChain[len(w.activeChain)-1]) {
			continue
		}

		hash, _ := chainhash.NewHash(bestHash.CloneBytes())
		pendingBlockHashes := make([]*chainhash.Hash, 0)
		startIndex := 0
		for {
			header, err := w.rpcClient.GetBlockHeader(hash)
			if err != nil {
				fmt.Printf("Error getting block header: %s\n", err.Error())
				continue
			}

			newHash, _ := chainhash.NewHash(hash.CloneBytes())
			pendingBlockHashes = append([]*chainhash.Hash{newHash}, pendingBlockHashes...)
			hash = &header.PrevBlock
			idx := w.activeChain.FindBlock(&header.PrevBlock)
			if idx > -1 {
				// We found a way to connect to our activeChain
				// Remove all blocks after idx, if any
				newChain := w.activeChain[:idx+1]
				newChain = append(newChain, pendingBlockHashes...)
				w.activeChain = newChain
				startIndex = idx
				break
			}
		}

		for _, hash := range w.activeChain[startIndex:] {
			block, err := w.rpcClient.GetBlock(hash)
			if err != nil {
				fmt.Printf("Error getting block: %s\n", err.Error())
				continue
			}

			err = w.processBlock(block)
			if err != nil {
				fmt.Printf("Error processing block: %s\n", err.Error())
				continue
			}
		}
		w.persistChainState()
	}
}

func (w *Wallet) AddInputsAndChange(tx *wire.MsgTx, totalValueNeeded uint64) error {
	valueAdded := uint64(0)
	utxosToAdd := []Utxo{}
	for _, utxo := range w.utxos {
		utxosToAdd = append(utxosToAdd, utxo)
		valueAdded += utxo.Value
		if valueAdded > totalValueNeeded {
			break
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
	for _, tx := range block.Transactions {
		w.processTransaction(tx)
	}

	fmt.Printf("New block processed. Our balance is now %d", w.Balance())

	return nil
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
	w.utxos = append(w.utxos, utxo)
	err := w.db.Update(func(dtx *buntdb.Tx) error {
		key := fmt.Sprintf("utxo-%s-%d", utxo.TxHash.String(), utxo.Outpoint)
		_, _, err := dtx.Set(key, string(utxo.Bytes()), nil)
		return err
	})
	if err != nil {
		fmt.Printf("[Wallet] Error registering utxo: %s", err.Error())
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

func (w *Wallet) Commit(commitment []byte) ([]byte, error) {
	tx := wire.NewMsgTx(1)
	neededInputs := uint64(1000) // minfee

	var scriptBuf bytes.Buffer
	scriptBuf.WriteByte(0x6A) // OP_RETURN
	scriptBuf.WriteByte(byte(len(commitment)))
	scriptBuf.Write(commitment)
	tx.AddTxOut(wire.NewTxOut(0, scriptBuf.Bytes()))

	w.AddInputsAndChange(tx, neededInputs)

	txid, err := w.rpcClient.SendRawTransaction(tx, false)
	return txid.CloneBytes(), err
}
