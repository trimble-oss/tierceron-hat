package main

import (
	"fmt"

	"github.com/trimble-oss/tierceron-hat/cap"
)

// The next crowning
func main() {
	fmt.Println("Starting tiara")
	go cap.Feather("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", func(int, string) bool { return true })

	keyvar := new(string)
	*keyvar = "therefore I am."
	cap.TapFeather("I think", keyvar)

	keyvar2 := new(string)
	*keyvar2 = "The main thing is to use it well."
	cap.TapFeather("It is not enough to have a good mind.", keyvar2)

	keyvar3 := new(string)
	*keyvar3 = "me this."
	cap.TapMemorize("Ponder", keyvar3)

	keyvar4 := new(string)
	*keyvar4 = "a feather."
	cap.TapFeather("Ponder", keyvar4)

	cap.TapServer("127.0.0.1:1534")
}
