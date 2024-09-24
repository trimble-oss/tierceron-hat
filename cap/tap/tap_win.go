//go:build windows
// +build windows

package tap

import "errors"

func TapInit(pd string) {
	// Unsupported
	return
}

func Tap(tapMap map[string]string, group string, skipPathControls bool) error {
	// Unsupported
	return errors.New("Unsupported")
}

func TapWriter(pense string) (map[string]string, error) {
	// Unsupported
	return nil, errors.New("Unsupported")
}
