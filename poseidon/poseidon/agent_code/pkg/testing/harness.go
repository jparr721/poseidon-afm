// Package testing provides integration testing utilities for the Poseidon agent.
package testing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/commands"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

// Common errors returned by the harness.
var (
	// ErrNotSetup indicates the harness has not been set up yet.
	ErrNotSetup = errors.New("harness not set up")

	// ErrAgentNotSpawned indicates the agent has not been spawned yet.
	ErrAgentNotSpawned = errors.New("agent not spawned")

	// ErrAgentNotCheckedIn indicates the agent has not checked in yet.
	ErrAgentNotCheckedIn = errors.New("agent not checked in")

	// ErrBuildFailed indicates the agent build failed.
	ErrBuildFailed = errors.New("agent build failed")

	// ErrAgentStartFailed indicates the agent failed to start.
	ErrAgentStartFailed = errors.New("agent failed to start")

	// ErrCommandFailed indicates a command execution failed.
	ErrCommandFailed = errors.New("command execution failed")
)

// HarnessConfig holds configuration for the test harness.
type HarnessConfig struct {
	// PSK is the base64-encoded 32-byte pre-shared key for encryption.
	PSK string

	// OperationID is the operation ID used in the URL path.
	OperationID string

	// AgentUUID is the UUID of the agent (36 characters).
	AgentUUID string

	// BuildTags are the build tags to use (e.g., ["http"]).
	BuildTags []string

	// Debug enables debug output from the agent.
	Debug bool

	// AgentCodeDir is the path to the agent_code directory.
	// If empty, it will be auto-detected.
	AgentCodeDir string

	// BuildTimeout is the timeout for the build process.
	// Default is 2 minutes.
	BuildTimeout time.Duration

	// AgentStartTimeout is the timeout for the agent to start.
	// Default is 10 seconds.
	AgentStartTimeout time.Duration
}

// Harness orchestrates integration tests: build -> spawn -> test -> cleanup.
type Harness struct {
	config HarnessConfig

	mu           sync.RWMutex
	server       *mockafm.MockAFMServer
	agentCmd     *exec.Cmd
	agentCancel  context.CancelFunc
	tempDir      string
	configPath   string
	binaryPath   string
	isSetup      bool
	isSpawned    bool
	isCheckedIn  bool
	agentCodeDir string
}

// NewHarness creates a new test harness with the given configuration.
func NewHarness(config HarnessConfig) *Harness {
	// Apply defaults
	if config.BuildTimeout == 0 {
		config.BuildTimeout = 2 * time.Minute
	}
	if config.AgentStartTimeout == 0 {
		config.AgentStartTimeout = 10 * time.Second
	}
	if len(config.BuildTags) == 0 {
		config.BuildTags = []string{"http"}
	}

	return &Harness{
		config: config,
	}
}

// Setup starts the mock server and builds the agent.
// This must be called before SpawnAgent.
func (h *Harness) Setup() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.isSetup {
		return nil
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "poseidon-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	h.tempDir = tempDir

	// Determine agent_code directory
	agentCodeDir := h.config.AgentCodeDir
	if agentCodeDir == "" {
		agentCodeDir, err = findAgentCodeDir()
		if err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to find agent_code directory: %w", err)
		}
	}
	h.agentCodeDir = agentCodeDir

	// Start mock server
	serverConfig := mockafm.ServerConfig{
		PSK:         h.config.PSK,
		OperationID: h.config.OperationID,
	}
	h.server = mockafm.NewServer(serverConfig)
	if err := h.server.Start(0); err != nil {
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to start mock server: %w", err)
	}

	// Generate config file
	h.configPath = filepath.Join(tempDir, "agent_config.json")
	h.binaryPath = filepath.Join(tempDir, "agent")
	if runtime.GOOS == "windows" {
		h.binaryPath += ".exe"
	}

	configJSON, err := h.generateConfigJSON()
	if err != nil {
		h.server.Stop()
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if err := os.WriteFile(h.configPath, configJSON, 0644); err != nil {
		h.server.Stop()
		os.RemoveAll(tempDir)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Build the agent
	if err := h.buildAgent(); err != nil {
		h.server.Stop()
		os.RemoveAll(tempDir)
		return fmt.Errorf("%w: %v", ErrBuildFailed, err)
	}

	h.isSetup = true
	return nil
}

// SpawnAgent starts the agent binary.
// Setup must be called before this.
func (h *Harness) SpawnAgent() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isSetup {
		return ErrNotSetup
	}

	if h.isSpawned {
		return nil
	}

	// Create a context with cancel for the agent process
	ctx, cancel := context.WithCancel(context.Background())
	h.agentCancel = cancel

	// Start the agent
	h.agentCmd = exec.CommandContext(ctx, h.binaryPath)
	h.agentCmd.Dir = h.tempDir

	// Capture output for debugging
	if h.config.Debug {
		h.agentCmd.Stdout = os.Stdout
		h.agentCmd.Stderr = os.Stderr
	}

	if err := h.agentCmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("%w: %v", ErrAgentStartFailed, err)
	}

	h.isSpawned = true
	return nil
}

// WaitForCheckin waits for the agent to check in with the mock server.
func (h *Harness) WaitForCheckin(timeout time.Duration) error {
	h.mu.RLock()
	if !h.isSetup {
		h.mu.RUnlock()
		return ErrNotSetup
	}
	if !h.isSpawned {
		h.mu.RUnlock()
		return ErrAgentNotSpawned
	}
	server := h.server
	h.mu.RUnlock()

	_, err := server.WaitForCheckin(timeout)
	if err != nil {
		return fmt.Errorf("agent check-in failed: %w", err)
	}

	h.mu.Lock()
	h.isCheckedIn = true
	h.mu.Unlock()

	return nil
}

// RunCommand queues a command and waits for a response.
func (h *Harness) RunCommand(cmd commands.CommandTest, timeout time.Duration) (mockafm.Response, error) {
	h.mu.RLock()
	if !h.isSetup {
		h.mu.RUnlock()
		return mockafm.Response{}, ErrNotSetup
	}
	if !h.isSpawned {
		h.mu.RUnlock()
		return mockafm.Response{}, ErrAgentNotSpawned
	}
	if !h.isCheckedIn {
		h.mu.RUnlock()
		return mockafm.Response{}, ErrAgentNotCheckedIn
	}
	server := h.server
	tempDir := h.tempDir
	h.mu.RUnlock()

	// Run setup if provided
	if cmd.Setup != nil {
		if err := cmd.Setup(tempDir); err != nil {
			return mockafm.Response{}, fmt.Errorf("command setup failed: %w", err)
		}
	}

	// Generate a task ID
	taskID := uuid.New().String()

	// Queue the task
	server.QueueTask(taskID, cmd.Name, cmd.Parameters)

	// Wait for response
	resp, err := server.WaitForResponse(taskID, timeout)
	if err != nil {
		// Run teardown even on error
		if cmd.Teardown != nil {
			cmd.Teardown(tempDir)
		}
		return mockafm.Response{}, fmt.Errorf("%w: %v", ErrCommandFailed, err)
	}

	// Run teardown if provided
	if cmd.Teardown != nil {
		if err := cmd.Teardown(tempDir); err != nil {
			// Log but don't fail - the command succeeded
			fmt.Printf("Warning: command teardown failed: %v\n", err)
		}
	}

	return resp, nil
}

// Cleanup stops the agent and server, removes temp files.
func (h *Harness) Cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Try to gracefully stop the agent with exit command
	if h.isCheckedIn && h.server != nil {
		h.sendExitCommand()
	}

	// Kill agent process if still running
	if h.agentCmd != nil && h.agentCmd.Process != nil {
		if h.agentCancel != nil {
			h.agentCancel()
		}

		// Give the process a moment to exit gracefully
		done := make(chan struct{})
		go func() {
			h.agentCmd.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Process exited
		case <-time.After(5 * time.Second):
			// Force kill
			h.agentCmd.Process.Kill()
		}
	}

	// Stop the server
	if h.server != nil {
		h.server.Stop()
	}

	// Remove temp directory
	if h.tempDir != "" {
		os.RemoveAll(h.tempDir)
	}

	// Reset state
	h.isSetup = false
	h.isSpawned = false
	h.isCheckedIn = false
	h.server = nil
	h.agentCmd = nil
	h.agentCancel = nil
	h.tempDir = ""
	h.configPath = ""
	h.binaryPath = ""
}

// GetServerURL returns the mock server's URL.
func (h *Harness) GetServerURL() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.server == nil {
		return ""
	}
	return h.server.GetURL()
}

// GetServer returns the underlying mock server for advanced testing.
func (h *Harness) GetServer() *mockafm.MockAFMServer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.server
}

// GetTempDir returns the temporary directory used by the harness.
func (h *Harness) GetTempDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.tempDir
}

// GetBinaryPath returns the path to the built agent binary.
func (h *Harness) GetBinaryPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.binaryPath
}

// GetConfigPath returns the path to the generated config file.
func (h *Harness) GetConfigPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.configPath
}

// IsSetup returns whether the harness has been set up.
func (h *Harness) IsSetup() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isSetup
}

// IsSpawned returns whether the agent has been spawned.
func (h *Harness) IsSpawned() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isSpawned
}

// IsCheckedIn returns whether the agent has checked in.
func (h *Harness) IsCheckedIn() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isCheckedIn
}

// generateConfigJSON generates the agent configuration JSON.
func (h *Harness) generateConfigJSON() ([]byte, error) {
	// Parse server address
	addr := h.server.GetAddr()
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid server address: %s", addr)
	}
	host := parts[0]
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid server port: %s", parts[1])
	}

	// Build the config based on profiles
	config := &agentConfig{
		UUID:     h.config.AgentUUID,
		Debug:    h.config.Debug,
		Profiles: h.config.BuildTags,
		Build: buildConfig{
			OS:     runtime.GOOS,
			Arch:   runtime.GOARCH,
			Output: h.binaryPath,
		},
		Egress: egressConfig{
			Order:           h.config.BuildTags,
			Failover:        "failover",
			FailedThreshold: 10,
		},
	}

	// Add profile-specific configuration
	for _, profile := range h.config.BuildTags {
		switch profile {
		case "http":
			encryptedExchangeCheck := false
			config.HTTP = &httpConfig{
				CallbackHost:           fmt.Sprintf("http://%s", host),
				CallbackPort:           port,
				AesPsk:                 h.config.PSK,
				Killdate:               "2099-12-31",
				Interval:               1,
				Jitter:                 0,
				PostUri:                fmt.Sprintf("/api/v1/operations/%s/agent", h.config.OperationID),
				GetUri:                 fmt.Sprintf("/api/v1/operations/%s/agent", h.config.OperationID),
				EncryptedExchangeCheck: &encryptedExchangeCheck,
			}
		case "websocket":
			encryptedExchangeCheck := false
			config.Websocket = &websocketConfig{
				CallbackHost:           fmt.Sprintf("ws://%s", host),
				CallbackPort:           port,
				AesPsk:                 h.config.PSK,
				Killdate:               "2099-12-31",
				Interval:               1,
				Jitter:                 0,
				Endpoint:               fmt.Sprintf("/api/v1/operations/%s/agent", h.config.OperationID),
				EncryptedExchangeCheck: &encryptedExchangeCheck,
			}
		case "tcp":
			encryptedExchangeCheck := false
			config.TCP = &tcpConfig{
				Port:                   port,
				AesPsk:                 h.config.PSK,
				Killdate:               "2099-12-31",
				EncryptedExchangeCheck: &encryptedExchangeCheck,
			}
		}
	}

	return json.MarshalIndent(config, "", "  ")
}

// buildAgent builds the agent using the builder tool.
func (h *Harness) buildAgent() error {
	ctx, cancel := context.WithTimeout(context.Background(), h.config.BuildTimeout)
	defer cancel()

	// Build using go run ./cmd/builder
	args := []string{
		"run",
		"./cmd/builder",
		"--config", h.configPath,
		"--output", h.binaryPath,
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = h.agentCodeDir

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build command failed: %w\nOutput: %s", err, string(output))
	}

	// Verify binary exists
	if _, err := os.Stat(h.binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found after build: %s", h.binaryPath)
	}

	return nil
}

// sendExitCommand sends an exit command to gracefully stop the agent.
func (h *Harness) sendExitCommand() {
	taskID := uuid.New().String()
	h.server.QueueTask(taskID, "exit", "{}")

	// Wait briefly for the exit to process
	h.server.WaitForResponse(taskID, 2*time.Second)
}

// findAgentCodeDir attempts to find the agent_code directory.
func findAgentCodeDir() (string, error) {
	// Try current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check if we're already in agent_code
	if filepath.Base(cwd) == "agent_code" {
		return cwd, nil
	}

	// Check if agent_code is a subdirectory
	agentCodePath := filepath.Join(cwd, "agent_code")
	if _, err := os.Stat(agentCodePath); err == nil {
		return agentCodePath, nil
	}

	// Try to find it relative to this file (for tests)
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		// This file is in pkg/testing/harness.go
		// agent_code is two levels up
		agentCodePath = filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
		if _, err := os.Stat(filepath.Join(agentCodePath, "go.mod")); err == nil {
			return agentCodePath, nil
		}
	}

	// Look for it in parent directories
	dir := cwd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "poseidon", "poseidon", "agent_code")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		candidate = filepath.Join(dir, "agent_code")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("could not find agent_code directory")
}

// Config types for JSON generation

type agentConfig struct {
	UUID      string           `json:"uuid"`
	Debug     bool             `json:"debug"`
	Build     buildConfig      `json:"build"`
	Profiles  []string         `json:"profiles"`
	Egress    egressConfig     `json:"egress"`
	HTTP      *httpConfig      `json:"http,omitempty"`
	Websocket *websocketConfig `json:"websocket,omitempty"`
	TCP       *tcpConfig       `json:"tcp,omitempty"`
}

type buildConfig struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Output string `json:"output"`
}

type egressConfig struct {
	Order           []string `json:"order"`
	Failover        string   `json:"failover"`
	FailedThreshold int      `json:"failedThreshold"`
}

type httpConfig struct {
	CallbackHost           string `json:"callbackHost"`
	CallbackPort           int    `json:"callbackPort"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	Interval               int    `json:"interval"`
	Jitter                 int    `json:"jitter"`
	PostUri                string `json:"postUri"`
	GetUri                 string `json:"getUri"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
}

type websocketConfig struct {
	CallbackHost           string `json:"callbackHost"`
	CallbackPort           int    `json:"callbackPort"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	Interval               int    `json:"interval"`
	Jitter                 int    `json:"jitter"`
	Endpoint               string `json:"endpoint"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
}

type tcpConfig struct {
	Port                   int    `json:"port"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
}
