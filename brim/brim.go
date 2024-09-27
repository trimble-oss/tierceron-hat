package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"math/rand"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	"github.com/trimble-oss/tierceron-hat/cap/tap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const penseDir = "/tmp/trccarrier/"

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func penseQuery(pense string) {
	penseCode := randomString(7 + rand.Intn(7))
	penseArray := sha256.Sum256([]byte(penseCode))
	penseSum := hex.EncodeToString(penseArray[:])

	conn, err := grpc.Dial("127.0.0.1:1534", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := cap.NewCapClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var r *cap.PenseReply
	retry := 0

	for {
		_, err := c.Pense(ctx, &cap.PenseRequest{Pense: "", PenseIndex: ""})
		if err != nil {
			st, ok := status.FromError(err)

			if ok && (retry < 5) && st.Code() == codes.Unavailable {
				retry = retry + 1
				continue
			} else {
				log.Fatalf("did not connect: %v", err)
			}
		} else {
			break
		}
	}

	eyeMap, err := tap.TapWriter(penseDir, penseSum)
	if err != nil {
		log.Fatalf("Failure to communicate: %v", err)
	}
	log.Printf("%v", eyeMap)

	retry = 0

	for {
		r, err = c.Pense(ctx, &cap.PenseRequest{Pense: penseCode, PenseIndex: pense})
		if err != nil {
			st, ok := status.FromError(err)

			if ok && (retry < 5) && st.Code() == codes.Unavailable {
				retry = retry + 1
				continue
			} else {
				log.Fatalf("did not connect: %v", err)
			}
		} else {
			break
		}
	}

	log.Println(pense, r.GetPense())
}

// The original hat brim...
func main() {
	penseQuery("I think")
	penseQuery("It is not enough to have a good mind.")
}
