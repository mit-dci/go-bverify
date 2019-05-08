package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/mit-dci/go-bverify/bitcoin/chainhash"

	"github.com/mit-dci/go-bverify/bitcoin/bech32"

	"github.com/mit-dci/go-bverify/bitcoin/txscript"

	"github.com/mit-dci/go-bverify/bitcoin/wire"

	"github.com/mit-dci/go-bverify/bitcoin/btcutil"
	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/logging"
)

const APP_NAME = "B_Verify"

var overrideClientDataDir string
var maidenHash []byte

// CloneByteSlice clones a byte slice and returns the clone
func CloneByteSlice(b []byte) []byte {
	clone := make([]byte, len(b))
	copy(clone[:], b[:])
	return clone
}

// Max returns the larger of x or y.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// GetBit gets the bit at index in a byte array. byte array: byte[0]|| byte[1] ||
// byte[2] || byte[3] index [0...7] [8...15] [16...23] [24...31]
func GetBit(b []byte, idx uint) bool {
	bitIdx := uint(idx % 8)
	byteIdx := (idx - bitIdx) / 8
	return (b[byteIdx] & (1 << (7 - bitIdx))) > 0
}

func DataDirectory() string {

	if runtime.GOOS == "windows" {
		return path.Join(os.Getenv("APPDATA"), APP_NAME)
	} else if runtime.GOOS == "darwin" {
		return path.Join(os.Getenv("HOME"), "Library", "Application Support", APP_NAME)
	} else if runtime.GOOS == "linux" {
		return path.Join(os.Getenv("HOME"), fmt.Sprintf(".%s", strings.ToLower(APP_NAME)))
	}
	return "."
}

func SetOverrideClientDataDirectory(dataDir string) {
	overrideClientDataDir = dataDir
}

func ClientDataDirectory() string {
	if overrideClientDataDir != "" {
		return overrideClientDataDir
	}

	if runtime.GOOS == "windows" {
		return path.Join(os.Getenv("APPDATA"), APP_NAME+" Client")
	} else if runtime.GOOS == "darwin" {
		return path.Join(os.Getenv("HOME"), "Library", "Application Support", APP_NAME+" Client")
	} else if runtime.GOOS == "linux" {
		return path.Join(os.Getenv("HOME"), fmt.Sprintf(".%s", strings.ToLower(APP_NAME+"_client")))
	}
	return "."
}

func KeyHashFromPkScript(pkscript []byte) []byte {
	// match p2wpkh
	if len(pkscript) == 22 && pkscript[0] == 0x00 && pkscript[1] == 0x14 {
		return pkscript[2:]
	}

	// match p2wsh
	if len(pkscript) == 34 && pkscript[0] == 0x00 && pkscript[1] == 0x20 {
		return pkscript[2:]
	}

	return nil
}

func KeyHashFromPubKey(pk *btcec.PublicKey) [20]byte {
	pkh := [20]byte{}
	copy(pkh[:], btcutil.Hash160(pk.SerializeCompressed()))
	return pkh
}

func PrintTx(tx *wire.MsgTx) {
	var buf bytes.Buffer

	tx.Serialize(&buf)
	logging.Debugf("TX: %x\n", buf.Bytes())
}

func DirectWPKHScriptFromPKH(pkh [20]byte) []byte {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0).AddData(pkh[:])
	b, _ := builder.Script()
	return b
}

func DirectWSHScriptFromSH(sh [32]byte) []byte {
	builder := txscript.NewScriptBuilder()
	builder.AddOp(txscript.OP_0).AddData(sh[:])
	b, _ := builder.Script()
	return b
}

func MaidenHash() []byte {
	return maidenHash
}

func DirectWSHScriptFromAddress(adr string) ([]byte, error) {
	var scriptHash [32]byte
	decoded, err := bech32.SegWitAddressDecode(adr)
	if err != nil {
		return []byte{}, err
	}
	copy(scriptHash[:], decoded[2:]) // skip version and pushdata byte returned by SegWitAddressDecode
	return DirectWSHScriptFromSH(scriptHash), nil
}

func DirectWPKHScriptFromAddress(adr string) ([]byte, error) {
	var pubkeyHash [20]byte
	decoded, err := bech32.SegWitAddressDecode(adr)
	if err != nil {
		return []byte{}, err
	}
	copy(pubkeyHash[:], decoded[2:]) // skip version and pushdata byte returned by SegWitAddressDecode
	return DirectWPKHScriptFromPKH(pubkeyHash), nil
}

func NextPowerOfTwo(n uint64) (e uint) {
	for ; (1 << e) < n; e++ {
	}
	return
}

// HashToBig converts a chainhash.Hash into a big.Int that can be used to
// perform math comparisons.
func HashToBig(hash *chainhash.Hash) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

func GetEnvOrDefault(evar string, def string) string {
	val := os.Getenv(evar)
	if val == "" {
		return def
	}
	return val
}

func init() {
	// The maidenHash is a fixed result of the following log statements being added to the
	// server upon first startup:
	//
	// srv.RegisterLogID([32]byte{}, [33]byte{})
	// logHash := fastsha256.Sum256([]byte("Maiden commitment for b_verify"))
	// srv.RegisterLogStatement([32]byte{}, 0, logHash[:])
	// srv.Commit()
	//
	// Every commitment chain of b_verify servers will start with this hash - which
	// makes it very easy to recognize there were no prior commitments
	maidenHash, _ = hex.DecodeString("523e59cfc5235b915dc89de188d87449453b083a8b7d97c1ee64d875da403361")
	overrideClientDataDir = ""
}
