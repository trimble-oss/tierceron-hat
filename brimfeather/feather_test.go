package main

import (
	"net"
	"os"
	"sync"
	"testing"
	"time"

	cap2 "github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

func featherInterrupted(featherCtx *cap2.FeatherContext) error {
	cap2.FeatherCtlEmit(featherCtx, string(cap2.MODE_PERCH), *featherCtx.SessionIdentifier, true)
	os.Exit(-1)
	return nil
}

func TestGetSaltyGuardian(t *testing.T) {
	cap2.TapInitCodeSaltGuard(func() string { return "ExtraSaltPlease" })
	tlsConfig := cap2.NewFeatherSelfSignedTLSConfig()

	var serverStart sync.WaitGroup
	serverStart.Add(1)
	go func() {
		go cap2.FeatherWithTLS("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", tlsConfig, func(int, string) bool { return true })

		keyvar := new(string)
		*keyvar = "therefore I am."
		cap2.TapFeather("I think", keyvar)

		keyvar2 := new(string)
		*keyvar2 = "The main thing is to use it well."
		cap2.TapFeather("It is not enough to have a good mind.", keyvar2)

		keyvar3 := new(string)
		*keyvar3 = "a feather."
		cap2.TapFeather("Ponder", keyvar3)

		go cap2.TapServer("127.0.0.1:1534")
		serverStart.Done()
	}()

	serverStart.Wait()
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:1534", 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("tap server did not start listening: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	localHostAddr := "localhost:1534"
	encryptPass := "Som18vhjqa72935h"
	encryptSalt := "1cx7v89as7df89"
	hostAddr := "127.0.0.1:1832"
	handshakeCode := "ThisIsACode"
	sessionIdentifier := "FeatherSessionOne"
	env := "SomeEnv"

	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	featherCtx := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &env, tlsConfig, captiplib.AcceptRemote, featherInterrupted)

	expected := "therefore I am."
	msg, err := captiplib.FeatherQueryCache(featherCtx, "I think")
	if err != nil {
		t.Fatalf("Expected '%s', got %s", expected, err.Error())
	}

	if msg != expected {
		t.Fatalf("Expected '%s', got %s", expected, msg)
	}
}
