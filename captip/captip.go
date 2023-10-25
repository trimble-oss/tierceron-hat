package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
)

func featherCtl(pense string) {
	flapMode := cap.MODE_FLAP
	ctlFlapMode := flapMode
	var err error = errors.New("init")

	for {
		if err == nil && ctlFlapMode == cap.MODE_PERCH {
			break
		} else {
			callFlap := flapMode
			if err == nil {
				if strings.HasPrefix(ctlFlapMode, cap.MODE_FLAP) {
					ctl := strings.Split(ctlFlapMode, "_")
					fmt.Print(ctl)
					callFlap = cap.MODE_GLIDE
				} else {
					callFlap = cap.MODE_GAZE
				}
				time.Sleep(200 * time.Millisecond)
			} else {
				if err.Error() != "init" {
					fmt.Println("Waiting...")
					time.Sleep(1 * time.Second)
					callFlap = cap.MODE_GAZE
				}
			}
			ctlFlapMode, err = cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", callFlap, "HelloWorld")
		}
	}
}

func main() {
	featherCtl("HelloWorld")
	featherCtl("HelloWorld")
}
