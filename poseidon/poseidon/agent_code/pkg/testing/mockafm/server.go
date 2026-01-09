// Package mockafm provides a mock AFM server for integration testing.
package mockafm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Common errors returned by the mock server.
var (
	// ErrServerNotRunning indicates the server is not running.
	ErrServerNotRunning = errors.New("server is not running")

	// ErrTimeout indicates a timeout occurred waiting for an operation.
	ErrTimeout = errors.New("timeout waiting for operation")

	// ErrTaskNotFound indicates the requested task was not found.
	ErrTaskNotFound = errors.New("task not found")

	// ErrNoAgentConnected indicates no agent has connected yet.
	ErrNoAgentConnected = errors.New("no agent connected")
)

// Task represents a task to be sent to the agent.
type Task struct {
	ID         string
	Command    string
	Parameters string
	Timestamp  int64
}

// Response represents a response from the agent.
type Response struct {
	TaskID      string
	UserOutput  string
	Completed   bool
	Status      string
	FileBrowser interface{}
	Processes   interface{}
	Stdout      string
	Stderr      string
}

// ServerConfig holds configuration for the mock AFM server.
type ServerConfig struct {
	// PSK is the base64-encoded 32-byte pre-shared key for encryption.
	PSK string
	// OperationID is the operation ID used in the URL path.
	OperationID string
}

// MockAFMServer is a mock AFM-1 API server for integration testing.
type MockAFMServer struct {
	config ServerConfig

	mu       sync.RWMutex
	server   *http.Server
	listener net.Listener
	running  bool

	// Agent state
	agentUUID    string
	agentDBID    string
	checkinChan  chan string // Channel to signal check-in with agent UUID

	// Task queue and responses
	taskQueue     []Task
	taskQueueCond *sync.Cond
	responses     map[string]Response
	responseConds map[string]*sync.Cond
}

// NewServer creates a new mock AFM server with the given configuration.
func NewServer(config ServerConfig) *MockAFMServer {
	s := &MockAFMServer{
		config:        config,
		checkinChan:   make(chan string, 1),
		taskQueue:     make([]Task, 0),
		responses:     make(map[string]Response),
		responseConds: make(map[string]*sync.Cond),
		agentDBID:     "mock-agent-db-id-12345",
	}
	s.taskQueueCond = sync.NewCond(&s.mu)
	return s
}

// Start starts the server on the specified port.
// Use port 0 to let the system choose an available port.
func (s *MockAFMServer) Start(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Create listener
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	// Create HTTP server
	mux := http.NewServeMux()
	// Register the agent endpoint with operation ID
	pattern := fmt.Sprintf("/api/v1/operations/%s/agent", s.config.OperationID)
	mux.HandleFunc(pattern, s.handleAgentRequest)
	// Also register a catch-all pattern for flexibility
	mux.HandleFunc("/", s.handleAgentRequest)

	s.server = &http.Server{
		Handler: mux,
	}

	s.running = true

	// Start serving in background
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			// Log error but don't panic - server might be shutting down
			fmt.Printf("mockafm server error: %v\n", err)
		}
	}()

	return nil
}

// Stop gracefully stops the server.
func (s *MockAFMServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	// Wake up any waiting goroutines
	s.taskQueueCond.Broadcast()
	for _, cond := range s.responseConds {
		cond.Broadcast()
	}

	return nil
}

// GetAddr returns the server's address (host:port).
func (s *MockAFMServer) GetAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// WaitForCheckin blocks until an agent checks in or the timeout expires.
// Returns the agent UUID on success.
func (s *MockAFMServer) WaitForCheckin(timeout time.Duration) (string, error) {
	select {
	case uuid := <-s.checkinChan:
		return uuid, nil
	case <-time.After(timeout):
		return "", ErrTimeout
	}
}

// QueueTask adds a task to the queue for the agent to receive.
func (s *MockAFMServer) QueueTask(taskID, command, parameters string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := Task{
		ID:         taskID,
		Command:    command,
		Parameters: parameters,
		Timestamp:  time.Now().Unix(),
	}
	s.taskQueue = append(s.taskQueue, task)

	// Create condition variable for this task's response
	s.responseConds[taskID] = sync.NewCond(&s.mu)

	// Signal that new tasks are available
	s.taskQueueCond.Broadcast()
}

// WaitForResponse waits for a response to a specific task.
func (s *MockAFMServer) WaitForResponse(taskID string, timeout time.Duration) (Response, error) {
	deadline := time.Now().Add(timeout)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get or create condition variable for this task
	cond, ok := s.responseConds[taskID]
	if !ok {
		cond = sync.NewCond(&s.mu)
		s.responseConds[taskID] = cond
	}

	// Create a timer for the overall timeout with a done channel for cleanup
	timer := time.NewTimer(timeout)
	done := make(chan struct{})
	timerFired := false

	go func() {
		select {
		case <-timer.C:
			s.mu.Lock()
			timerFired = true
			cond.Broadcast()
			s.mu.Unlock()
		case <-done:
			// Function returned, stop waiting
		}
	}()

	defer func() {
		timer.Stop()
		close(done)
	}()

	for {
		// Check if response is available
		if resp, ok := s.responses[taskID]; ok {
			return resp, nil
		}

		// Check timeout
		if time.Now().After(deadline) || timerFired {
			return Response{}, ErrTimeout
		}

		cond.Wait()
	}
}

// GetAgentUUID returns the UUID of the connected agent.
func (s *MockAFMServer) GetAgentUUID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agentUUID
}

// GetResponses returns all collected responses.
func (s *MockAFMServer) GetResponses() map[string]Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]Response)
	for k, v := range s.responses {
		result[k] = v
	}
	return result
}

// handleAgentRequest handles incoming requests from the agent.
func (s *MockAFMServer) handleAgentRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decrypt the message
	uuid, bodyMap, err := DecryptAgentMessage(string(body), s.config.PSK)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to decrypt message: %v", err), http.StatusBadRequest)
		return
	}

	// Determine action
	action, _ := bodyMap["action"].(string)

	var response interface{}
	switch action {
	case "checkin":
		response = s.handleCheckin(uuid, bodyMap)
	case "get_tasking":
		response = s.handleGetTasking(uuid, bodyMap)
	default:
		// Default to get_tasking behavior for poll messages
		response = s.handleGetTasking(uuid, bodyMap)
	}

	// Encrypt and send response
	s.mu.RLock()
	agentUUID := s.agentUUID
	if agentUUID == "" {
		agentUUID = uuid
	}
	s.mu.RUnlock()

	encrypted, err := EncryptAgentResponse(agentUUID, response, s.config.PSK)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encrypt response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(encrypted))
}

// handleCheckin processes a check-in message from the agent.
func (s *MockAFMServer) handleCheckin(uuid string, body map[string]interface{}) map[string]interface{} {
	s.mu.Lock()
	s.agentUUID = uuid
	agentDBID := s.agentDBID
	s.mu.Unlock()

	// Signal check-in (non-blocking)
	select {
	case s.checkinChan <- uuid:
	default:
	}

	return map[string]interface{}{
		"action": "checkin",
		"status": "success",
		"id":     agentDBID,
	}
}

// handleGetTasking processes a get_tasking/poll message from the agent.
func (s *MockAFMServer) handleGetTasking(uuid string, body map[string]interface{}) map[string]interface{} {
	// Process any responses in the incoming message
	s.processResponses(body)

	// Get queued tasks
	s.mu.Lock()
	tasks := make([]map[string]interface{}, len(s.taskQueue))
	for i, task := range s.taskQueue {
		tasks[i] = map[string]interface{}{
			"id":         task.ID,
			"command":    task.Command,
			"parameters": task.Parameters,
			"timestamp":  float64(task.Timestamp),
		}
	}
	// Clear the queue after sending
	s.taskQueue = s.taskQueue[:0]
	s.mu.Unlock()

	return map[string]interface{}{
		"action": "get_tasking",
		"tasks":  tasks,
	}
}

// processResponses extracts and stores responses from agent messages.
func (s *MockAFMServer) processResponses(body map[string]interface{}) {
	responses, ok := body["responses"].([]interface{})
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range responses {
		respMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		taskID, _ := respMap["task_id"].(string)
		if taskID == "" {
			continue
		}

		resp := Response{
			TaskID: taskID,
		}

		if v, ok := respMap["user_output"].(string); ok {
			resp.UserOutput = v
		}
		if v, ok := respMap["completed"].(bool); ok {
			resp.Completed = v
		}
		if v, ok := respMap["status"].(string); ok {
			resp.Status = v
		}
		if v, ok := respMap["file_browser"]; ok {
			resp.FileBrowser = v
		}
		if v, ok := respMap["processes"]; ok {
			resp.Processes = v
		}
		if v, ok := respMap["stdout"].(string); ok {
			resp.Stdout = v
		}
		if v, ok := respMap["stderr"].(string); ok {
			resp.Stderr = v
		}

		// Store response (update if exists, agent may send multiple updates)
		existing, hasExisting := s.responses[taskID]
		if hasExisting {
			// Merge: keep non-empty values, prefer newer completed state
			if resp.UserOutput != "" {
				existing.UserOutput += resp.UserOutput
			}
			if resp.Completed {
				existing.Completed = true
			}
			if resp.Status != "" {
				existing.Status = resp.Status
			}
			if resp.FileBrowser != nil {
				existing.FileBrowser = resp.FileBrowser
			}
			if resp.Processes != nil {
				existing.Processes = resp.Processes
			}
			if resp.Stdout != "" {
				existing.Stdout += resp.Stdout
			}
			if resp.Stderr != "" {
				existing.Stderr += resp.Stderr
			}
			s.responses[taskID] = existing
		} else {
			s.responses[taskID] = resp
		}

		// Signal waiters for this task
		if cond, ok := s.responseConds[taskID]; ok {
			cond.Broadcast()
		}
	}
}

// Reset clears all server state (tasks, responses, agent info).
func (s *MockAFMServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agentUUID = ""
	s.taskQueue = s.taskQueue[:0]
	s.responses = make(map[string]Response)

	// Drain the checkin channel
	select {
	case <-s.checkinChan:
	default:
	}
}

// SetAgentDBID sets the database ID to return for agent check-ins.
func (s *MockAFMServer) SetAgentDBID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentDBID = id
}

// GetPendingTaskCount returns the number of tasks waiting to be sent.
func (s *MockAFMServer) GetPendingTaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.taskQueue)
}

// HasResponse checks if a response has been received for a task.
func (s *MockAFMServer) HasResponse(taskID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.responses[taskID]
	return ok
}

// GetResponse returns the response for a task without waiting.
func (s *MockAFMServer) GetResponse(taskID string) (Response, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resp, ok := s.responses[taskID]
	return resp, ok
}

// IsRunning returns whether the server is currently running.
func (s *MockAFMServer) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetURL returns the full URL for the agent endpoint.
func (s *MockAFMServer) GetURL() string {
	addr := s.GetAddr()
	if addr == "" {
		return ""
	}
	// Ensure addr has proper format
	if !strings.Contains(addr, ":") {
		return ""
	}
	return fmt.Sprintf("http://%s/api/v1/operations/%s/agent", addr, s.config.OperationID)
}
