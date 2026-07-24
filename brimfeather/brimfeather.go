package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
	"github.com/trimble-oss/tierceron-hat/localcert"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func loadLocalFeatherTLSConfig(serverName string) (*cap.FeatherTLSConfig, error) {
	return localcert.LoadFeatherTLSConfig(
		[]string{"./servicecert.crt", "./local_config/servicecert.crt", "../servicecert.crt", "../local_config/servicecert.crt", "./serv_cert.pem", "../serv_cert.pem"},
		[]string{"./servicekey.key", "./local_config/servicekey.key", "../servicekey.key", "../local_config/servicekey.key", "./serv_key.pem", "../serv_key.pem"},
		[]string{"./serviceclientcert.pem", "./local_config/serviceclientcert.pem", "../serviceclientcert.pem", "../local_config/serviceclientcert.pem", "./servicecert.crt", "./local_config/servicecert.crt", "../servicecert.crt", "../local_config/servicecert.crt", "./serv_cert.pem", "../serv_cert.pem"},
		serverName,
	)
}

var modeCtlTrail []string = []string{"I", "wa", "a", "nde", "er", "thro", "ough", "the", "e", "lo", "o", "vly", "y", "wo", "ods", "I", "i", "wa", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "a", "an", "der", "through", "the", "woods."}

var penses []string = []string{"I think", "It is not enough to have a good mind.", "Ponder"}

// Global mpb container and bars for multi-line output
var (
	mpbContainer     *mpb.Progress
	barOne           *mpb.Bar
	barTwo           *mpb.Bar
	outputMutex      sync.Mutex
	messageOneBuffer string // accumulates messages for Session 1
	messageTwoBuffer string // accumulates messages for Session 2
)

func emote(featherCtx *cap.FeatherContext, ctlFlapMode []byte, msg string) {
	// Filter out control and status messages using case-insensitive contains checks
	msgLower := strings.ToLower(msg)
	if strings.Contains(msgLower, "waiting") || strings.Contains(msgLower, "perch and gaze") || strings.Contains(msgLower, "aborting connection") || strings.Contains(msgLower, "fly away") {
		return
	}

	outputMutex.Lock()
	defer outputMutex.Unlock()

	// Append message to the accumulating buffer
	if *featherCtx.SessionIdentifier == "FeatherSessionOne" {
		messageOneBuffer += msg + " "
		// Keep buffer size reasonable (truncate to last 500 chars if too long)
		if len(messageOneBuffer) > 500 {
			messageOneBuffer = messageOneBuffer[len(messageOneBuffer)-500:]
		}
		barOne.Increment()
	} else {
		messageTwoBuffer += msg + " "
		// Keep buffer size reasonable (truncate to last 500 chars if too long)
		if len(messageTwoBuffer) > 500 {
			messageTwoBuffer = messageTwoBuffer[len(messageTwoBuffer)-500:]
		}
		barTwo.Increment()
	}
}

func interrupted(featherCtx *cap.FeatherContext) error {
	os.Exit(-1)
	return nil
}

func queryAction(featherCtx *cap.FeatherContext, ctl string) (string, error) {
	if *featherCtx.SessionIdentifier == "FeatherSessionTwo" {
		// More leasurely walk through the woods.
		time.Sleep(time.Millisecond * 250)
	} else {
		if ctl == "thro" {
			return captiplib.FeatherQueryCache(featherCtx, "I think")
		}
	}
	return "", nil
}

func brimFeatherer(featherCtx *cap.FeatherContext) {
	var modeCtlTrailChan chan string = make(chan string, 1)

	go captiplib.FeatherCtlEmitter(featherCtx, modeCtlTrailChan, emote, queryAction)

rerun:
	// Reset buffers for new run
	outputMutex.Lock()
	messageOneBuffer = ""
	messageTwoBuffer = ""
	outputMutex.Unlock()

	atomic.StoreInt64(&featherCtx.RunState, cap.RUN_STARTED)
	for _, modeCtl := range modeCtlTrail {
		modeCtlTrailChan <- modeCtl
		if atomic.LoadInt64(&featherCtx.RunState) == cap.RESETTING {
			goto rerun
		}
	}
	// Wait briefly to ensure last item is transmitted before completing
	// TODO: need ack... but that defeats purpose of kcp...
	time.Sleep(3 * time.Second)
	modeCtlTrailChan <- cap.CTL_COMPLETE
	for {
		if atomic.LoadInt64(&featherCtx.RunState) == cap.RUNNING {
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	goto rerun
}

func main() {
	featherServerName := flag.String("fsn", "", "TLS server name covered by the local feather certificate")
	flag.Parse()

	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGALRM)

	localHostAddr := "127.0.0.1:1534"
	encryptPass := "Som18vhjqa72935h"
	encryptSalt := "1cx7v89as7df89"
	hostAddr := "127.0.0.1:1832"
	handshakeCode := "ThisIsACode"
	sessionIdentifier := "FeatherSessionOne"
	env := "SomeEnv"
	tlsConfig, err := loadLocalFeatherTLSConfig(*featherServerName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Initialize mpb container for graceful multi-line output
	mpbContainer = mpb.New()

	// Create custom bar fillers that display the accumulated message text
	barOneFillerFunc := func(w io.Writer, st decor.Statistics) error {
		outputMutex.Lock()
		msg := messageOneBuffer
		outputMutex.Unlock()

		// Write message up to the bar width, then pad with spaces
		barWidth := int(st.Total) - 5 // Account for brackets and prefix
		if len(msg) >= barWidth {
			_, err := fmt.Fprint(w, msg[:barWidth])
			return err
		}
		// Pad with spaces to fill the bar
		_, err := fmt.Fprintf(w, "%-*s", barWidth, msg)
		return err
	}

	barTwoFillerFunc := func(w io.Writer, st decor.Statistics) error {
		outputMutex.Lock()
		msg := messageTwoBuffer
		outputMutex.Unlock()

		// Write message up to the bar width, then pad with spaces
		barWidth := int(st.Total) - 5 // Account for brackets and prefix
		if len(msg) >= barWidth {
			_, err := fmt.Fprint(w, msg[:barWidth])
			return err
		}
		// Pad with spaces to fill the bar
		_, err := fmt.Fprintf(w, "%-*s", barWidth, msg)
		return err
	}

	// Create bars with custom fillers showing message text
	barOne = mpbContainer.AddBar(100,
		mpb.PrependDecorators(decor.Name("S1: ")),
		mpb.BarFillerMiddleware(func(base mpb.BarFiller) mpb.BarFiller {
			return mpb.BarFillerFunc(barOneFillerFunc)
		}))

	barTwo = mpbContainer.AddBar(100,
		mpb.PrependDecorators(decor.Name("S2: ")),
		mpb.BarFillerMiddleware(func(base mpb.BarFiller) mpb.BarFiller {
			return mpb.BarFillerFunc(barTwoFillerFunc)
		}))

	featherCtx := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &env, tlsConfig, captiplib.AcceptRemoteNoTimeout, interrupted)

	go brimFeatherer(featherCtx)

	sessionIdentifierTwo := "FeatherSessionTwo"

	featherCtxTwo := captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifierTwo, &env, tlsConfig, captiplib.AcceptRemoteNoTimeout, interrupted)

	go brimFeatherer(featherCtxTwo)

	<-interruptChan
	cap.FeatherStop()
}
