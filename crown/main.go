package main

import (
	"os"
	"strings"

	"github.com/mrjrieke/hat/cap"
)

func main() {
	exePath, exePathErr := os.Readlink("/proc/self/exe")
	if exePathErr != nil {
		os.Exit(-1)
	}
	brimPath := strings.Replace(exePath, "/crown", "/brim", 1)
	go cap.Tap(brimPath, "f634bed34ba6bb6a198187705e38cd58d64972c14586608b93acaa6f84cd4e38")
	cap.TapServer("127.0.0.1:1534")

}
