package lib

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var interruptChan chan os.Signal
var twoHundredMilliInterruptTicker *time.Ticker = time.NewTicker(200 * time.Millisecond)
var multiSecondInterruptTicker *time.Ticker = time.NewTicker(time.Second)
var fifteenSecondInterruptTicker *time.Ticker = time.NewTicker(time.Second * 15)
var thirtySecondInterruptTicker *time.Ticker = time.NewTicker(time.Second * 30)

var gEncryptPass string
var gEncryptSalt string
var gLocalHostAddr string
var gHostAddr string
var gHandshakeCode string
var gSessionIdentifier string
var gAcceptRemoteFunc func(int, string) (bool, error)
var gInterruptFunc func()

func init() {
	rand.Seed(time.Now().UnixNano())
}
func FeatherCtlInit(icIn chan os.Signal, localHostAddr string, encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, sessionIdentifier string, acceptRemoteFunc func(int, string) (bool, error), interruptFunc func()) {
	gLocalHostAddr = localHostAddr
	gEncryptPass = encryptPass
	gEncryptSalt = encryptSalt
	gHostAddr = hostAddr
	gHandshakeCode = handshakeCode
	gSessionIdentifier = sessionIdentifier
	gAcceptRemoteFunc = acceptRemoteFunc
	gInterruptFunc = interruptFunc
	interruptChan = icIn
}

func acceptInterruptFun(tickerContinue *time.Ticker, tickerBreak *time.Ticker, tickerInterrupt *time.Ticker) (bool, error) {
	select {
	case <-interruptChan:
		cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_PERCH, gSessionIdentifier, true, nil)
		return true, errors.New("you shall not pass")
	case <-tickerContinue.C:
		// don't break... continue...
		return false, nil
	case <-tickerBreak.C:
		// break and continue
		return true, nil
	case <-tickerInterrupt.C:
		// full stop
		return true, errors.New("timeout")
	}
	return true, errors.New("not possible")
}

func AcceptRemote(int, string) (bool, error) {
	return acceptInterruptFun(multiSecondInterruptTicker, fifteenSecondInterruptTicker, thirtySecondInterruptTicker)
}

func interruptFun(tickerInterrupt *time.Ticker) error {
	select {
	case <-interruptChan:
		cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_PERCH, gSessionIdentifier, true, nil)
		return errors.New("interrupted")
	case <-tickerInterrupt.C:
	}
	return nil
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
				err := interruptFun(twoHundredMilliInterruptTicker)
				if err != nil {
					if gInterruptFunc != nil {
						gInterruptFunc()
					} else {
						os.Exit(-1)
					}
				}
			} else {
				if err.Error() != "init" {
					if err.Error() == "you shall not pass" {
						if gInterruptFunc != nil {
							gInterruptFunc()
						} else {
							os.Exit(-1)
						}
					}
					fmt.Printf("\nWaiting...\n")
					err := interruptFun(multiSecondInterruptTicker)
					if err != nil {
						if gInterruptFunc != nil {
							gInterruptFunc()
						} else {
							os.Exit(-1)
						}
					}
					callFlap = cap.MODE_GAZE
				}
			}
			ctlFlapMode, err = cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, callFlap, gSessionIdentifier, bypass, gAcceptRemoteFunc)
		}
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func FeatherQueryCache(pense string) (string, error) {
	penseCode := randomString(7 + rand.Intn(7))
	penseArray := sha256.Sum256([]byte(penseCode))
	penseSum := hex.EncodeToString(penseArray[:])

	_, featherErr := cap.FeatherWriter(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, penseSum)
	if featherErr != nil {
		return "", featherErr
	}

	conn, err := grpc.Dial(gLocalHostAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", err
	}
	defer conn.Close()
	c := cap.NewCapClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.Pense(ctx, &cap.PenseRequest{Pense: penseCode, PenseIndex: pense})
	if err != nil {
		return "", err
	}

	return r.GetPense(), nil
}

func FeatherCtlEmitter(modeCtlTrailChan chan string,
	emote func(string),
	queryAction func(string) (string, error)) (string, error) {
	if emote == nil {
		emote = func(msg string) { fmt.Printf(msg) }
	}

	for {
	perching:
		if featherMode, featherErr := cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_FLAP, gSessionIdentifier, false, gAcceptRemoteFunc); featherErr == nil && strings.HasPrefix(featherMode, cap.MODE_GAZE) {
			emote("Fly away!\n")

			for modeCtl := range modeCtlTrailChan {
				if queryAction != nil {
					queryAction(modeCtl)
				}
				flapMode := cap.MODE_FLAP + "_" + modeCtl
				ctlFlapMode := flapMode
				var err error = errors.New("init")
				emote(fmt.Sprintf("%s.", modeCtl))

				for {
					if err == nil && ctlFlapMode == cap.MODE_PERCH {
						// Acknowledge perching...
						cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_PERCH, gSessionIdentifier, true, gAcceptRemoteFunc)
						ctlFlapMode = cap.MODE_PERCH
						goto perching
					}

					if err == nil && flapMode != ctlFlapMode {
						// Flap, Gaze, etc...
						err := interruptFun(twoHundredMilliInterruptTicker)
						if err != nil {
							if gInterruptFunc != nil {
								gInterruptFunc()
							} else {
								os.Exit(-1)
							}
						}

						break
					} else {
						callFlap := flapMode
						if err == nil {
							err := interruptFun(twoHundredMilliInterruptTicker)
							if err != nil {
								if gInterruptFunc != nil {
									gInterruptFunc()
								} else {
									os.Exit(-1)
								}
							}

						} else {
							if err.Error() != "init" {
								if err.Error() == "you shall not pass" {
									if gInterruptFunc != nil {
										gInterruptFunc()
									} else {
										os.Exit(-1)
									}
								}
								emote("\nWaiting...\n")
								err := interruptFun(multiSecondInterruptTicker)
								if err != nil {
									if gInterruptFunc != nil {
										gInterruptFunc()
									} else {
										os.Exit(-1)
									}
								}
							}
						}
						ctlFlapMode, err = cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, callFlap, gSessionIdentifier, true, gAcceptRemoteFunc)
					}
				}
			}
			emote("\nGliding....\n")
			cap.FeatherCtlEmit(gEncryptPass, gEncryptSalt, gHostAddr, gHandshakeCode, cap.MODE_GLIDE, gSessionIdentifier, true, gAcceptRemoteFunc)
		} else {
			emote("\nPerch and Gaze...\n")
			err := interruptFun(multiSecondInterruptTicker)
			if err != nil {
				if gInterruptFunc != nil {
					gInterruptFunc()
				} else {
					os.Exit(-1)
				}
			}
		}
	}
}
