package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

var interruptChan chan os.Signal = make(chan os.Signal)
var twoHundredMilliInterruptTicker *time.Ticker = time.NewTicker(200 * time.Millisecond)
var multiSecondInterruptTicker *time.Ticker = time.NewTicker(time.Second)

func acceptRemote(int, string) bool {
	interruptFun(multiSecondInterruptTicker)
	return true
}

func interruptFun(tickerInterrupt *time.Ticker) {
	select {
	case <-interruptChan:
		cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_PERCH, "HelloWorld", true, nil)
		os.Exit(1)
	case <-tickerInterrupt.C:
	}
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
	var ic chan os.Signal = make(chan os.Signal)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		x := <-ic
		interruptChan <- x
	}()

	for {
	perching:
		if featherMode, featherErr := cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_FLAP, "HelloWorld", false, acceptRemote); featherErr == nil && strings.HasPrefix(featherMode, cap.MODE_GAZE) {
			fmt.Println("Fly away!")

			for i, modeCtl := range modeCtlTrail {
				penseQuery(penses[i%3]) // Random activities...
				flapMode := cap.MODE_FLAP + "_" + modeCtl
				ctlFlapMode := flapMode
				var err error = errors.New("init")
				fmt.Printf("%s.", modeCtl)

				for {
					if err == nil && ctlFlapMode == cap.MODE_PERCH {
						// Acknowledge perching...
						cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_PERCH, "HelloWorld", true, acceptRemote)
						ctlFlapMode = cap.MODE_PERCH
						goto perching
					}

					if err == nil && flapMode != ctlFlapMode {
						// Flap, Gaze, etc...
						interruptFun(twoHundredMilliInterruptTicker)
						break
					} else {
						callFlap := flapMode
						if err == nil {
							interruptFun(twoHundredMilliInterruptTicker)
						} else {
							if err.Error() != "init" {
								fmt.Printf("\nWaiting...\n")
								interruptFun(multiSecondInterruptTicker)
							}
						}
						ctlFlapMode, err = cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", callFlap, "HelloWorld", true, acceptRemote)
					}
				}
			}
			fmt.Printf("\nGliding....\n")
			cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_GLIDE, "HelloWorld", true, acceptRemote)
		} else {
			fmt.Printf("\nPerch and Gaze...\n")
			interruptFun(multiSecondInterruptTicker)
		}
	}
}
