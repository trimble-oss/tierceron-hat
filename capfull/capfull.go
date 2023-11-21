package main

import (
	"syscall/js"

	"github.com/mrjrieke/hat/captip/captiplibjs"
)

func main() {
	js.Global().Set("FeatherCtlInit", js.FuncOf(captiplibjs.FeatherCtlInit))
	js.Global().Set("FeatherCtl", js.FuncOf(captiplibjs.FeatherCtl))

	select {}
}
