//go:build windows

package clipboard

import "errors"

func init() {
	Unsupported = true
}

func readAll() (string, error) {
	return "", errors.New("Not Implemented on Windows")
}

func writeAll(text string) error {
	return errors.New("Not Implemented on Windows")
}
