//go:build windows

package execute_library

type WindowsExecuteMemory struct {
	Message string
}

func executeLibrary(_, _ string, _ []string) (WindowsExecuteMemory, error) {
	res := WindowsExecuteMemory{}
	res.Message = "Not Supported"
	return res, nil
}
