package main

import (
	"crypto/rand"
	"fmt"

	"github.com/mit-dci/go-bverify/server"
)

func main() {
	srv, _ := server.NewServer(":9100")
	srv.Run()
}
