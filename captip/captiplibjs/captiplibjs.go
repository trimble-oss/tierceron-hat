package captiplibjs

import (
	"os"
	"syscall/js"

	"github.com/mrjrieke/hat/cap"
	captiplib "github.com/mrjrieke/hat/captip/captiplib"
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

	var interruptChan chan os.Signal

	localHostAddr := ""

	gFeatherCtx = captiplib.FeatherCtlInit(interruptChan, &localHostAddr, &encryptPass, &encryptSalt, &hostAddr, &handshakeCode, &sessionIdentifier, captiplib.AcceptRemote, interrupted)

	return map[string]any{"message": ""}
}

func FeatherCtl(this js.Value, args []js.Value) any {
	go captiplib.FeatherCtl(gFeatherCtx, args[0].String(), emote)
	return map[string]any{"message": "featherctl"}
}
