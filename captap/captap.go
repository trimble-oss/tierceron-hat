package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/trimble-oss/tierceron-hat/cap"
	"github.com/trimble-oss/tierceron-hat/cap/tap"
)

// Not really part of example set... extracted from cap to simplify that library.
func main() {
	ex, err := os.Executable()
	if err != nil {
		os.Exit(-1)
	}
	exePath := filepath.Dir(ex)
	brimPath := strings.Replace(exePath, "/Cap", "/brim", 1)
	go tap.Tap(brimPath, "f19431f322ea015ef871d267cc75e58b73d16617f9ff47ed7e0f0c1dbfb276b5")
	cap.TapServer("127.0.0.1:1534")
}
