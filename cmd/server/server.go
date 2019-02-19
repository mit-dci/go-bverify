package main

import (
	"crypto/rand"
	"fmt"

	"github.com/mit-dci/go-bverify/server"
)

func main2() {
	srv, _ := server.NewServer(":9100")
	srv.Run()
}

func main() {

	fmt.Printf("TestLogAndCommit\n")

	srv, err := server.NewServer("")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Registering logs...\n")
	// Register 100 logs
	logId := [32]byte{}
	pubKey := [33]byte{}
	for i := 0; i < 100; i++ {
		rand.Read(logId[:])
		rand.Read(pubKey[:])
		err = srv.RegisterLogID(logId, pubKey)
		if err != nil {
			fmt.Println(err)
			return
		}
		if i == 0 { // test this only once. Expect log 0 as first log!
			err = srv.RegisterLogStatement(logId, 1, []byte("Hello world"))
			if err == nil {
				fmt.Println("Expected error when inserting idx != 0 as first statement, got none")
			}
		}

		err = srv.RegisterLogStatement(logId, 0, []byte("Hello world"))
		if err != nil {
			fmt.Println(err)
			return
		}

	}

	fmt.Printf("Calling fullmpt.commitment\n")

	// Trigger actual commitment. Should call the SendProofs() on all mock logprocessors
	// which is tested by the mock framework on ctrl.Finish()
	srv.Commit()

	fmt.Printf("Done with commitment")

	/*

		// This should trigger an error because log 0 was already inserted.
		// The next sequence is supposed to be 1
		err = srv.RegisterLogStatement(logId, 0, []byte("Hello world"))
		if err == nil {
			fmt.Println("Expected error when inserting duplicate log sequence, got none")
		}

		srv.RegisterLogStatement(logId, 1, []byte("Hello world2"))

		comm = srv.fullmpt.Commitment()
		if bytes.Equal(comm, srv.lastCommitment[:]) {
			fmt.Println("Commitment hasn't changed after ")
		}
		srv.Commit()

		if !bytes.Equal(comm, srv.lastCommitment[:]) {
			fmt.Println("lastCommitment was not updated after Commit()")
		}

		// Calling commit again without changes - it shouldn't do anything.
		// If it triggers SendProofs() again, the count will be off and it will
		// trigger an error in the mock framework
		srv.Commit()*/
}
