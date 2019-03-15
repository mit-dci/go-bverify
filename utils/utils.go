package utils

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/adiabat/bech32"

	"github.com/btcsuite/btcd/txscript"

	"github.com/btcsuite/btcd/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil"
)

const APP_NAME = "B_Verify"

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
	return ""
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
	fmt.Printf("TX: %x\n", buf.Bytes())
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
