package lib

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}
func FeatherCtlInit(icIn chan os.Signal,
	localHostAddr *string,
	encryptPass *string,
	encryptSalt *string,
	hostAddr *string,
	handshakeCode *string,
	sessionIdentifier *string,
	acceptRemoteFunc func(*cap.FeatherContext, int, string) (bool, error),
	interruptedFunc func(*cap.FeatherContext) error) *cap.FeatherContext {
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
		cap.FeatherCtlEmit(featherCtx, string(cap.MODE_PERCH), *featherCtx.SessionIdentifier, true)
		return true, errors.New(YOU_SHALL_NOT_PASS)
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
		cap.FeatherCtlEmit(featherCtx, string(cap.MODE_PERCH), *featherCtx.SessionIdentifier, true)
		return errors.New("interrupted")
	case <-tickerInterrupt.C:
	}
	return nil
}

func FeatherCtl(featherCtx *cap.FeatherContext,
	emote func(*cap.FeatherContext, string, string),
) {
	flapMode := string(cap.MODE_GAZE)
	ctlFlapMode := flapMode
	var err error = errors.New("init")
	bypass := err == nil || err.Error() != "init"
	if emote == nil {
		emote = func(featherCtx *cap.FeatherContext, flapMode string, msg string) { fmt.Printf("%s.", msg) }
	}

	for {
		gazeCnt := 0
		if err == nil && len(ctlFlapMode) > 0 && ctlFlapMode[0] == cap.MODE_GLIDE {
			emote(featherCtx, ctlFlapMode, "\nGliding...\n")
			break
		} else {
			callFlap := flapMode
			if err == nil {
				if len(ctlFlapMode) > 0 && ctlFlapMode[0] == cap.MODE_PERCH {
					ctl := strings.Split(ctlFlapMode, "_")
					if len(ctl) > 1 {
						if ctl[1] == cap.CTL_COMPLETE {
							break
						}
					}

				} else if len(ctlFlapMode) > 0 && ctlFlapMode[0] == cap.MODE_FLAP {
					ctl := strings.Split(ctlFlapMode, "_")
					if len(ctl) > 1 {
						if ctl[1] == cap.CTL_COMPLETE {
							break
						}
						emote(featherCtx, ctlFlapMode, fmt.Sprintf("%s.", ctl[1]))
					}
					callFlap = string(cap.MODE_GAZE)
					gazeCnt = 0
				} else {
					gazeCnt = gazeCnt + 1
					if gazeCnt > 5 {
						// Too much gazing
						bypass = false
					}
					callFlap = string(cap.MODE_GAZE)
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
					if err.Error() == YOU_SHALL_NOT_PASS {
						if featherCtx.InterruptHandlerFunc != nil {
							featherCtx.InterruptHandlerFunc(featherCtx)
						} else {
							os.Exit(-1)
						}
					}
					emote(featherCtx, ctlFlapMode, "\nWaiting...\n")
					err := interruptFun(featherCtx, featherCtx.MultiSecondInterruptTicker)
					if err != nil {
						if featherCtx.InterruptHandlerFunc != nil {
							featherCtx.InterruptHandlerFunc(featherCtx)
						} else {
							os.Exit(-1)
						}
					}
					callFlap = string(cap.MODE_GAZE)
				}
			}
			ctlFlapMode, err = cap.FeatherCtlEmit(featherCtx, callFlap, *featherCtx.SessionIdentifier, bypass)
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

	conn, err := grpc.Dial(*featherCtx.LocalHostAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

const (
	YOU_SHALL_NOT_PASS = "you shall not pass"
	MSG_FLY_AWAY       = "Fly away!\n"
	MSG_WAITING        = "\nWaiting...\n"
	MSG_GLIDING        = "\nGliding....\n"
	MSG_PERCH_AND_GAZE = "\nPerch and Gaze...\n"
)

func FeatherCtlEmitter(featherCtx *cap.FeatherContext, modeCtlTrailChan chan string,
	emote func(*cap.FeatherContext, []byte, string),
	queryAction func(*cap.FeatherContext, string) (string, error)) (string, error) {
	if emote == nil {
		emote = func(featherCtx *cap.FeatherContext, ctlFlapMode []byte, msg string) {
			fmt.Print(msg)
		}
	}
	sessionIdBinary := []byte(*featherCtx.SessionIdentifier)

	for {
	perching:
		if ctlFlapMode, featherErr := cap.FeatherCtlEmitBinary(featherCtx, string(cap.MODE_FLAP), sessionIdBinary, false); featherErr == nil && len(ctlFlapMode) > 0 && ctlFlapMode[0] == cap.MODE_GAZE {
			emote(featherCtx, cap.MODE_FLAP_BYTES, MSG_FLY_AWAY)
			// If it's still running, reset it...
			atomic.CompareAndSwapInt64(&featherCtx.RunState, cap.RUNNING, cap.RESETTING)

			for modeCtl := range modeCtlTrailChan {
				atomic.StoreInt64(&featherCtx.RunState, cap.RUNNING)
				if modeCtl == cap.CTL_COMPLETE {
					flapMode := []byte{cap.MODE_PERCH, '_'}
					flapMode = append(flapMode, []byte(cap.CTL_COMPLETE)...)

					cap.FeatherCtlEmitBinary(featherCtx, string(flapMode), sessionIdBinary, true)
					goto perching
				}
				if queryAction != nil {
					queryAction(featherCtx, modeCtl)
				}
				flapMode := []byte{cap.MODE_FLAP, '_'}
				flapMode = append(flapMode, []byte(modeCtl)...)

				ctlFlapMode := flapMode
				var err error = errors.New("init")
				emote(featherCtx, ctlFlapMode, modeCtl)

				for {
					if err == nil && len(ctlFlapMode) > 0 && ctlFlapMode[0] == cap.MODE_PERCH {
						// Acknowledge perching...
						cap.FeatherCtlEmitBinary(featherCtx, string(cap.MODE_PERCH), sessionIdBinary, true)
						atomic.StoreInt64(&featherCtx.RunState, cap.RESETTING)
						goto perching
					}

					if err == nil && flapMode[0] != ctlFlapMode[0] {
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
								if err.Error() == YOU_SHALL_NOT_PASS {
									if featherCtx.InterruptHandlerFunc != nil {
										featherCtx.InterruptHandlerFunc(featherCtx)
									} else {
										os.Exit(-1)
									}
								}
								emote(featherCtx, ctlFlapMode, MSG_WAITING)
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
						ctlFlapMode, err = cap.FeatherCtlEmitBinary(featherCtx, string(callFlap), sessionIdBinary, true)
					}
				}
			}
			emote(featherCtx, cap.MODE_GLIDE_BYTES, MSG_GLIDING)
			cap.FeatherCtlEmitBinary(featherCtx, string(cap.MODE_GLIDE), sessionIdBinary, true)
		} else {
			if featherErr != nil && featherErr.Error() == YOU_SHALL_NOT_PASS {
				if featherCtx.InterruptHandlerFunc != nil {
					featherCtx.InterruptHandlerFunc(featherCtx)
				} else {
					os.Exit(-1)
				}
			}
			emote(featherCtx, ctlFlapMode, MSG_PERCH_AND_GAZE)
			if featherErr == nil {
				if bytes.HasSuffix(ctlFlapMode, cap.CTL_COMPLETE_BYTES) {
					// Picked up our own complete message.
					flapMode := []byte{cap.MODE_PERCH, '_'}
					flapMode = append(flapMode, []byte(cap.CTL_COMPLETE)...)

					cap.FeatherCtlEmitBinary(featherCtx, string(flapMode), sessionIdBinary, true)

				} else {
					if atomic.LoadInt64(&featherCtx.RunState) == cap.RUNNING {
						for {
							// drain before reset.
							select {
							case <-modeCtlTrailChan:
							default:
								atomic.StoreInt64(&featherCtx.RunState, cap.RESETTING)
								goto cleancomplete
							}
						}
					cleancomplete:
					}
				}

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
