package main

import (
	"log"

	"github.com/trimble-oss/tierceron-hat/cap"
)

func featherCtl(pense string) {
	_, featherErr := cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_FEATHER, pense)
	if featherErr != nil {
		log.Fatalf("Failed to feather ctl emit: %v", featherErr)
	}
}

func main() {
	featherCtl("HelloWorld")
	featherCtl("HelloWorld")
}
