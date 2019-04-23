package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/mit-dci/go-bverify/crypto/btcec"
	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/crypto/sig64"
	"github.com/mit-dci/go-bverify/wire"
)

func main() {
	var err error

	// Create two keys
	var key [32]byte
	rand.Read(key[:])
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), key[:])
	rand.Read(key[:])
	priv2, pub2 := btcec.PrivKeyFromBytes(btcec.S256(), key[:])

	var pub33 [33]byte
	copy(pub33[:], pub.SerializeCompressed())

	// Create the log
	statement := fastsha256.Sum256([]byte("This is just a test"))
	l := wire.NewSignedCreateLogStatement(pub33, statement[:])
	logId := fastsha256.Sum256(l.CreateStatement.Bytes())

	// Create various log statements we use to challenge
	sal1, v1, r1, s1, ph1 := CreateSignedStatement(priv, logId, 1, []byte("Yeah this is just a test"))
	sal2, v2, r2, s2, ph2 := CreateSignedStatement(priv2, logId, 2, []byte("This is just a 2nd test"))
	sal3, v3, r3, s3, ph3 := CreateSignedStatement(priv, logId, 1, []byte("3rd test"))

	// Replace the template values to create the unit-test script
	dat, err := ioutil.ReadFile("penalty_template.js")
	if err != nil {
		panic(err)
	}

	sourceCode := string(dat)

	sourceCode = strings.ReplaceAll(sourceCode, "%sal1%", Encode(sal1.Bytes()))
	sourceCode = strings.ReplaceAll(sourceCode, "%sal2%", Encode(sal2.Bytes()))
	sourceCode = strings.ReplaceAll(sourceCode, "%sal3%", Encode(sal3.Bytes()))

	sourceCode = strings.ReplaceAll(sourceCode, "%pub1%", Encode(pub.SerializeUncompressed()))
	sourceCode = strings.ReplaceAll(sourceCode, "%pub2%", Encode(pub2.SerializeUncompressed()))

	sourceCode = strings.ReplaceAll(sourceCode, "%v1%", fmt.Sprintf("%d", v1))
	sourceCode = strings.ReplaceAll(sourceCode, "%v2%", fmt.Sprintf("%d", v2))
	sourceCode = strings.ReplaceAll(sourceCode, "%v3%", fmt.Sprintf("%d", v3))

	sourceCode = strings.ReplaceAll(sourceCode, "%r1%", Encode(r1))
	sourceCode = strings.ReplaceAll(sourceCode, "%r2%", Encode(r2))
	sourceCode = strings.ReplaceAll(sourceCode, "%r3%", Encode(r3))

	sourceCode = strings.ReplaceAll(sourceCode, "%s1%", Encode(s1))
	sourceCode = strings.ReplaceAll(sourceCode, "%s2%", Encode(s2))
	sourceCode = strings.ReplaceAll(sourceCode, "%s3%", Encode(s3))

	sourceCode = strings.ReplaceAll(sourceCode, "%ph1%", Encode(ph1))
	sourceCode = strings.ReplaceAll(sourceCode, "%ph2%", Encode(ph2))
	sourceCode = strings.ReplaceAll(sourceCode, "%ph3%", Encode(ph3))

	sourceCode = strings.ReplaceAll(sourceCode, "%cls%", Encode(l.CreateStatement.Bytes()))

	err = ioutil.WriteFile("penalty_generated.js", []byte(sourceCode), 0644)
	if err != nil {
		panic(err)
	}

}

func CreateSignedStatement(priv *btcec.PrivateKey, logId [32]byte, idx uint64, statement []byte) (sal *wire.SignedLogStatement, v uint, r, s, proofHash []byte) {
	shash := fastsha256.Sum256(statement)
	sal = wire.NewSignedLogStatement(idx, logId, shash[:])
	signHash := fastsha256.Sum256(sal.Statement.Bytes())
	ecsig, err := priv.Sign(signHash[:])
	if err != nil {
		panic(err)
	}
	sal.Signature, _ = sig64.SigCompress(ecsig.Serialize())
	msgHash := fastsha256.Sum256(sal.Bytes())

	sig, err := btcec.SignCompact(btcec.S256(), priv, signHash[:], false)
	if err != nil {
		panic(err)
	}

	v = uint(sig[0])
	r = make([]byte, 32)
	copy(r, sig[1:33])
	s = make([]byte, 32)
	copy(s, sig[33:65])

	proofHash32 := fastsha256.Sum256(append(logId[:], msgHash[:]...))
	proofHash = proofHash32[:]

	return
}

func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}
