package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/mrjrieke/hat/cap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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

	_, featherErr := cap.FeatherWriter("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", penseSum)
	if featherErr != nil {
		log.Fatalf("Failed to feather writer: %v", featherErr)
	}

	conn, err := grpc.Dial("127.0.0.1:1534", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := cap.NewCapClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.Pense(ctx, &cap.PenseRequest{Pense: penseCode, PenseIndex: pense})
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	// log.Println(pense, r.GetPense())
}

var modeCtlTrail []string = []string{"I", "wa", "a", "nde", "er", "thro", "ough", "the", "e", "lo", "o", "vly", "y", "wo", "ods", "I", "i", "wa", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "an", "der", "through", "the", "woods."}

var penses []string = []string{"I think", "It is not enough to have a good mind.", "Ponder"}

func main() {
	for {
		if featherMode, featherErr := cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_FLAP, "HelloWorld"); featherErr == nil && strings.HasPrefix(featherMode, cap.MODE_GAZE) {
			for i, modeCtl := range modeCtlTrail {
				penseQuery(penses[i%3]) // Random activities...
				flapMode := cap.MODE_FLAP + "_" + modeCtl
				ctlFlapMode := flapMode
				var err error = errors.New("init")
				fmt.Printf("%s.", modeCtl)

				for {
					if err == nil && flapMode != ctlFlapMode {
						// Glide, etc...
						break
					} else {
						callFlap := flapMode
						if err == nil {
							time.Sleep(200 * time.Millisecond)
						} else {
							if err.Error() != "init" {
								fmt.Println("Waiting...")
								time.Sleep(1 * time.Second)
							}
						}
						ctlFlapMode, err = cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", callFlap, "HelloWorld")
					}
				}
			}
			cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_PERCH, "HelloWorld")
		} else {
			fmt.Println("Waiting...")
			time.Sleep(1 * time.Second)
		}
	}
}
