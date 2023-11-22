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

func init() {
	rand.Seed(time.Now().UnixNano())
}
func FeatherCtlInit(icIn chan os.Signal, localHostAddr string, encryptPass string, encryptSalt string, hostAddr string, handshakeCode string, sessionIdentifier string, acceptRemoteFunc func(*cap.FeatherContext, int, string) (bool, error), interruptedFunc func(*cap.FeatherContext) error) *cap.FeatherContext {
	return &cap.FeatherContext{
		LocalHostAddr:                  localHostAddr,
		EncryptPass:                    encryptPass,
		EncryptSalt:                    encryptSalt,
		HostAddr:                       hostAddr,
		HandshakeCode:                  handshakeCode,
		SessionIdentifier:              sessionIdentifier,
		AcceptRemoteFunc:               acceptRemoteFunc,
		InterruptHandlerFunc:           interruptedFunc,
		InterruptChan:                  icIn,
		TwoHundredMilliInterruptTicker: time.NewTicker(200 * time.Millisecond),
		MultiSecondInterruptTicker:     time.NewTicker(time.Second),
		FifteenSecondInterruptTicker:   time.NewTicker(time.Second * 15),
		ThirtySecondInterruptTicker:    time.NewTicker(time.Second * 30),
	}
}

func acceptInterruptFun(featherCtx *cap.FeatherContext, tickerContinue *time.Ticker, tickerBreak *time.Ticker, tickerInterrupt *time.Ticker) (bool, error) {
	select {
	case <-featherCtx.InterruptChan:
		cap.FeatherCtlEmit(featherCtx, cap.MODE_PERCH, featherCtx.SessionIdentifier, true)
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

func AcceptRemote(featherCtx *cap.FeatherContext, x int, y string) (bool, error) {
	return acceptInterruptFun(featherCtx, featherCtx.MultiSecondInterruptTicker, featherCtx.FifteenSecondInterruptTicker, featherCtx.ThirtySecondInterruptTicker)
}

func interruptFun(featherCtx *cap.FeatherContext, tickerInterrupt *time.Ticker) error {
	select {
	case <-featherCtx.InterruptChan:
		cap.FeatherCtlEmit(featherCtx, cap.MODE_PERCH, featherCtx.SessionIdentifier, true)
		return errors.New("interrupted")
	case <-tickerInterrupt.C:
	}
	return nil
}

func FeatherCtl(featherCtx *cap.FeatherContext, pense string) {
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
				err := interruptFun(featherCtx, featherCtx.TwoHundredMilliInterruptTicker)
				if err != nil {
					if featherCtx.InterruptHandlerFunc != nil {
						featherCtx.InterruptHandlerFunc(featherCtx)
					} else {
						os.Exit(-1)
					}
				}
			} else {
				if err.Error() != "init" {
					if err.Error() == "you shall not pass" {
						if featherCtx.InterruptHandlerFunc != nil {
							featherCtx.InterruptHandlerFunc(featherCtx)
						} else {
							os.Exit(-1)
						}
					}
					fmt.Printf("\nWaiting...\n")
					err := interruptFun(featherCtx, featherCtx.MultiSecondInterruptTicker)
					if err != nil {
						if featherCtx.InterruptHandlerFunc != nil {
							featherCtx.InterruptHandlerFunc(featherCtx)
						} else {
							os.Exit(-1)
						}
					}
					callFlap = cap.MODE_GAZE
				}
			}
			ctlFlapMode, err = cap.FeatherCtlEmit(featherCtx, callFlap, featherCtx.SessionIdentifier, bypass)
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

func FeatherQueryCache(featherCtx *cap.FeatherContext, pense string) (string, error) {
	penseCode := randomString(7 + rand.Intn(7))
	penseArray := sha256.Sum256([]byte(penseCode))
	penseSum := hex.EncodeToString(penseArray[:])

	_, featherErr := cap.FeatherWriter(featherCtx, penseSum)
	if featherErr != nil {
		return "", featherErr
	}

	conn, err := grpc.Dial(featherCtx.LocalHostAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func FeatherCtlEmitter(featherCtx *cap.FeatherContext, modeCtlTrailChan chan string,
	emote func(string),
	queryAction func(*cap.FeatherContext, string) (string, error)) (string, error) {
	if emote == nil {
		emote = func(msg string) { fmt.Print(msg) }
	}

	for {
	perching:
		if featherMode, featherErr := cap.FeatherCtlEmit(featherCtx, cap.MODE_FLAP, featherCtx.SessionIdentifier, false); featherErr == nil && strings.HasPrefix(featherMode, cap.MODE_GAZE) {
			emote("Fly away!\n")

			for modeCtl := range modeCtlTrailChan {
				if queryAction != nil {
					queryAction(featherCtx, modeCtl)
				}
				flapMode := cap.MODE_FLAP + "_" + modeCtl
				ctlFlapMode := flapMode
				var err error = errors.New("init")
				emote(fmt.Sprintf("%s.", modeCtl))

				for {
					if err == nil && ctlFlapMode == cap.MODE_PERCH {
						// Acknowledge perching...
						cap.FeatherCtlEmit(featherCtx, cap.MODE_PERCH, featherCtx.SessionIdentifier, true)
						ctlFlapMode = cap.MODE_PERCH
						goto perching
					}

					if err == nil && flapMode != ctlFlapMode {
						// Flap, Gaze, etc...
						err := interruptFun(featherCtx, featherCtx.TwoHundredMilliInterruptTicker)
						if err != nil {
							if featherCtx.InterruptHandlerFunc != nil {
								featherCtx.InterruptHandlerFunc(featherCtx)
							} else {
								os.Exit(-1)
							}
						}

						break
					} else {
						callFlap := flapMode
						if err == nil {
							err := interruptFun(featherCtx, featherCtx.TwoHundredMilliInterruptTicker)
							if err != nil {
								if featherCtx.InterruptHandlerFunc != nil {
									featherCtx.InterruptHandlerFunc(featherCtx)
								} else {
									os.Exit(-1)
								}
							}

						} else {
							if err.Error() != "init" {
								if err.Error() == "you shall not pass" {
									if featherCtx.InterruptHandlerFunc != nil {
										featherCtx.InterruptHandlerFunc(featherCtx)
									} else {
										os.Exit(-1)
									}
								}
								emote("\nWaiting...\n")
								err := interruptFun(featherCtx, featherCtx.MultiSecondInterruptTicker)
								if err != nil {
									if featherCtx.InterruptHandlerFunc != nil {
										featherCtx.InterruptHandlerFunc(featherCtx)
									} else {
										os.Exit(-1)
									}
								}
							}
						}
						ctlFlapMode, err = cap.FeatherCtlEmit(featherCtx, callFlap, featherCtx.SessionIdentifier, true)
					}
				}
			}
			emote("\nGliding....\n")
			cap.FeatherCtlEmit(featherCtx, cap.MODE_GLIDE, featherCtx.SessionIdentifier, true)
		} else {
			emote("\nPerch and Gaze...\n")
			if featherCtx.RunState == cap.RUNNING {
				for {
					// drain before reset.
					select {
					case <-modeCtlTrailChan:
					default:
						featherCtx.RunState = cap.RESETTING
						goto cleancomplete
					}
				}
			cleancomplete:
			}
			err := interruptFun(featherCtx, featherCtx.MultiSecondInterruptTicker)
			if err != nil {
				if featherCtx.InterruptHandlerFunc != nil {
					featherCtx.InterruptHandlerFunc(featherCtx)
				} else {
					os.Exit(-1)
				}
			}
		}
	}
}
