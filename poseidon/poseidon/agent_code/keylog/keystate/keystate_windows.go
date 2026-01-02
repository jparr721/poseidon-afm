//go:build windows

package keystate

import "errors"

func keyLogger() error {
	return errors.New("Not Implemented on Windows")
}
