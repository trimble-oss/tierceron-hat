package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

var modeCtlTrail []string = []string{"I", "wa", "a", "nde", "er", "thro", "ough", "the", "e", "lo", "o", "vly", "y", "wo", "ods", "I", "i", "wa", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "an", "der", "through", "the", "woods."}

var penses []string = []string{"I think", "It is not enough to have a good mind.", "Ponder"}

func emote(featherCtx *cap.FeatherContext, ctlFlapMode []byte, msg string) {
	fmt.Print(msg)
}

func interrupted(featherCtx *cap.FeatherContext) error {
	os.Exit(-1)
	return nil
}

func main() {
	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGALRM)

	localHostAddr := ""
	encryptPass := "Som18vhjqa72935h"
	encryptSalt := "1cx7v89as7df89"
	hostAddr := "127.0.0.1:1832"
	handshakeCode := "ThisIsACode"
	sessionIdentifier := "HelloWorld"

	featherCtx := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, captiplib.AcceptRemote, interrupted)

	var modeCtlTrailChan chan string = make(chan string)

	go captiplib.FeatherCtlEmitter(featherCtx, modeCtlTrailChan, emote, captiplib.FeatherQueryCache)

rerun:
	atomic.StoreInt64(&featherCtx.RunState, cap.RUN_STARTED)
	for _, modeCtl := range modeCtlTrail {
		atomic.StoreInt64(&featherCtx.RunState, cap.RUNNING)
		modeCtlTrailChan <- modeCtl
		if atomic.LoadInt64(&featherCtx.RunState) == cap.RESETTING {
			goto rerun
		}
	}
}
