// Package commands provides a registry for command tests used in integration testing.
package commands

import (
	"sync"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

// CommandTest defines a test case for an agent command.
type CommandTest struct {
	// Name is the command name (e.g., "pwd", "ls", "shell").
	Name string

	// Parameters is the JSON-encoded parameters to send with the command.
	Parameters string

	// Validate is the validation function that checks the response.
	// It should return nil if the response is valid, or an error describing the failure.
	Validate func(mockafm.Response) error

	// Setup is an optional function to create test fixtures before the command runs.
	// The workdir parameter is the working directory for test fixtures.
	Setup func(workdir string) error

	// Teardown is an optional function to clean up test fixtures after the command completes.
	// The workdir parameter is the working directory used for test fixtures.
	Teardown func(workdir string) error
}

// registry holds all registered command tests.
var registry = struct {
	mu    sync.RWMutex
	tests map[string]CommandTest
}{
	tests: make(map[string]CommandTest),
}

// Register adds a command test to the registry.
// If a test with the same name already exists, it will be overwritten.
// This function is safe for concurrent use.
func Register(test CommandTest) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.tests[test.Name] = test
}

// GetAll returns all registered command tests.
// The returned slice is a copy and safe to modify.
// This function is safe for concurrent use.
func GetAll() []CommandTest {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	tests := make([]CommandTest, 0, len(registry.tests))
	for _, test := range registry.tests {
		tests = append(tests, test)
	}
	return tests
}

// Get returns a specific command test by name.
// Returns the test and true if found, or an empty CommandTest and false if not found.
// This function is safe for concurrent use.
func Get(name string) (CommandTest, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	test, ok := registry.tests[name]
	return test, ok
}

// Count returns the number of registered command tests.
// This function is safe for concurrent use.
func Count() int {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return len(registry.tests)
}

// Clear removes all registered command tests.
// This is primarily useful for testing the registry itself.
// This function is safe for concurrent use.
func Clear() {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.tests = make(map[string]CommandTest)
}

// Names returns the names of all registered command tests.
// The returned slice is a copy and safe to modify.
// This function is safe for concurrent use.
func Names() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.tests))
	for name := range registry.tests {
		names = append(names, name)
	}
	return names
}
