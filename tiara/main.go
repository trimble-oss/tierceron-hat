package main

import (
	"fmt"

	"github.com/mrjrieke/hat/cap"
)

func main() {
	fmt.Println("Starting tiara")
	go cap.Feather("Som18vhjqa72935h", "1cx7v89as7df89", "1832", "ThisIsACode", func(string) bool { return true })

	cap.TapFeather("I think", "therefore I am.")
	cap.TapFeather("It is not enough to have a good mind.", "The main thing is to use it well.")

	cap.TapMemorize("Ponder", "Ponder me this.")
	cap.TapFeather("Ponder", "Ponder a feather.")

	cap.TapServer("127.0.0.1:1534")
}
