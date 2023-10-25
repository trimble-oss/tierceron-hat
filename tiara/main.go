package main

import (
	"fmt"

	"github.com/mrjrieke/hat/cap"
)

// The next crowning
func main() {
	fmt.Println("Starting tiara")
	go cap.Feather("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", func(int, string) bool { return true })

	cap.TapFeather("I think", "therefore I am.")
	cap.TapFeather("It is not enough to have a good mind.", "The main thing is to use it well.")

	cap.TapMemorize("Ponder", "me this.")
	cap.TapFeather("Ponder", "a feather.")

	cap.TapServer("127.0.0.1:1534")
}
