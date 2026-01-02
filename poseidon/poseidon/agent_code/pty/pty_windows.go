//go:build windows

package pty

import (
	"errors"
	"os"
	"os/exec"
)

func customPtyStart(command *exec.Cmd) (*os.File, error) {
	return nil, errors.New("Not Implemented on Windows")
}
