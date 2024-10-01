package main

import (
	"os"
	"strings"

	"github.com/trimble-oss/tierceron-hat/cap"
	"github.com/trimble-oss/tierceron-hat/cap/tap"
)

const penseDir = "/tmp/trccarrier/"

// The original crown
func main() {
	exePath, exePathErr := os.Readlink("/proc/self/exe")
	if exePathErr != nil {
		os.Exit(-1)
	}
	brimPath := strings.Replace(exePath, "/crown", "/brim", 1)
	tapMap := map[string]string{brimPath: "2c1d03a2869e2040bbd125661f49d4bca2b9b0751ec92d0119a744edc31932ff"}
	go tap.Tap(penseDir, tapMap, "", false)

	keyvar := new(string)
	*keyvar = "rememeber"
	tap.TapEyeRemember("eye", keyvar)
	keyvar2 := new(string)
	*keyvar2 = "therefore I am."
	cap.TapMemorize("I think", keyvar2)
	keyvar3 := new(string)
	*keyvar3 = "The main thing is to use it well."
	cap.TapMemorize("It is not enough to have a good mind.", keyvar3)

	cap.TapServer("127.0.0.1:1534")
}
