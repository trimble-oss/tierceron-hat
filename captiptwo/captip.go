package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
	"github.com/trimble-oss/tierceron-hat/localcert"
)

func loadLocalFeatherTLSConfig(serverName string) (*cap.FeatherTLSConfig, error) {
	return localcert.LoadFeatherTLSConfig(
		[]string{"./servicecert.crt", "./local_config/servicecert.crt", "../servicecert.crt", "../local_config/servicecert.crt", "./serv_cert.pem", "../serv_cert.pem"},
		[]string{"./servicekey.key", "./local_config/servicekey.key", "../servicekey.key", "../local_config/servicekey.key", "./serv_key.pem", "../serv_key.pem"},
		[]string{"./serviceclientcert.pem", "./local_config/serviceclientcert.pem", "../serviceclientcert.pem", "../local_config/serviceclientcert.pem", "./servicecert.crt", "./local_config/servicecert.crt", "../servicecert.crt", "../local_config/servicecert.crt", "./serv_cert.pem", "../serv_cert.pem"},
		serverName,
	)
}

func emote(featherCtx *cap.FeatherContext, ctlFlapMode string, msg string) {
	msgLower := strings.ToLower(msg)
	if strings.Contains(msgLower, "waiting") || strings.Contains(msgLower, "perch and gaze") || strings.Contains(msgLower, "aborting connection") || strings.Contains(msgLower, "fly away") {
		return
	}
	fmt.Print(msg)
}

func interrupted(featherCtx *cap.FeatherContext) error {
	os.Exit(130)
	return nil
}

func main() {
	featherServerName := flag.String("fsn", "", "TLS server name covered by the local feather certificate")
	flag.Parse()

	var interruptChan chan os.Signal = make(chan os.Signal, 5)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGALRM)
	var controlInterruptChan chan os.Signal = make(chan os.Signal, 1)

	localHostAddr := ""
	encryptPass := "Som18vhjqa72935h"
	encryptSalt := "1cx7v89as7df89"
	hostAddr := "127.0.0.1:1832"
	handshakeCode := "ThisIsACode"
	sessionIdentifier := "FeatherSessionTwo"
	env := "SomeEnv"
	tlsConfig, err := loadLocalFeatherTLSConfig(*featherServerName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	featherCtx := captiplib.FeatherCtlInit(controlInterruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &env, tlsConfig, captiplib.AcceptRemote, interrupted)

	done := make(chan struct{})
	go func() {
		fmt.Printf("\nFirst run\n")
		captiplib.FeatherCtl(featherCtx, emote)
		fmt.Printf("\nResting....\n")
		time.Sleep(2 * time.Second)

		// Reset server state before 2nd run
		cap.FeatherCtlEmit(featherCtx, string(cap.MODE_PERCH), *featherCtx.SessionIdentifier, true)
		fmt.Printf("\nTime for work....\n")
		fmt.Printf("\n2nd run\n")
		captiplib.FeatherCtl(featherCtx, emote)
		fmt.Printf("\nResting....\n")
		time.Sleep(1 * time.Second)

		// Reset server state before 3rd run
		cap.FeatherCtlEmit(featherCtx, string(cap.MODE_PERCH), *featherCtx.SessionIdentifier, true)
		fmt.Printf("\nTime for work....\n")
		fmt.Printf("\n3rd run\n")
		captiplib.FeatherCtl(featherCtx, emote)
		fmt.Printf("\nResting....\n")
		time.Sleep(2 * time.Second)

		// Reset server state before 4th run
		cap.FeatherCtlEmit(featherCtx, string(cap.MODE_PERCH), *featherCtx.SessionIdentifier, true)
		fmt.Printf("\nTime for work....\n")
		fmt.Printf("\n4th run\n")
		captiplib.FeatherCtl(featherCtx, emote)
		fmt.Printf("\nResting....\n")
		time.Sleep(2 * time.Second)

		close(done)
	}()

	<-interruptChan
	interrupted(featherCtx)
	<-done
}
