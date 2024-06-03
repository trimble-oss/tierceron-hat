package main

import (
	"os"
	"strings"

	"github.com/mrjrieke/hat/cap"
	"github.com/mrjrieke/hat/cap/tap"
)

// The original crown
func main() {
	exePath, exePathErr := os.Readlink("/proc/self/exe")
	if exePathErr != nil {
		os.Exit(-1)
	}
	brimPath := strings.Replace(exePath, "/crown", "/brim", 1)
	tapMap := map[string]string{brimPath: "2c1d03a2869e2040bbd125661f49d4bca2b9b0751ec92d0119a744edc31932ff"}
	go tap.Tap(tapMap)

	tap.TapEyeRemember("eye", "rememeber")
	cap.TapMemorize("I think", "therefore I am.")
	cap.TapMemorize("It is not enough to have a good mind.", "The main thing is to use it well.")

	cap.TapServer("127.0.0.1:1534")
}
