package commands

import (
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func TestRegisterAndGet(t *testing.T) {
	// Clear registry before test
	Clear()

	test := CommandTest{
		Name:       "test-cmd",
		Parameters: `{"param": "value"}`,
		Validate: func(r mockafm.Response) error {
			return nil
		},
	}

	Register(test)

	// Verify we can get it back
	got, ok := Get("test-cmd")
	if !ok {
		t.Fatal("Get returned false for registered test")
	}

	if got.Name != test.Name {
		t.Errorf("Name: got %q, want %q", got.Name, test.Name)
	}
	if got.Parameters != test.Parameters {
		t.Errorf("Parameters: got %q, want %q", got.Parameters, test.Parameters)
	}
	if got.Validate == nil {
		t.Error("Validate function is nil")
	}
}

func TestGetNotFound(t *testing.T) {
	Clear()

	_, ok := Get("nonexistent")
	if ok {
		t.Error("Get returned true for unregistered test")
	}
}

func TestRegisterOverwrite(t *testing.T) {
	Clear()

	test1 := CommandTest{
		Name:       "overwrite-test",
		Parameters: "params1",
	}
	test2 := CommandTest{
		Name:       "overwrite-test",
		Parameters: "params2",
	}

	Register(test1)
	Register(test2)

	got, ok := Get("overwrite-test")
	if !ok {
		t.Fatal("Get returned false")
	}

	if got.Parameters != "params2" {
		t.Errorf("Parameters: got %q, want %q", got.Parameters, "params2")
	}
}

func TestGetAll(t *testing.T) {
	Clear()

	tests := []CommandTest{
		{Name: "cmd1", Parameters: "p1"},
		{Name: "cmd2", Parameters: "p2"},
		{Name: "cmd3", Parameters: "p3"},
	}

	for _, test := range tests {
		Register(test)
	}

	all := GetAll()
	if len(all) != 3 {
		t.Errorf("GetAll returned %d tests, want 3", len(all))
	}

	// Sort by name for deterministic comparison
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	for i, got := range all {
		if got.Name != tests[i].Name {
			t.Errorf("all[%d].Name: got %q, want %q", i, got.Name, tests[i].Name)
		}
	}
}

func TestGetAllEmpty(t *testing.T) {
	Clear()

	all := GetAll()
	if all == nil {
		t.Error("GetAll returned nil, want empty slice")
	}
	if len(all) != 0 {
		t.Errorf("GetAll returned %d tests, want 0", len(all))
	}
}

func TestCount(t *testing.T) {
	Clear()

	if count := Count(); count != 0 {
		t.Errorf("Count on empty registry: got %d, want 0", count)
	}

	Register(CommandTest{Name: "test1"})
	if count := Count(); count != 1 {
		t.Errorf("Count after 1 register: got %d, want 1", count)
	}

	Register(CommandTest{Name: "test2"})
	if count := Count(); count != 2 {
		t.Errorf("Count after 2 registers: got %d, want 2", count)
	}
}

func TestClear(t *testing.T) {
	Clear()

	Register(CommandTest{Name: "test1"})
	Register(CommandTest{Name: "test2"})

	if count := Count(); count != 2 {
		t.Fatalf("setup failed: expected 2 tests, got %d", count)
	}

	Clear()

	if count := Count(); count != 0 {
		t.Errorf("Count after Clear: got %d, want 0", count)
	}
}

func TestNames(t *testing.T) {
	Clear()

	Register(CommandTest{Name: "alpha"})
	Register(CommandTest{Name: "beta"})
	Register(CommandTest{Name: "gamma"})

	names := Names()
	if len(names) != 3 {
		t.Errorf("Names returned %d names, want 3", len(names))
	}

	// Sort for deterministic comparison
	sort.Strings(names)
	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d]: got %q, want %q", i, name, expected[i])
		}
	}
}

func TestNamesEmpty(t *testing.T) {
	Clear()

	names := Names()
	if names == nil {
		t.Error("Names returned nil, want empty slice")
	}
	if len(names) != 0 {
		t.Errorf("Names returned %d names, want 0", len(names))
	}
}

func TestValidateFunction(t *testing.T) {
	Clear()

	expectedErr := errors.New("validation failed")
	test := CommandTest{
		Name: "validate-test",
		Validate: func(r mockafm.Response) error {
			if r.UserOutput != "expected" {
				return expectedErr
			}
			return nil
		},
	}

	Register(test)

	got, _ := Get("validate-test")

	// Test failure case
	err := got.Validate(mockafm.Response{UserOutput: "wrong"})
	if err != expectedErr {
		t.Errorf("Validate with wrong output: got %v, want %v", err, expectedErr)
	}

	// Test success case
	err = got.Validate(mockafm.Response{UserOutput: "expected"})
	if err != nil {
		t.Errorf("Validate with correct output: got %v, want nil", err)
	}
}

func TestSetupAndTeardown(t *testing.T) {
	Clear()

	setupCalled := false
	teardownCalled := false

	test := CommandTest{
		Name: "setup-teardown-test",
		Setup: func(workdir string) error {
			setupCalled = true
			if workdir != "/test/dir" {
				return errors.New("wrong workdir")
			}
			return nil
		},
		Teardown: func(workdir string) error {
			teardownCalled = true
			if workdir != "/test/dir" {
				return errors.New("wrong workdir")
			}
			return nil
		},
	}

	Register(test)

	got, _ := Get("setup-teardown-test")

	// Call Setup
	if err := got.Setup("/test/dir"); err != nil {
		t.Errorf("Setup failed: %v", err)
	}
	if !setupCalled {
		t.Error("Setup was not called")
	}

	// Call Teardown
	if err := got.Teardown("/test/dir"); err != nil {
		t.Errorf("Teardown failed: %v", err)
	}
	if !teardownCalled {
		t.Error("Teardown was not called")
	}
}

func TestNilSetupAndTeardown(t *testing.T) {
	Clear()

	// Test with nil Setup and Teardown (should be valid)
	test := CommandTest{
		Name:       "nil-hooks",
		Parameters: "",
		Setup:      nil,
		Teardown:   nil,
	}

	Register(test)

	got, ok := Get("nil-hooks")
	if !ok {
		t.Fatal("Get returned false")
	}

	if got.Setup != nil {
		t.Error("Setup should be nil")
	}
	if got.Teardown != nil {
		t.Error("Teardown should be nil")
	}
}

func TestConcurrentRegister(t *testing.T) {
	Clear()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Register tests concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			test := CommandTest{
				Name:       string(rune('A' + id%26)),
				Parameters: "",
			}
			Register(test)
		}(i)
	}

	wg.Wait()

	// Should have at most 26 unique tests (A-Z)
	count := Count()
	if count > 26 {
		t.Errorf("Count after concurrent register: got %d, want <= 26", count)
	}
	if count == 0 {
		t.Error("Count is 0, expected some tests to be registered")
	}
}

func TestConcurrentRead(t *testing.T) {
	Clear()

	// Register some tests
	for i := 0; i < 10; i++ {
		Register(CommandTest{
			Name:       string(rune('A' + i)),
			Parameters: "",
		})
	}

	var wg sync.WaitGroup
	numReaders := 100

	// Read concurrently
	for i := 0; i < numReaders; i++ {
		wg.Add(3)

		// GetAll reader
		go func() {
			defer wg.Done()
			_ = GetAll()
		}()

		// Get reader
		go func() {
			defer wg.Done()
			_, _ = Get("A")
		}()

		// Count reader
		go func() {
			defer wg.Done()
			_ = Count()
		}()
	}

	wg.Wait()
}

func TestConcurrentReadWrite(t *testing.T) {
	Clear()

	var wg sync.WaitGroup
	numOperations := 50

	// Mix of reads and writes
	for i := 0; i < numOperations; i++ {
		wg.Add(4)

		// Writer
		go func(id int) {
			defer wg.Done()
			Register(CommandTest{
				Name:       string(rune('A' + id%26)),
				Parameters: "",
			})
		}(i)

		// GetAll reader
		go func() {
			defer wg.Done()
			_ = GetAll()
		}()

		// Get reader
		go func() {
			defer wg.Done()
			_, _ = Get("A")
		}()

		// Names reader
		go func() {
			defer wg.Done()
			_ = Names()
		}()
	}

	wg.Wait()
}

// TestInitPatternUsage demonstrates how command files will use the registry
func TestInitPatternUsage(t *testing.T) {
	Clear()

	// Simulate what a command file's init() would do
	init1 := func() {
		Register(CommandTest{
			Name:       "pwd",
			Parameters: "",
			Validate: func(r mockafm.Response) error {
				if r.UserOutput == "" {
					return errors.New("pwd returned empty output")
				}
				return nil
			},
		})
	}

	init2 := func() {
		Register(CommandTest{
			Name:       "hostname",
			Parameters: "",
			Validate: func(r mockafm.Response) error {
				if r.UserOutput == "" {
					return errors.New("hostname returned empty output")
				}
				return nil
			},
		})
	}

	// Call init functions (simulating Go's init() mechanism)
	init1()
	init2()

	// Verify both are registered
	if count := Count(); count != 2 {
		t.Errorf("Count: got %d, want 2", count)
	}

	pwdTest, ok := Get("pwd")
	if !ok {
		t.Fatal("pwd test not found")
	}

	hostnameTest, ok := Get("hostname")
	if !ok {
		t.Fatal("hostname test not found")
	}

	// Test validate functions work
	if err := pwdTest.Validate(mockafm.Response{UserOutput: "/home/user"}); err != nil {
		t.Errorf("pwd validate failed: %v", err)
	}
	if err := hostnameTest.Validate(mockafm.Response{UserOutput: "myhost"}); err != nil {
		t.Errorf("hostname validate failed: %v", err)
	}
}
