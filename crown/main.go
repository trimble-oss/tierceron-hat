package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mrjrieke/hat/cap"
)

func main() {
	ex, err := os.Executable()
	if err != nil {
		os.Exit(-1)
	}
	exePath := filepath.Dir(ex)
	brimPath := strings.Replace(exePath, "/crown", "/brim", 1)
	go cap.Tap(brimPath, "76b2e62226ea89a690808afa60f9062f43e2c6c21b5c436e7c6e6d136aa0715d")
	cap.TapServer("127.0.0.1:1534")

}
