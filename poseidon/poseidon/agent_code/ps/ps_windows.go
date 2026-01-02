//go:build windows

package ps

import "errors"

// WindowsProcess is a stub implementation of Process for Windows
type WindowsProcess struct {
	pid    int
	ppid   int
	binary string
}

func (p *WindowsProcess) Pid() int {
	return p.pid
}

func (p *WindowsProcess) PPid() int {
	return p.ppid
}

func (p *WindowsProcess) Arch() string {
	return ""
}

func (p *WindowsProcess) Executable() string {
	return p.binary
}

func (p *WindowsProcess) Owner() string {
	return ""
}

func (p *WindowsProcess) BinPath() string {
	return ""
}

func (p *WindowsProcess) ProcessArguments() []string {
	return []string{}
}

func (p *WindowsProcess) ProcessEnvironment() map[string]string {
	return map[string]string{}
}

func (p *WindowsProcess) SandboxPath() string {
	return ""
}

func (p *WindowsProcess) ScriptingProperties() map[string]interface{} {
	return map[string]interface{}{}
}

func (p *WindowsProcess) Name() string {
	return p.binary
}

func (p *WindowsProcess) BundleID() string {
	return ""
}

func (p *WindowsProcess) AdditionalInfo() map[string]interface{} {
	return map[string]interface{}{}
}

func Processes() ([]Process, error) {
	return []Process{}, errors.New("Not Implemented on Windows")
}
