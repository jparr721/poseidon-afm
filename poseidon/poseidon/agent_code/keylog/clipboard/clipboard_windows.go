//go:build windows

package clipboard

import "errors"

var Unsupported = true

func readAll() (string, error) {
	return "", errors.New("Not Implemented on Windows")
}

func writeAll(text string) error {
	return errors.New("Not Implemented on Windows")
}
