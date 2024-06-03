//go:build windows
// +build windows

package tap

import "errors"

func Tap(tapMap map[string]string) error {
	// Unsupported
	return errors.New("Unsupported")
}

func TapWriter(pense string) (map[string]string, error) {
	// Unsupported
	return nil, errors.New("Unsupported")
}
