package captiplibjs

import (
	"syscall/js"

	captiplib "github.com/mrjrieke/hat/captip/captiplib"
)

func FeatherCtlInit(this js.Value, args []js.Value) any {
	encryptPass := args[0].String()
	encryptSalt := args[1].String()
	hostAddr := args[2].String()
	handshakeCode := args[3].String()
	captiplib.Init(encryptPass, encryptSalt, hostAddr, handshakeCode, captiplib.AcceptRemote)

	return map[string]any{"message": ""}
}

func FeatherCtl(this js.Value, args []js.Value) any {
	go captiplib.FeatherCtl(args[0].String())
	return map[string]any{"message": "featherctl"}
}
