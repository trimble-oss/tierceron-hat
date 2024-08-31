package main

import (
	cap2 "github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
	"os"
	"sync"
	"testing"
)

func featherInterrupted(featherCtx *cap2.FeatherContext) error {
	cap2.FeatherCtlEmit(featherCtx, string(cap2.MODE_PERCH), *featherCtx.SessionIdentifier, true)
	os.Exit(-1)
	return nil
}

func TestGetSaltyGuardian(t *testing.T) {
	cap2.TapInitCodeSaltGuard(func() string { return "ExtraSaltPlease" })

	var serverStart sync.WaitGroup
	serverStart.Add(1)
	go func() {
		go cap2.Feather("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", func(int, string) bool { return true })

		cap2.TapFeather("I think", "therefore I am.")
		cap2.TapFeather("It is not enough to have a good mind.", "The main thing is to use it well.")

		cap2.TapFeather("Ponder", "a feather.")

		go cap2.TapServer("127.0.0.1:1534")
		serverStart.Done()
	}()

	serverStart.Wait()
	localHostAddr := "localhost:1534"
	encryptPass := "Som18vhjqa72935h"
	encryptSalt := "1cx7v89as7df89"
	hostAddr := "127.0.0.1:1832"
	handshakeCode := "ThisIsACode"
	sessionIdentifier := "FeatherSessionOne"
	env := "SomeEnv"

	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	featherCtx := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &env, captiplib.AcceptRemote, featherInterrupted)

	expected := "therefore I am."
	msg, err := captiplib.FeatherQueryCache(featherCtx, "I think")
	if err != nil {
		t.Fatalf("Expected '%s', got %s", expected, err.Error())
	}

	if msg != expected {
		t.Fatalf("Expected '%s', got %s", expected, msg)
	}
}
