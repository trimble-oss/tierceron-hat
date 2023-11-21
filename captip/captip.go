package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

func interupted() {
	os.Exit(-1)
}
func main() {
	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	captiplib.FeatherCtlInit(interruptChan, "", "Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", "HelloWorld", captiplib.AcceptRemote, interupted)

	fmt.Printf("\nFirst run\n")
	captiplib.FeatherCtl("HelloWorld")
	fmt.Printf("\nResting....\n")
	time.Sleep(20 * time.Second)
	fmt.Printf("\nTime for work....\n")
	fmt.Printf("\n2nd run\n")
	captiplib.FeatherCtl("HelloWorld")
}
