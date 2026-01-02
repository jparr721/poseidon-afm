//go:build windows

package listtasks

import (
	"errors"
)

type ListtasksWindows struct {
	Results map[string]interface{}
}

func (l *ListtasksWindows) Result() map[string]interface{} {
	return l.Results
}

func getAvailableTasks() (ListtasksWindows, error) {
	n := ListtasksWindows{}
	m := map[string]interface{}{
		"result": "Not Implemented on Windows",
	}
	n.Results = m
	return n, errors.New("Not Implemented on Windows")
}
