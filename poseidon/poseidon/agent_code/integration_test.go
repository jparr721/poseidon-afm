//go:build integration

package main

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	itesting "github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/commands"
)

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create harness with test configuration
	h := itesting.NewHarness(itesting.HarnessConfig{
		PSK:         generateTestPSK(),
		OperationID: "integration-test",
		AgentUUID:   generateTestUUID(),
		BuildTags:   []string{"http"},
		Debug:       testing.Verbose(),
	})
	defer h.Cleanup()

	// Setup (starts mock server, builds agent)
	if err := h.Setup(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Log server URL for debugging
	t.Logf("Mock server URL: %s", h.GetServerURL())
	t.Logf("Agent binary: %s", h.GetBinaryPath())

	// Spawn agent
	if err := h.SpawnAgent(); err != nil {
		t.Fatalf("SpawnAgent failed: %v", err)
	}

	// Wait for check-in
	if err := h.WaitForCheckin(30 * time.Second); err != nil {
		t.Fatalf("WaitForCheckin failed: %v", err)
	}

	t.Log("Agent checked in successfully")

	// Run all registered command tests
	cmdTests := commands.GetAll()
	if len(cmdTests) == 0 {
		t.Skip("No command tests registered")
	}

	for _, cmd := range cmdTests {
		t.Run(cmd.Name, func(t *testing.T) {
			t.Logf("Running command: %s with parameters: %s", cmd.Name, cmd.Parameters)

			resp, err := h.RunCommand(cmd, 30*time.Second)
			if err != nil {
				t.Fatalf("RunCommand failed: %v", err)
			}

			t.Logf("Response: completed=%v, status=%s, output=%q",
				resp.Completed, resp.Status, truncateOutput(resp.UserOutput, 200))

			if err := cmd.Validate(resp); err != nil {
				t.Errorf("Validation failed: %v", err)
			}
		})
	}
}

func TestIntegrationSingleCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This test demonstrates running a single specific command
	// Useful for debugging individual command issues
	cmdName := "pwd" // Change this to test specific commands

	cmd, ok := commands.Get(cmdName)
	if !ok {
		t.Skipf("Command %q not registered", cmdName)
	}

	h := itesting.NewHarness(itesting.HarnessConfig{
		PSK:         generateTestPSK(),
		OperationID: "single-cmd-test",
		AgentUUID:   generateTestUUID(),
		BuildTags:   []string{"http"},
		Debug:       testing.Verbose(),
	})
	defer h.Cleanup()

	if err := h.Setup(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	if err := h.SpawnAgent(); err != nil {
		t.Fatalf("SpawnAgent failed: %v", err)
	}

	if err := h.WaitForCheckin(30 * time.Second); err != nil {
		t.Fatalf("WaitForCheckin failed: %v", err)
	}

	resp, err := h.RunCommand(cmd, 30*time.Second)
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	t.Logf("Response: %+v", resp)

	if err := cmd.Validate(resp); err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}

// generateTestPSK generates a random base64-encoded 32-byte key.
func generateTestPSK() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		// Fallback to a deterministic key for testing
		for i := range key {
			key[i] = byte(i)
		}
	}
	return base64.StdEncoding.EncodeToString(key)
}

// generateTestUUID generates a new random UUID string.
func generateTestUUID() string {
	return uuid.New().String()
}

// truncateOutput truncates output for logging, adding ellipsis if truncated.
func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
