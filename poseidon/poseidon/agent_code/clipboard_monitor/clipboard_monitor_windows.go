//go:build windows

package clipboard_monitor

import (
	"errors"
)

func CheckClipboard(oldCount int) (string, error) {
	return "", errors.New("Not Implemented on Windows")
}

func GetClipboardCount() (int, error) {
	return 0, errors.New("Not Implemented on Windows")
}

func GetFrontmostApp() (string, error) {
	return "", errors.New("Not Implemented on Windows")
}

func WaitForTime() {
}
