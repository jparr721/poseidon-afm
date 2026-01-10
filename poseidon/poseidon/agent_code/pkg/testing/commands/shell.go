// Package commands provides command test definitions for integration testing.
// This file defines the test for the "shell" command.
package commands

import (
	"errors"
	"strings"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func init() {
	Register(CommandTest{
		Name:       "shell",
		Parameters: `{"command": "echo hello"}`,
		Validate: func(resp mockafm.Response) error {
			// shell might use user_output or stdout
			output := resp.UserOutput
			if output == "" {
				output = resp.Stdout
			}
			if !strings.Contains(output, "hello") {
				return errors.New("shell output should contain 'hello'")
			}
			return nil
		},
	})
}
