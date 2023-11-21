package main

import (
	"fmt"
	"time"

	captiplib "github.com/mrjrieke/hat/captip/captiplib"
)

func main() {
	captiplib.Init("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", captiplib.AcceptRemote)

	fmt.Printf("\nFirst run\n")
	captiplib.FeatherCtl("HelloWorld")
	fmt.Printf("\nResting....\n")
	time.Sleep(20 * time.Second)
	fmt.Printf("\nTime for work....\n")
	fmt.Printf("\n2nd run\n")
	captiplib.FeatherCtl("HelloWorld")
}
