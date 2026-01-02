//go:build windows

package screencapture

import "errors"

type WindowsScreenshot struct {
	MonitorIndex   int
	ScreenshotData []byte
}

func (d *WindowsScreenshot) Monitor() int {
	return d.MonitorIndex
}

func (d *WindowsScreenshot) Data() []byte {
	return d.ScreenshotData
}

func getscreenshot() ([]ScreenShot, error) {
	return nil, errors.New("Not Implemented on Windows")
}
