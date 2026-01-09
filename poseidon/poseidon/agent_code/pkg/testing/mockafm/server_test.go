package mockafm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/crypto"
)

// Test configuration
var testServerConfig = ServerConfig{
	PSK:         base64.StdEncoding.EncodeToString(make([]byte, 32)), // 32 bytes of zeros
	OperationID: "test-operation-123",
}

func TestServerStartStop(t *testing.T) {
	server := NewServer(testServerConfig)

	// Start server
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify running
	if !server.IsRunning() {
		t.Error("server should be running after Start")
	}

	// Get address
	addr := server.GetAddr()
	if addr == "" {
		t.Error("GetAddr returned empty string")
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify stopped
	if server.IsRunning() {
		t.Error("server should not be running after Stop")
	}
}

func TestServerDoubleStartStop(t *testing.T) {
	server := NewServer(testServerConfig)

	// Double start should be safe
	if err := server.Start(0); err != nil {
		t.Fatalf("First Start failed: %v", err)
	}
	if err := server.Start(0); err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	// Double stop should be safe
	if err := server.Stop(); err != nil {
		t.Fatalf("First Stop failed: %v", err)
	}
	if err := server.Stop(); err != nil {
		t.Fatalf("Second Stop failed: %v", err)
	}
}

func TestServerCheckin(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Send checkin message
	agentUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	checkinBody := map[string]interface{}{
		"action": "checkin",
		"ips":    []string{"192.168.1.100"},
		"os":     "linux",
		"user":   "testuser",
		"host":   "testhost",
		"pid":    1234,
	}

	resp, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("sendAgentMessage failed: %v", err)
	}

	// Verify response
	status, _ := resp["status"].(string)
	if status != "success" {
		t.Errorf("expected status 'success', got %q", status)
	}

	id, _ := resp["id"].(string)
	if id == "" {
		t.Error("expected non-empty id in response")
	}

	// Verify server recorded the agent
	if server.GetAgentUUID() != agentUUID {
		t.Errorf("GetAgentUUID: got %q, want %q", server.GetAgentUUID(), agentUUID)
	}
}

func TestWaitForCheckin(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// Start waiting for checkin in goroutine
	var wg sync.WaitGroup
	var checkinUUID string
	var checkinErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		checkinUUID, checkinErr = server.WaitForCheckin(5 * time.Second)
	}()

	// Give the goroutine time to start waiting
	time.Sleep(100 * time.Millisecond)

	// Send checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("sendAgentMessage failed: %v", err)
	}

	// Wait for goroutine
	wg.Wait()

	if checkinErr != nil {
		t.Fatalf("WaitForCheckin failed: %v", checkinErr)
	}
	if checkinUUID != agentUUID {
		t.Errorf("WaitForCheckin: got %q, want %q", checkinUUID, agentUUID)
	}
}

func TestWaitForCheckinTimeout(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Wait with short timeout, no checkin
	_, err := server.WaitForCheckin(100 * time.Millisecond)
	if err != ErrTimeout {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
}

func TestQueueTaskAndGetTasking(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// First, do checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	// Queue a task
	server.QueueTask("task-001", "shell", `{"command": "whoami"}`)

	// Verify pending count
	if count := server.GetPendingTaskCount(); count != 1 {
		t.Errorf("GetPendingTaskCount: got %d, want 1", count)
	}

	// Get tasking
	taskingBody := map[string]interface{}{
		"action":       "get_tasking",
		"tasking_size": -1,
	}
	resp, err := sendAgentMessage(server.GetURL(), agentUUID, taskingBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("get_tasking failed: %v", err)
	}

	// Verify tasks in response
	tasks, ok := resp["tasks"].([]interface{})
	if !ok {
		t.Fatal("tasks not in response")
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0].(map[string]interface{})
	if task["id"] != "task-001" {
		t.Errorf("task id: got %v, want task-001", task["id"])
	}
	if task["command"] != "shell" {
		t.Errorf("task command: got %v, want shell", task["command"])
	}

	// Verify queue is now empty
	if count := server.GetPendingTaskCount(); count != 0 {
		t.Errorf("GetPendingTaskCount after fetch: got %d, want 0", count)
	}
}

func TestMultipleTasks(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// Checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	// Queue multiple tasks
	server.QueueTask("task-001", "shell", `{"command": "whoami"}`)
	server.QueueTask("task-002", "pwd", "")
	server.QueueTask("task-003", "ls", `{"path": "/tmp"}`)

	if count := server.GetPendingTaskCount(); count != 3 {
		t.Errorf("GetPendingTaskCount: got %d, want 3", count)
	}

	// Get tasking
	taskingBody := map[string]interface{}{
		"action":       "get_tasking",
		"tasking_size": -1,
	}
	resp, err := sendAgentMessage(server.GetURL(), agentUUID, taskingBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("get_tasking failed: %v", err)
	}

	tasks, _ := resp["tasks"].([]interface{})
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestResponseCollection(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// Checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	// Send response with get_tasking
	responseBody := map[string]interface{}{
		"action":       "get_tasking",
		"tasking_size": -1,
		"responses": []interface{}{
			map[string]interface{}{
				"task_id":     "task-001",
				"user_output": "root",
				"completed":   true,
				"status":      "success",
			},
		},
	}
	_, err = sendAgentMessage(server.GetURL(), agentUUID, responseBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("send response failed: %v", err)
	}

	// Verify response was collected
	if !server.HasResponse("task-001") {
		t.Error("expected response for task-001")
	}

	resp, ok := server.GetResponse("task-001")
	if !ok {
		t.Fatal("GetResponse failed")
	}

	if resp.TaskID != "task-001" {
		t.Errorf("TaskID: got %q, want task-001", resp.TaskID)
	}
	if resp.UserOutput != "root" {
		t.Errorf("UserOutput: got %q, want root", resp.UserOutput)
	}
	if !resp.Completed {
		t.Error("expected Completed to be true")
	}
	if resp.Status != "success" {
		t.Errorf("Status: got %q, want success", resp.Status)
	}
}

func TestWaitForResponse(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// Checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	// Queue a task first (creates the condition variable)
	server.QueueTask("task-001", "shell", `{"command": "whoami"}`)

	// Start waiting for response in goroutine
	var wg sync.WaitGroup
	var respResult Response
	var respErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		respResult, respErr = server.WaitForResponse("task-001", 5*time.Second)
	}()

	// Give the goroutine time to start waiting
	time.Sleep(100 * time.Millisecond)

	// Send response
	responseBody := map[string]interface{}{
		"action": "get_tasking",
		"responses": []interface{}{
			map[string]interface{}{
				"task_id":     "task-001",
				"user_output": "test output",
				"completed":   true,
			},
		},
	}
	_, err = sendAgentMessage(server.GetURL(), agentUUID, responseBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("send response failed: %v", err)
	}

	// Wait for goroutine
	wg.Wait()

	if respErr != nil {
		t.Fatalf("WaitForResponse failed: %v", respErr)
	}
	if respResult.TaskID != "task-001" {
		t.Errorf("TaskID: got %q, want task-001", respResult.TaskID)
	}
	if respResult.UserOutput != "test output" {
		t.Errorf("UserOutput: got %q, want 'test output'", respResult.UserOutput)
	}
}

func TestWaitForResponseTimeout(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Wait for response that never comes
	_, err := server.WaitForResponse("nonexistent-task", 100*time.Millisecond)
	if err != ErrTimeout {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
}

func TestResponseMerging(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// Checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	// Send first partial response
	response1 := map[string]interface{}{
		"action": "get_tasking",
		"responses": []interface{}{
			map[string]interface{}{
				"task_id":     "task-001",
				"user_output": "partial1",
				"completed":   false,
			},
		},
	}
	_, err = sendAgentMessage(server.GetURL(), agentUUID, response1, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("send response1 failed: %v", err)
	}

	// Send second partial response
	response2 := map[string]interface{}{
		"action": "get_tasking",
		"responses": []interface{}{
			map[string]interface{}{
				"task_id":     "task-001",
				"user_output": "partial2",
				"completed":   true,
				"status":      "success",
			},
		},
	}
	_, err = sendAgentMessage(server.GetURL(), agentUUID, response2, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("send response2 failed: %v", err)
	}

	// Verify merged response
	resp, ok := server.GetResponse("task-001")
	if !ok {
		t.Fatal("GetResponse failed")
	}

	// Output should be concatenated
	if resp.UserOutput != "partial1partial2" {
		t.Errorf("UserOutput: got %q, want 'partial1partial2'", resp.UserOutput)
	}
	// Completed should be true (from second response)
	if !resp.Completed {
		t.Error("expected Completed to be true")
	}
	// Status should be set
	if resp.Status != "success" {
		t.Errorf("Status: got %q, want success", resp.Status)
	}
}

func TestServerReset(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	agentUUID := "12345678-1234-1234-1234-123456789012"

	// Checkin
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	_, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	// Queue task
	server.QueueTask("task-001", "shell", "")

	// Reset
	server.Reset()

	// Verify state is cleared
	if server.GetAgentUUID() != "" {
		t.Error("agent UUID should be empty after reset")
	}
	if server.GetPendingTaskCount() != 0 {
		t.Error("task queue should be empty after reset")
	}
	if len(server.GetResponses()) != 0 {
		t.Error("responses should be empty after reset")
	}
}

func TestSetAgentDBID(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	customDBID := "custom-db-id-12345"
	server.SetAgentDBID(customDBID)

	agentUUID := "12345678-1234-1234-1234-123456789012"
	checkinBody := map[string]interface{}{
		"action": "checkin",
	}
	resp, err := sendAgentMessage(server.GetURL(), agentUUID, checkinBody, testServerConfig.PSK)
	if err != nil {
		t.Fatalf("checkin failed: %v", err)
	}

	id, _ := resp["id"].(string)
	if id != customDBID {
		t.Errorf("id: got %q, want %q", id, customDBID)
	}
}

func TestGetURL(t *testing.T) {
	server := NewServer(testServerConfig)

	// Before starting, URL should be empty
	if url := server.GetURL(); url != "" {
		t.Errorf("GetURL before start: got %q, want empty", url)
	}

	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	url := server.GetURL()
	if url == "" {
		t.Error("GetURL returned empty after start")
	}

	// Verify URL contains expected path
	expectedPath := "/api/v1/operations/test-operation-123/agent"
	if !containsPath(url, expectedPath) {
		t.Errorf("URL %q does not contain expected path %q", url, expectedPath)
	}
}

func TestConcurrentTaskQueueing(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Queue tasks concurrently
	var wg sync.WaitGroup
	numTasks := 100

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := "task-" + string(rune('A'+id%26)) + "-" + string(rune('0'+id%10))
			server.QueueTask(taskID, "shell", "")
		}(i)
	}

	wg.Wait()

	if count := server.GetPendingTaskCount(); count != numTasks {
		t.Errorf("GetPendingTaskCount: got %d, want %d", count, numTasks)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Send GET request instead of POST
	resp, err := http.Get(server.GetURL())
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, resp.StatusCode)
	}
}

func TestInvalidEncryption(t *testing.T) {
	server := NewServer(testServerConfig)
	if err := server.Start(0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer server.Stop()

	// Send invalid (not encrypted) body
	resp, err := http.Post(server.GetURL(), "text/plain", bytes.NewReader([]byte("not encrypted data")))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

// Helper function to send an encrypted agent message
func sendAgentMessage(url, uuid string, body map[string]interface{}, psk string) (map[string]interface{}, error) {
	// Marshal body
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// Encrypt like the agent does
	key, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		return nil, err
	}
	encrypted := crypto.AesEncrypt(key, jsonBytes)

	// Prepend UUID
	message := append([]byte(uuid), encrypted...)

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(message)

	// Send request
	resp, err := http.Post(url, "text/plain", bytes.NewReader([]byte(encoded)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(resp.Body) // drain body
		return nil, http.ErrServerClosed // Use an error type that indicates failure
	}

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Decrypt response
	_, respMap, err := DecryptAgentMessage(string(respBody), psk)
	if err != nil {
		return nil, err
	}

	return respMap, nil
}

func containsPath(url, path string) bool {
	return len(url) > len(path) && url[len(url)-len(path):] == path ||
		   len(url) >= len(path) && url[len(url)-len(path):] == path
}
