//go:build windows

package keys

import (
	"errors"
)

type WindowsKeyInformation struct {
	KeyType string
	KeyData []byte
}

func (l *WindowsKeyInformation) Type() string {
	return l.KeyType
}

func (l *WindowsKeyInformation) Data() []byte {
	return l.KeyData
}

func getkeydata(opts Arguments) (WindowsKeyInformation, error) {
	d := WindowsKeyInformation{}
	d.KeyType = ""
	d.KeyData = []byte{}
	return d, errors.New("Not Implemented on Windows")
}
