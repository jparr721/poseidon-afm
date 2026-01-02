//go:build windows

package xpc

import (
	"errors"
)

func runCommand(command string) ([]byte, error) {
	return nil, errors.New("Not Implemented on Windows")
}
