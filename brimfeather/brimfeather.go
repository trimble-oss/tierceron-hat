package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

var modeCtlTrail []string = []string{"I", "wa", "a", "nde", "er", "thro", "ough", "the", "e", "lo", "o", "vly", "y", "wo", "ods", "I", "i", "wa", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "an", "der", "through", "the", "woods."}

var penses []string = []string{"I think", "It is not enough to have a good mind.", "Ponder"}

func emote(msg string) {
	fmt.Print(msg)
}

func interrupted(featherCtx *cap.FeatherContext) error {
	os.Exit(-1)
	return nil
}

func main() {
	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGALRM)

	featherCtx := captiplib.FeatherCtlInit(interruptChan, "127.0.0.1:1534", "Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", "HelloWorld", captiplib.AcceptRemote, interrupted)

	var modeCtlTrailChan chan string = make(chan string)

	go captiplib.FeatherCtlEmitter(featherCtx, modeCtlTrailChan, emote, captiplib.FeatherQueryCache)

rerun:
	featherCtx.RunState = cap.RUN_STARTED
	for _, modeCtl := range modeCtlTrail {
		featherCtx.RunState = cap.RUNNING
		modeCtlTrailChan <- modeCtl
		if featherCtx.RunState == cap.RESETTING {
			goto rerun
		}
	}
}
