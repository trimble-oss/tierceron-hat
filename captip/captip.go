package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
)

var interruptChan chan os.Signal = make(chan os.Signal)
var twoHundredMilliInterruptTicker *time.Ticker = time.NewTicker(200 * time.Millisecond)
var multiSecondInterruptTicker *time.Ticker = time.NewTicker(time.Second)

func acceptRemote(int, string) bool {
	interruptFun(multiSecondInterruptTicker)
	return true
}

func interruptFun(tickerInterrupt *time.Ticker) {
	select {
	case <-interruptChan:
		cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", cap.MODE_PERCH, "HelloWorld", true, nil)
		os.Exit(1)
	case <-tickerInterrupt.C:
	}
}

func featherCtl(pense string) {
	flapMode := cap.MODE_GAZE
	ctlFlapMode := flapMode
	var err error = errors.New("init")
	bypass := err == nil || err.Error() != "init"

	for {
		gazeCnt := 0
		if err == nil && ctlFlapMode == cap.MODE_GLIDE {
			break
		} else {
			callFlap := flapMode
			if err == nil {
				if strings.HasPrefix(ctlFlapMode, cap.MODE_FLAP) {
					ctl := strings.Split(ctlFlapMode, "_")
					if len(ctl) > 1 {
						fmt.Printf("%s.", ctl[1])
					}
					callFlap = cap.MODE_GAZE
					gazeCnt = 0
				} else {
					gazeCnt = gazeCnt + 1
					if gazeCnt > 5 {
						// Too much gazing
						bypass = false
					}
					callFlap = cap.MODE_GAZE
				}
				interruptFun(twoHundredMilliInterruptTicker)
			} else {
				if err.Error() != "init" {
					fmt.Printf("\nWaiting...\n")
					interruptFun(multiSecondInterruptTicker)
					callFlap = cap.MODE_GAZE
				}
			}
			ctlFlapMode, err = cap.FeatherCtlEmit("Som18vhjqa72935h", "1cx7v89as7df89", "127.0.0.1:1832", "ThisIsACode", callFlap, "HelloWorld", bypass, acceptRemote)
		}
	}
}

func main() {
	var ic chan os.Signal = make(chan os.Signal)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		x := <-ic
		interruptChan <- x
	}()

	fmt.Printf("\nFirst run\n")
	featherCtl("HelloWorld")
	fmt.Printf("\nResting....\n")
	time.Sleep(20 * time.Second)
	fmt.Printf("\nTime for work....\n")
	fmt.Printf("\n2nd run\n")
	featherCtl("HelloWorld")
}
