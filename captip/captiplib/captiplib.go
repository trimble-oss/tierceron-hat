package lib

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mrjrieke/hat/cap"
)

var interruptChan chan os.Signal = make(chan os.Signal)
var twoHundredMilliInterruptTicker *time.Ticker = time.NewTicker(200 * time.Millisecond)
var multiSecondInterruptTicker *time.Ticker = time.NewTicker(time.Second)
var fifteenSecondInterruptTicker *time.Ticker = time.NewTicker(time.Second * 15)
var thirtySecondInterruptTicker *time.Ticker = time.NewTicker(time.Second * 30)

var gEncryptPass string
var gEncryptSalt string
var gHostAddr string
var gHandshakeCode string
var gAcceptRemote func(int, string) (bool, error)

func Init(encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, acceptRemote func(int, string) (bool, error)) {
	gEncryptPass = encryptPass
	gEncryptSalt = encryptSalt
	gHostAddr = hostAddr
	gHandshakeCode = handshakeCode
	gAcceptRemote = acceptRemote

	var ic chan os.Signal = make(chan os.Signal)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		x := <-ic
		interruptChan <- x
	}()
}

func acceptInterruptFun(tickerContinue *time.Ticker, tickerBreak *time.Ticker, tickerInterrupt *time.Ticker) (bool, error) {
	select {
	case <-interruptChan:
		cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_PERCH, "HelloWorld", true, nil)
		return true, errors.New("you shall not pass")
	case <-tickerContinue.C:
		// don't break... continue...
		return false, nil
	case <-tickerBreak.C:
		// break and continue
		return true, nil
	case <-tickerInterrupt.C:
		// full stop
		return true, errors.New("you shall not pass")
	}
	return true, errors.New("not possible")
}

func AcceptRemote(int, string) (bool, error) {
	return acceptInterruptFun(multiSecondInterruptTicker, fifteenSecondInterruptTicker, thirtySecondInterruptTicker)
}

func interruptFun(tickerInterrupt *time.Ticker) {
	select {
	case <-interruptChan:
		cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_PERCH, "HelloWorld", true, nil)
		os.Exit(1)
	case <-tickerInterrupt.C:
	}
}

func FeatherCtl(pense string) {
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
			ctlFlapMode, err = cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, callFlap, "HelloWorld", bypass, gAcceptRemote)
		}
	}
}
