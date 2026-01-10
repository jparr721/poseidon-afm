package testing

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/commands"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

// TestNewHarness tests harness creation with various configurations.
func TestNewHarness(t *testing.T) {
	tests := []struct {
		name           string
		config         HarnessConfig
		wantBuildTags  []string
		wantBuildTime  time.Duration
		wantStartTime  time.Duration
	}{
		{
			name: "default values",
			config: HarnessConfig{
				PSK:         "dGVzdGtleS10aGlydHktdHdvLWJ5dGVzLWxvbmc=",
				OperationID: "test-op",
				AgentUUID:   "test-uuid-1234-5678-9abc-def012345678",
			},
			wantBuildTags: []string{"http"},
			wantBuildTime: 2 * time.Minute,
			wantStartTime: 10 * time.Second,
		},
		{
			name: "custom values",
			config: HarnessConfig{
				PSK:               "dGVzdGtleS10aGlydHktdHdvLWJ5dGVzLWxvbmc=",
				OperationID:       "custom-op",
				AgentUUID:         "custom-uuid-1234-5678-9abc-def012345678",
				BuildTags:         []string{"websocket"},
				Debug:             true,
				BuildTimeout:      5 * time.Minute,
				AgentStartTimeout: 30 * time.Second,
			},
			wantBuildTags: []string{"websocket"},
			wantBuildTime: 5 * time.Minute,
			wantStartTime: 30 * time.Second,
		},
		{
			name: "multiple build tags",
			config: HarnessConfig{
				PSK:         "dGVzdGtleS10aGlydHktdHdvLWJ5dGVzLWxvbmc=",
				OperationID: "multi-op",
				AgentUUID:   "multi-uuid-1234-5678-9abc-def012345678",
				BuildTags:   []string{"http", "tcp"},
			},
			wantBuildTags: []string{"http", "tcp"},
			wantBuildTime: 2 * time.Minute,
			wantStartTime: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHarness(tt.config)

			if len(h.config.BuildTags) != len(tt.wantBuildTags) {
				t.Errorf("BuildTags length = %d, want %d", len(h.config.BuildTags), len(tt.wantBuildTags))
			}
			for i, tag := range h.config.BuildTags {
				if tag != tt.wantBuildTags[i] {
					t.Errorf("BuildTags[%d] = %s, want %s", i, tag, tt.wantBuildTags[i])
				}
			}

			if h.config.BuildTimeout != tt.wantBuildTime {
				t.Errorf("BuildTimeout = %v, want %v", h.config.BuildTimeout, tt.wantBuildTime)
			}

			if h.config.AgentStartTimeout != tt.wantStartTime {
				t.Errorf("AgentStartTimeout = %v, want %v", h.config.AgentStartTimeout, tt.wantStartTime)
			}

			// Check initial state
			if h.isSetup {
				t.Error("isSetup should be false initially")
			}
			if h.isSpawned {
				t.Error("isSpawned should be false initially")
			}
			if h.isCheckedIn {
				t.Error("isCheckedIn should be false initially")
			}
		})
	}
}

// TestHarnessStateChecks tests state validation in harness methods.
func TestHarnessStateChecks(t *testing.T) {
	h := NewHarness(HarnessConfig{
		PSK:         "dGVzdGtleS10aGlydHktdHdvLWJ5dGVzLWxvbmc=",
		OperationID: "test-op",
		AgentUUID:   "test-uuid-1234-5678-9abc-def012345678",
	})

	// SpawnAgent should fail without Setup
	err := h.SpawnAgent()
	if err != ErrNotSetup {
		t.Errorf("SpawnAgent without Setup should return ErrNotSetup, got %v", err)
	}

	// WaitForCheckin should fail without Setup
	err = h.WaitForCheckin(1 * time.Second)
	if err != ErrNotSetup {
		t.Errorf("WaitForCheckin without Setup should return ErrNotSetup, got %v", err)
	}

	// RunCommand should fail without Setup
	_, err = h.RunCommand(commands.CommandTest{Name: "test"}, 1*time.Second)
	if err != ErrNotSetup {
		t.Errorf("RunCommand without Setup should return ErrNotSetup, got %v", err)
	}

	// GetServerURL should return empty without Setup
	url := h.GetServerURL()
	if url != "" {
		t.Errorf("GetServerURL without Setup should return empty, got %s", url)
	}
}

// TestGenerateConfigJSON tests the configuration JSON generation.
func TestGenerateConfigJSON(t *testing.T) {
	// Generate a valid 32-byte key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}
	psk := base64.StdEncoding.EncodeToString(testKey)

	tests := []struct {
		name       string
		config     HarnessConfig
		wantProfiles []string
		checkHTTP  bool
		checkWS    bool
		checkTCP   bool
	}{
		{
			name: "http profile",
			config: HarnessConfig{
				PSK:         psk,
				OperationID: "op-123",
				AgentUUID:   "uuid-1234-5678-9abc-def012345678",
				BuildTags:   []string{"http"},
				Debug:       true,
			},
			wantProfiles: []string{"http"},
			checkHTTP:    true,
		},
		{
			name: "websocket profile",
			config: HarnessConfig{
				PSK:         psk,
				OperationID: "op-456",
				AgentUUID:   "uuid-abcd-efgh-ijkl-mnopqrstuvwx",
				BuildTags:   []string{"websocket"},
				Debug:       false,
			},
			wantProfiles: []string{"websocket"},
			checkWS:      true,
		},
		{
			name: "tcp profile",
			config: HarnessConfig{
				PSK:         psk,
				OperationID: "op-789",
				AgentUUID:   "uuid-tcp-test-uuid-1234567890ab",
				BuildTags:   []string{"tcp"},
			},
			wantProfiles: []string{"tcp"},
			checkTCP:     true,
		},
		{
			name: "multiple profiles",
			config: HarnessConfig{
				PSK:         psk,
				OperationID: "op-multi",
				AgentUUID:   "uuid-multi-test-1234-567890abcdef",
				BuildTags:   []string{"http", "tcp"},
			},
			wantProfiles: []string{"http", "tcp"},
			checkHTTP:    true,
			checkTCP:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a harness with a mock server so we have an address
			h := NewHarness(tt.config)

			// Start a mock server to get an address
			serverConfig := mockafm.ServerConfig{
				PSK:         tt.config.PSK,
				OperationID: tt.config.OperationID,
			}
			h.server = mockafm.NewServer(serverConfig)
			if err := h.server.Start(0); err != nil {
				t.Fatalf("Failed to start mock server: %v", err)
			}
			defer h.server.Stop()

			h.binaryPath = "/tmp/test-agent"

			// Generate config
			jsonBytes, err := h.generateConfigJSON()
			if err != nil {
				t.Fatalf("generateConfigJSON failed: %v", err)
			}

			// Parse the generated JSON
			var config agentConfig
			if err := json.Unmarshal(jsonBytes, &config); err != nil {
				t.Fatalf("Failed to parse generated JSON: %v", err)
			}

			// Check basic fields
			if config.UUID != tt.config.AgentUUID {
				t.Errorf("UUID = %s, want %s", config.UUID, tt.config.AgentUUID)
			}
			if config.Debug != tt.config.Debug {
				t.Errorf("Debug = %v, want %v", config.Debug, tt.config.Debug)
			}
			if config.Build.OS != runtime.GOOS {
				t.Errorf("Build.OS = %s, want %s", config.Build.OS, runtime.GOOS)
			}
			if config.Build.Arch != runtime.GOARCH {
				t.Errorf("Build.Arch = %s, want %s", config.Build.Arch, runtime.GOARCH)
			}

			// Check profiles
			if len(config.Profiles) != len(tt.wantProfiles) {
				t.Errorf("Profiles length = %d, want %d", len(config.Profiles), len(tt.wantProfiles))
			}
			for i, p := range config.Profiles {
				if p != tt.wantProfiles[i] {
					t.Errorf("Profiles[%d] = %s, want %s", i, p, tt.wantProfiles[i])
				}
			}

			// Check egress config
			if len(config.Egress.Order) != len(tt.wantProfiles) {
				t.Errorf("Egress.Order length = %d, want %d", len(config.Egress.Order), len(tt.wantProfiles))
			}
			if config.Egress.Failover != "failover" {
				t.Errorf("Egress.Failover = %s, want failover", config.Egress.Failover)
			}
			if config.Egress.FailedThreshold != 10 {
				t.Errorf("Egress.FailedThreshold = %d, want 10", config.Egress.FailedThreshold)
			}

			// Check HTTP config
			if tt.checkHTTP {
				if config.HTTP == nil {
					t.Fatal("HTTP config should not be nil")
				}
				if config.HTTP.AesPsk != tt.config.PSK {
					t.Errorf("HTTP.AesPsk mismatch")
				}
				if config.HTTP.Killdate != "2099-12-31" {
					t.Errorf("HTTP.Killdate = %s, want 2099-12-31", config.HTTP.Killdate)
				}
				if config.HTTP.Interval != 1 {
					t.Errorf("HTTP.Interval = %d, want 1", config.HTTP.Interval)
				}
				if config.HTTP.Jitter != 0 {
					t.Errorf("HTTP.Jitter = %d, want 0", config.HTTP.Jitter)
				}
				wantPath := "/api/v1/operations/" + tt.config.OperationID + "/agent"
				if config.HTTP.PostUri != wantPath {
					t.Errorf("HTTP.PostUri = %s, want %s", config.HTTP.PostUri, wantPath)
				}
				if config.HTTP.GetUri != wantPath {
					t.Errorf("HTTP.GetUri = %s, want %s", config.HTTP.GetUri, wantPath)
				}
				if config.HTTP.EncryptedExchangeCheck == nil || *config.HTTP.EncryptedExchangeCheck != false {
					t.Error("HTTP.EncryptedExchangeCheck should be false")
				}
			} else {
				if config.HTTP != nil {
					t.Error("HTTP config should be nil for non-HTTP profile")
				}
			}

			// Check Websocket config
			if tt.checkWS {
				if config.Websocket == nil {
					t.Fatal("Websocket config should not be nil")
				}
				if config.Websocket.AesPsk != tt.config.PSK {
					t.Errorf("Websocket.AesPsk mismatch")
				}
				wantEndpoint := "/api/v1/operations/" + tt.config.OperationID + "/agent"
				if config.Websocket.Endpoint != wantEndpoint {
					t.Errorf("Websocket.Endpoint = %s, want %s", config.Websocket.Endpoint, wantEndpoint)
				}
			} else {
				if config.Websocket != nil {
					t.Error("Websocket config should be nil for non-websocket profile")
				}
			}

			// Check TCP config
			if tt.checkTCP {
				if config.TCP == nil {
					t.Fatal("TCP config should not be nil")
				}
				if config.TCP.AesPsk != tt.config.PSK {
					t.Errorf("TCP.AesPsk mismatch")
				}
			} else {
				if config.TCP != nil {
					t.Error("TCP config should be nil for non-TCP profile")
				}
			}
		})
	}
}

// TestHarnessGetters tests the getter methods.
func TestHarnessGetters(t *testing.T) {
	h := NewHarness(HarnessConfig{
		PSK:         "dGVzdGtleS10aGlydHktdHdvLWJ5dGVzLWxvbmc=",
		OperationID: "test-op",
		AgentUUID:   "test-uuid-1234-5678-9abc-def012345678",
	})

	// Initial state should return empty/false values
	if h.IsSetup() {
		t.Error("IsSetup() should return false initially")
	}
	if h.IsSpawned() {
		t.Error("IsSpawned() should return false initially")
	}
	if h.IsCheckedIn() {
		t.Error("IsCheckedIn() should return false initially")
	}
	if h.GetServerURL() != "" {
		t.Error("GetServerURL() should return empty initially")
	}
	if h.GetServer() != nil {
		t.Error("GetServer() should return nil initially")
	}
	if h.GetTempDir() != "" {
		t.Error("GetTempDir() should return empty initially")
	}
	if h.GetBinaryPath() != "" {
		t.Error("GetBinaryPath() should return empty initially")
	}
	if h.GetConfigPath() != "" {
		t.Error("GetConfigPath() should return empty initially")
	}
}

// TestCleanupIdempotent tests that Cleanup can be called multiple times safely.
func TestCleanupIdempotent(t *testing.T) {
	h := NewHarness(HarnessConfig{
		PSK:         "dGVzdGtleS10aGlydHktdHdvLWJ5dGVzLWxvbmc=",
		OperationID: "test-op",
		AgentUUID:   "test-uuid-1234-5678-9abc-def012345678",
	})

	// Should not panic when called multiple times
	h.Cleanup()
	h.Cleanup()
	h.Cleanup()

	// All state should be reset
	if h.isSetup {
		t.Error("isSetup should be false after Cleanup")
	}
	if h.isSpawned {
		t.Error("isSpawned should be false after Cleanup")
	}
	if h.isCheckedIn {
		t.Error("isCheckedIn should be false after Cleanup")
	}
}

// TestFindAgentCodeDir tests the agent code directory discovery.
func TestFindAgentCodeDir(t *testing.T) {
	// This test verifies that the function doesn't panic
	// and returns a reasonable result
	dir, err := findAgentCodeDir()

	// It's OK if this fails in some environments
	// (e.g., when running tests from a different location)
	if err != nil {
		t.Skipf("findAgentCodeDir returned error (may be expected in this environment): %v", err)
	}

	// If it succeeded, verify it looks like an agent_code directory
	goModPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		t.Errorf("findAgentCodeDir returned %s, but go.mod not found", dir)
	}
}

// TestHarnessWithMockServer tests harness with an actual mock server.
func TestHarnessWithMockServer(t *testing.T) {
	// Generate a valid 32-byte key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}
	psk := base64.StdEncoding.EncodeToString(testKey)

	h := NewHarness(HarnessConfig{
		PSK:         psk,
		OperationID: "integration-test",
		AgentUUID:   "harness-test-uuid-1234567890ab",
		BuildTags:   []string{"http"},
	})

	// Create temp directory manually
	tempDir, err := os.MkdirTemp("", "harness-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	h.tempDir = tempDir

	// Start mock server
	serverConfig := mockafm.ServerConfig{
		PSK:         psk,
		OperationID: "integration-test",
	}
	h.server = mockafm.NewServer(serverConfig)
	if err := h.server.Start(0); err != nil {
		t.Fatalf("Failed to start mock server: %v", err)
	}
	defer h.server.Stop()

	h.binaryPath = filepath.Join(tempDir, "agent")
	h.configPath = filepath.Join(tempDir, "config.json")

	// Generate and write config
	configJSON, err := h.generateConfigJSON()
	if err != nil {
		t.Fatalf("Failed to generate config: %v", err)
	}

	if err := os.WriteFile(h.configPath, configJSON, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Verify config file was written
	written, err := os.ReadFile(h.configPath)
	if err != nil {
		t.Fatalf("Failed to read written config: %v", err)
	}

	var writtenConfig agentConfig
	if err := json.Unmarshal(written, &writtenConfig); err != nil {
		t.Fatalf("Written config is not valid JSON: %v", err)
	}

	// Verify the URL format
	serverURL := h.GetServerURL()
	if serverURL == "" {
		t.Error("GetServerURL returned empty")
	}
	t.Logf("Server URL: %s", serverURL)

	// Note: We don't actually build or spawn the agent in unit tests
	// That would be an integration test
}
