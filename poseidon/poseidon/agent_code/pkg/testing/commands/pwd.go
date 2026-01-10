// Package commands provides command test definitions for integration testing.
// This file defines the test for the "pwd" command.
package commands

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func init() {
	Register(CommandTest{
		Name:       "pwd",
		Parameters: "{}",
		Validate: func(resp mockafm.Response) error {
			output := strings.TrimSpace(resp.UserOutput)
			if output == "" {
				return errors.New("pwd returned empty output")
			}
			if !filepath.IsAbs(output) {
				return errors.New("pwd should return absolute path")
			}
			return nil
		},
	})
}
