//go:build windows

package libinject

import "errors"

type WindowsInjection struct {
	Target      int
	Successful  bool
	Payload     []byte
	LibraryPath string
}

func (l *WindowsInjection) TargetPid() int {
	return l.Target
}

func (l *WindowsInjection) Success() bool {
	return l.Successful
}

func (l *WindowsInjection) Shellcode() []byte {
	return l.Payload
}

func (l *WindowsInjection) SharedLib() string {
	return l.LibraryPath
}

func injectLibrary(pid int, path string) (WindowsInjection, error) {
	res := WindowsInjection{}
	res.Successful = false
	return res, errors.New("Not Implemented on Windows")
}
