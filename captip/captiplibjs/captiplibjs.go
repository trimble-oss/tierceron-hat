package captiplibjs

import (
	"os"
	"syscall/js"

	"github.com/trimble-oss/tierceron-hat/cap"
	captiplib "github.com/trimble-oss/tierceron-hat/captip/captiplib"
)

var gFeatherCtx *cap.FeatherContext

func emote(featherCtx *cap.FeatherContext, ctlFlapMode string, msg string) {
	js.Global().Call("console.log", msg)
}

func interrupted(featherCtx *cap.FeatherContext) error {
	// TODO: meaning in browser context
	return nil
}

func FeatherCtlInit(this js.Value, args []js.Value) any {
	encryptPass := args[0].String()
	encryptSalt := args[1].String()
	hostAddr := args[2].String()
	handshakeCode := args[3].String()
	sessionIdentifier := args[4].String()
	someEnv := args[5].String()

	var interruptChan chan os.Signal

	localHostAddr := ""

	gFeatherCtx = captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, &someEnv, captiplib.AcceptRemote, interrupted)

	return map[string]any{"message": ""}
}

func FeatherCtl(this js.Value, args []js.Value) any {
	go captiplib.FeatherCtl(gFeatherCtx, emote)
	return map[string]any{"message": "featherctl"}
}
