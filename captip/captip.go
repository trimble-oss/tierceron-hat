package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

func emote(featherCtx *cap.FeatherContext, ctlFlapMode string, msg string) {
	msgLower := strings.ToLower(msg)
	if strings.Contains(msgLower, "waiting") || strings.Contains(msgLower, "perch and gaze") || strings.Contains(msgLower, "aborting connection") || strings.Contains(msgLower, "fly away") {
		return
	}
	fmt.Print(msg)
}

func interrupted(featherCtx *cap.FeatherContext) error {
	cap.FeatherCtlEmit(featherCtx, string(cap.MODE_PERCH), *featherCtx.SessionIdentifier, true)
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
	sessionIdentifier := "FeatherSessionOne"
	env := "SomeEnv"
	tlsConfig := cap.NewFeatherSelfSignedTLSConfig()

	featherCtx := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &env, tlsConfig, captiplib.AcceptRemote, interrupted)

	done := make(chan struct{})
	go func() {
		fmt.Printf("\nFirst run\n")
		captiplib.FeatherCtl(featherCtx, emote)
		fmt.Printf("\nResting....\n")
		time.Sleep(20 * time.Second)
		fmt.Printf("\nTime for work....\n")
		fmt.Printf("\n2nd run\n")
		captiplib.FeatherCtl(featherCtx, emote)
		close(done)
	}()

	<-interruptChan
	interrupted(featherCtx)
	<-done
}
