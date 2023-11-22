package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrjrieke/hat/cap"
	captiplib "github.com/mrjrieke/hat/captip/captiplib"
)

func interrupted(featherCtx *cap.FeatherContext) error {
	cap.FeatherCtlEmit(featherCtx, cap.MODE_PERCH, featherCtx.SessionIdentifier, true)
	os.Exit(-1)
	return nil
}

func main() {
	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	featherCtx := captiplib.FeatherCtlInit(interruptChan, "", "Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", "HelloWorld", captiplib.AcceptRemote, interrupted)

	fmt.Printf("\nFirst run\n")
	captiplib.FeatherCtl(featherCtx, "HelloWorld")
	fmt.Printf("\nResting....\n")
	time.Sleep(20 * time.Second)
	fmt.Printf("\nTime for work....\n")
	fmt.Printf("\n2nd run\n")
	captiplib.FeatherCtl(featherCtx, "HelloWorld")
}
