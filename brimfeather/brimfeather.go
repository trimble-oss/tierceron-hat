package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

var modeCtlTrail []string = []string{"I", "wa", "a", "nde", "er", "thro", "ough", "the", "e", "lo", "o", "vly", "y", "wo", "ods", "I", "i", "wa", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "an", "der", "through", "the", "woods."}

var penses []string = []string{"I think", "It is not enough to have a good mind.", "Ponder"}

func emote(featherCtx *cap.FeatherContext, ctlFlapMode []byte, msg string) {
	fmt.Printf("%s.", msg)
}

func interrupted(featherCtx *cap.FeatherContext) error {
	os.Exit(-1)
	return nil
}

func queryAction(featherCtx *cap.FeatherContext, ctl string) (string, error) {
	if *featherCtx.SessionIdentifier == "FeatherSessionTwo" {
		// More leasurely walk through the woods.
		time.Sleep(time.Millisecond * 250)
	} else {
		if ctl == "thro" {
			return captiplib.FeatherQueryCache(featherCtx, "I think")
		}
	}
	return "", nil
}

func brimFeatherer(featherCtx *cap.FeatherContext) {

	var modeCtlTrailChan chan string = make(chan string, 1)

	go captiplib.FeatherCtlEmitter(featherCtx, modeCtlTrailChan, emote, queryAction)

rerun:
	atomic.StoreInt64(&featherCtx.RunState, cap.RUN_STARTED)
	for _, modeCtl := range modeCtlTrail {
		modeCtlTrailChan <- modeCtl
		if atomic.LoadInt64(&featherCtx.RunState) == cap.RESETTING {
			goto rerun
		}
	}
	modeCtlTrailChan <- cap.CTL_COMPLETE
	for {
		if atomic.LoadInt64(&featherCtx.RunState) == cap.RUNNING {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	goto rerun
}

func main() {
	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGALRM)

	localHostAddr := ""
	encryptPass := "Som18vhjqa72935h"
	encryptSalt := "1cx7v89as7df89"
	hostAddr := "127.0.0.1:1832"
	handshakeCode := "ThisIsACode"
	sessionIdentifier := "FeatherSessionOne"
	env := "SomeEnv"

	featherCtx := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &env, captiplib.AcceptRemote, interrupted)

	go brimFeatherer(featherCtx)

	sessionIdentifierTwo := "FeatherSessionTwo"

	featherCtxTwo := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifierTwo, &env, captiplib.AcceptRemote, interrupted)

	go brimFeatherer(featherCtxTwo)

	serverChan := make(chan struct{})
	<-serverChan
}
