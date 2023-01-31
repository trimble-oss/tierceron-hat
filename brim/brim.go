package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"time"

	"github.com/mrjrieke/hat/cap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	protocol = "unix"
	sockAddr = "/tmp/echo.sock"
)

func main() {

	pense := "Maelstrom"
	penseArray := sha256.Sum256([]byte(pense))
	penseSum := hex.EncodeToString(penseArray[:])

	cap.TapWriter(penseSum)

	conn, err := grpc.Dial("127.0.0.1:1534", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := cap.NewCapClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Pense(ctx, &cap.PenseRequest{Pense: pense, PenseIndex: "FooIndex"})
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	log.Printf("I sense: %s", r.GetPense())
}
