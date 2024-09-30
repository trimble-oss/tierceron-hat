//go:build windows
// +build windows

package tap

import "errors"

func TapInit(pd string) {
	// Unsupported
	return
}

func Tap(penseDir string, tapMap map[string]string, group string, skipPathControls bool) error {
	// Unsupported
	return errors.New("Unsupported")
}

func TapWriter(penseDir string, pense string) (map[string]string, error) {
	// Unsupported
	return nil, errors.New("Unsupported")
}
