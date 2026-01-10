// Package commands provides command test definitions for integration testing.
// This file defines the test for the "ls" command.
package commands

import (
	"errors"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func init() {
	Register(CommandTest{
		Name:       "ls",
		Parameters: `{"path": ".", "depth": 1}`,
		Validate: func(resp mockafm.Response) error {
			// ls returns file_browser data
			if resp.FileBrowser == nil {
				return errors.New("ls should return file_browser data")
			}
			return nil
		},
	})
}
