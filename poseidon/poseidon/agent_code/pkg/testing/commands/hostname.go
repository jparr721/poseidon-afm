// Package commands provides command test definitions for integration testing.
// This file defines the test for the "hostname" command.
package commands

import (
	"errors"
	"strings"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func init() {
	Register(CommandTest{
		Name:       "hostname",
		Parameters: "{}",
		Validate: func(resp mockafm.Response) error {
			output := strings.TrimSpace(resp.UserOutput)
			if output == "" {
				return errors.New("hostname returned empty output")
			}
			return nil
		},
	})
}
