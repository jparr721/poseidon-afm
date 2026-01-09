package mockafm

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/crypto"
)

// Test PSK: 32 bytes of zeros, base64-encoded
var testPSK = base64.StdEncoding.EncodeToString(make([]byte, 32))

// Test UUID (36 characters)
const testUUID = "12345678-1234-1234-1234-123456789012"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Test data
	body := map[string]interface{}{
		"action": "checkin",
		"data":   "test data",
		"number": float64(42), // JSON numbers are float64
	}

	// Encrypt
	encrypted, err := EncryptAgentResponse(testUUID, body, testPSK)
	if err != nil {
		t.Fatalf("EncryptAgentResponse failed: %v", err)
	}

	// Verify encrypted message is not empty
	if encrypted == "" {
		t.Fatal("encrypted message is empty")
	}

	// Decrypt
	uuid, decryptedBody, err := DecryptAgentMessage(encrypted, testPSK)
	if err != nil {
		t.Fatalf("DecryptAgentMessage failed: %v", err)
	}

	// Verify UUID
	if uuid != testUUID {
		t.Errorf("UUID mismatch: got %q, want %q", uuid, testUUID)
	}

	// Verify body contents
	if decryptedBody["action"] != body["action"] {
		t.Errorf("action mismatch: got %v, want %v", decryptedBody["action"], body["action"])
	}
	if decryptedBody["data"] != body["data"] {
		t.Errorf("data mismatch: got %v, want %v", decryptedBody["data"], body["data"])
	}
	if decryptedBody["number"] != body["number"] {
		t.Errorf("number mismatch: got %v, want %v", decryptedBody["number"], body["number"])
	}
}

func TestEncryptDecryptComplexBody(t *testing.T) {
	// Test with nested structures
	body := map[string]interface{}{
		"action": "get_tasking",
		"tasks": []interface{}{
			map[string]interface{}{
				"command": "shell",
				"args":    "whoami",
			},
		},
		"metadata": map[string]interface{}{
			"host": "testhost",
			"pid":  float64(1234),
		},
	}

	encrypted, err := EncryptAgentResponse(testUUID, body, testPSK)
	if err != nil {
		t.Fatalf("EncryptAgentResponse failed: %v", err)
	}

	uuid, decryptedBody, err := DecryptAgentMessage(encrypted, testPSK)
	if err != nil {
		t.Fatalf("DecryptAgentMessage failed: %v", err)
	}

	if uuid != testUUID {
		t.Errorf("UUID mismatch: got %q, want %q", uuid, testUUID)
	}

	// Verify nested data
	tasks, ok := decryptedBody["tasks"].([]interface{})
	if !ok {
		t.Fatal("tasks is not an array")
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task, ok := tasks[0].(map[string]interface{})
	if !ok {
		t.Fatal("task is not a map")
	}
	if task["command"] != "shell" {
		t.Errorf("command mismatch: got %v, want shell", task["command"])
	}
}

func TestDecryptAgentMessage_InvalidBase64(t *testing.T) {
	_, _, err := DecryptAgentMessage("not valid base64!!!", testPSK)
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecryptAgentMessage_MessageTooShort(t *testing.T) {
	// Create a message that's too short (less than 100 bytes)
	shortMsg := base64.StdEncoding.EncodeToString([]byte("short"))
	_, _, err := DecryptAgentMessage(shortMsg, testPSK)
	if err == nil {
		t.Error("expected error for message too short")
	}
}

func TestDecryptAgentMessage_InvalidPSK(t *testing.T) {
	encrypted, _ := EncryptAgentResponse(testUUID, map[string]interface{}{"test": "data"}, testPSK)

	_, _, err := DecryptAgentMessage(encrypted, "not valid base64!!!")
	if err == nil {
		t.Error("expected error for invalid PSK")
	}
}

func TestDecryptAgentMessage_WrongPSK(t *testing.T) {
	encrypted, _ := EncryptAgentResponse(testUUID, map[string]interface{}{"test": "data"}, testPSK)

	// Different PSK (32 bytes of 0xFF)
	wrongPSK := base64.StdEncoding.EncodeToString([]byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	})

	_, _, err := DecryptAgentMessage(encrypted, wrongPSK)
	if err == nil {
		t.Error("expected error for wrong PSK (HMAC should fail)")
	}
}

func TestEncryptAgentResponse_InvalidPSK(t *testing.T) {
	_, err := EncryptAgentResponse(testUUID, map[string]interface{}{"test": "data"}, "not valid base64!!!")
	if err == nil {
		t.Error("expected error for invalid PSK")
	}
}

func TestEncryptAgentResponse_EmptyBody(t *testing.T) {
	encrypted, err := EncryptAgentResponse(testUUID, map[string]interface{}{}, testPSK)
	if err != nil {
		t.Fatalf("EncryptAgentResponse failed: %v", err)
	}

	uuid, body, err := DecryptAgentMessage(encrypted, testPSK)
	if err != nil {
		t.Fatalf("DecryptAgentMessage failed: %v", err)
	}

	if uuid != testUUID {
		t.Errorf("UUID mismatch: got %q, want %q", uuid, testUUID)
	}

	if len(body) != 0 {
		t.Errorf("expected empty body, got %v", body)
	}
}

// TestCompatibilityWithAgentEncryption verifies that our mock server can decrypt
// messages encrypted the same way the actual agent encrypts them.
func TestCompatibilityWithAgentEncryption(t *testing.T) {
	// Simulate what the agent does in http.go SendMessage:
	// 1. Marshal JSON
	// 2. AesEncrypt
	// 3. Prepend UUID
	// 4. Base64 encode

	body := map[string]interface{}{
		"action": "checkin",
		"ip":     "192.168.1.100",
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	key, _ := base64.StdEncoding.DecodeString(testPSK)
	encrypted := crypto.AesEncrypt(key, jsonBytes)

	// Prepend UUID (as agent does)
	message := append([]byte(testUUID), encrypted...)

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(message)

	// Now decrypt using our mock server function
	uuid, decryptedBody, err := DecryptAgentMessage(encoded, testPSK)
	if err != nil {
		t.Fatalf("DecryptAgentMessage failed: %v", err)
	}

	if uuid != testUUID {
		t.Errorf("UUID mismatch: got %q, want %q", uuid, testUUID)
	}

	if decryptedBody["action"] != "checkin" {
		t.Errorf("action mismatch: got %v, want checkin", decryptedBody["action"])
	}

	if decryptedBody["ip"] != "192.168.1.100" {
		t.Errorf("ip mismatch: got %v, want 192.168.1.100", decryptedBody["ip"])
	}
}

// TestCompatibilityWithAgentDecryption verifies that messages encrypted by our
// mock server can be decrypted the same way the actual agent decrypts them.
func TestCompatibilityWithAgentDecryption(t *testing.T) {
	body := map[string]interface{}{
		"action": "get_tasking",
		"id":     "task-123",
	}

	// Encrypt using our mock server function
	encoded, err := EncryptAgentResponse(testUUID, body, testPSK)
	if err != nil {
		t.Fatalf("EncryptAgentResponse failed: %v", err)
	}

	// Now decrypt the same way the agent does in http.go SendMessage:
	// 1. Base64 decode
	// 2. Skip first 36 bytes (UUID)
	// 3. AesDecrypt

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	if len(raw) < 36 {
		t.Fatalf("message too short: %d bytes", len(raw))
	}

	// Verify UUID
	extractedUUID := string(raw[:36])
	if extractedUUID != testUUID {
		t.Errorf("UUID mismatch: got %q, want %q", extractedUUID, testUUID)
	}

	// Decrypt encrypted portion
	key, _ := base64.StdEncoding.DecodeString(testPSK)
	decrypted := crypto.AesDecrypt(key, raw[36:])
	if len(decrypted) == 0 {
		t.Fatal("decryption failed")
	}

	var decryptedBody map[string]interface{}
	if err := json.Unmarshal(decrypted, &decryptedBody); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if decryptedBody["action"] != "get_tasking" {
		t.Errorf("action mismatch: got %v, want get_tasking", decryptedBody["action"])
	}

	if decryptedBody["id"] != "task-123" {
		t.Errorf("id mismatch: got %v, want task-123", decryptedBody["id"])
	}
}

func TestUUIDPreservation(t *testing.T) {
	// Test with various UUID formats
	uuids := []string{
		"12345678-1234-1234-1234-123456789012",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		"AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE",
	}

	for _, testUUID := range uuids {
		t.Run(testUUID, func(t *testing.T) {
			encrypted, err := EncryptAgentResponse(testUUID, map[string]interface{}{"test": "data"}, testPSK)
			if err != nil {
				t.Fatalf("EncryptAgentResponse failed: %v", err)
			}

			uuid, _, err := DecryptAgentMessage(encrypted, testPSK)
			if err != nil {
				t.Fatalf("DecryptAgentMessage failed: %v", err)
			}

			if uuid != testUUID {
				t.Errorf("UUID mismatch: got %q, want %q", uuid, testUUID)
			}
		})
	}
}
