package server

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mit-dci/go-bverify/server/mocks"
)

func TestRegisterGetLogKey(t *testing.T) {
	srv, err := NewServer("")
	if err != nil {
		t.Error(err)
		return
	}

	logId, _ := hex.DecodeString("f5b9fb71ddcc3f691f7df5f0b6ff8391963a506e2cec989968e6299c743176a7")
	pubKey, _ := hex.DecodeString("449fd2597bdee83564de6f2659b2705a40474071c354b9e02efd07594ab1afb6d4")
	logId32 := [32]byte{}
	pubKey33 := [33]byte{}
	copy(logId32[:], logId)
	copy(pubKey33[:], pubKey)

	err = srv.RegisterLogID(logId32, pubKey33)
	if err != nil {
		t.Error(err)
		return
	}

	pk, err := srv.GetPubKeyForLogID(logId32)
	if err != nil {
		t.Error(err)
		return
	}

	if !bytes.Equal(pk[:], pubKey33[:]) {
		t.Error("GetPubKey did not return the registered key!")
	}

	// Register duplicate - should error out
	err = srv.RegisterLogID(logId32, pubKey33)
	if err == nil {
		t.Error("Registering a duplicate logID did not return an error.")
		return
	}

	// Request non existent log - should error out
	_, err = srv.GetPubKeyForLogID([32]byte{})
	if err == nil {
		t.Error("Registering a duplicate logID did not return an error.")
		return
	}
}

func TestLogAndCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv, err := NewServer("")
	if err != nil {
		t.Error(err)
		return
	}

	// Register 100 logs
	logId := [32]byte{}
	pubKey := [33]byte{}
	for i := 0; i < 100; i++ {
		rand.Read(logId[:])
		rand.Read(pubKey[:])
		err = srv.RegisterLogID(logId, pubKey)
		if err != nil {
			t.Error(err)
			return
		}
		if i == 0 { // test this only once. Expect log 0 as first log!
			err = srv.RegisterLogStatement(logId, 1, []byte("Hello world"))
			if err == nil {
				t.Error("Expected error when inserting idx != 0 as first statement, got none")
			}
		}

		err = srv.RegisterLogStatement(logId, 0, []byte("Hello world"))
		if err != nil {
			t.Error(err)
			return
		}
		m := mocks.NewMockLogProcessor(ctrl)
		srv.processors = append(srv.processors, m)

		// We should get two and only two calls to SendProofs
		// The last commit should not call SendProofs because nothing changed.
		m.EXPECT().SendProofs(gomock.Any()).Return(nil).MinTimes(2).MaxTimes(2)
	}

	// set comm to the expected commitment
	comm := srv.fullmpt.Commitment()

	if bytes.Equal(comm, srv.lastCommitment[:]) {
		t.Error("Unexpected equality between the MPT commitment and lastCommitment (which should be an zeroes-only byte array)")
	}

	// Trigger actual commitment. Should call the SendProofs() on all mock logprocessors
	// which is tested by the mock framework on ctrl.Finish()
	srv.Commit()

	if !bytes.Equal(comm, srv.lastCommitment[:]) {
		t.Error("lastCommitment was not updated after Commit()")
	}

	// This should trigger an error because log 0 was already inserted.
	// The next sequence is supposed to be 1
	err = srv.RegisterLogStatement(logId, 0, []byte("Hello world"))
	if err == nil {
		t.Error("Expected error when inserting duplicate log sequence, got none")
	}

	srv.RegisterLogStatement(logId, 1, []byte("Hello world2"))

	comm = srv.fullmpt.Commitment()
	if bytes.Equal(comm, srv.lastCommitment[:]) {
		t.Error("Commitment hasn't changed after ")
	}
	srv.Commit()

	if !bytes.Equal(comm, srv.lastCommitment[:]) {
		t.Error("lastCommitment was not updated after Commit()")
	}

	// Calling commit again without changes - it shouldn't do anything.
	// If it triggers SendProofs() again, the count will be off and it will
	// trigger an error in the mock framework
	srv.Commit()
}

func TestServerConnectivity(t *testing.T) {
	// Use weird port for test, not the actual
	// runtime default one
	srv, err := NewServer(":56199")
	if err != nil {
		t.Error(err)
		return
	}

	srv2, _ := NewServer("garbage")
	err = srv2.Run()
	if err == nil {
		t.Error("Expected invalid port argument to return error, but it didn't")
		return
	}

	go func() {
		err := srv.Run()
		if err != nil {
			t.Error(err)
		}
	}()

	<-srv.ready

	srv2, _ = NewServer(":56199")
	err = srv2.Run()
	if err == nil {
		t.Error("Expected using a port twice to throw an error (already in use) but it did not")
		return
	}

	conn, err := net.Dial("tcp", "127.0.0.1:56199")
	if err != nil {
		t.Error(err)
	}

	conn.Close()
	srv.Stop()

}
