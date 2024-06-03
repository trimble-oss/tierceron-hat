package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mrjrieke/hat/cap"
	"github.com/mrjrieke/hat/cap/tap"
)

// Not really part of example set... extracted from cap to simplify that library.
func main() {
	ex, err := os.Executable()
	if err != nil {
		os.Exit(-1)
	}
	exePath := filepath.Dir(ex)
	brimPath := strings.Replace(exePath, "/Cap", "/brim", 1)
	tapMap := map[string]string{brimPath: "2c1d03a2869e2040bbd125661f49d4bca2b9b0751ec92d0119a744edc31932ff"}
	go tap.Tap(tapMap)
	cap.TapServer("127.0.0.1:1534")
}
