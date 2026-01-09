// Package mockafm provides a mock AFM server for integration testing.
package mockafm

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/crypto"
)

const (
	// UUIDLength is the length of the agent UUID (36 bytes for standard UUID format)
	UUIDLength = 36
)

var (
	// ErrInvalidMessageLength indicates the encrypted message is too short
	ErrInvalidMessageLength = errors.New("encrypted message too short")

	// ErrBase64Decode indicates base64 decoding failed
	ErrBase64Decode = errors.New("failed to decode base64 message")

	// ErrDecryption indicates decryption failed
	ErrDecryption = errors.New("decryption failed")

	// ErrInvalidPSK indicates the PSK format is invalid
	ErrInvalidPSK = errors.New("invalid PSK format")

	// ErrJSONUnmarshal indicates JSON parsing failed
	ErrJSONUnmarshal = errors.New("failed to unmarshal JSON body")

	// ErrJSONMarshal indicates JSON serialization failed
	ErrJSONMarshal = errors.New("failed to marshal JSON body")
)

// DecryptAgentMessage decrypts an incoming message from a Poseidon agent.
//
// Message format (after base64 decode):
//
//	UUID[36 bytes] + IV[16 bytes] + AES-256-CBC(JSON) + HMAC-SHA256[32 bytes]
//
// Parameters:
//   - encryptedBody: base64-encoded encrypted message from agent
//   - psk: base64-encoded 32-byte pre-shared key
//
// Returns:
//   - uuid: the 36-byte agent UUID
//   - body: the decrypted JSON payload as a map
//   - err: any error that occurred
func DecryptAgentMessage(encryptedBody string, psk string) (uuid string, body map[string]interface{}, err error) {
	// Decode the PSK from base64
	key, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrInvalidPSK, err)
	}

	// Decode the message from base64
	raw, err := base64.StdEncoding.DecodeString(encryptedBody)
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrBase64Decode, err)
	}

	// Message must be at least UUID (36) + IV (16) + min ciphertext (16) + HMAC (32) = 100 bytes
	if len(raw) < UUIDLength+16+16+32 {
		return "", nil, fmt.Errorf("%w: got %d bytes, need at least 100", ErrInvalidMessageLength, len(raw))
	}

	// Extract UUID (first 36 bytes)
	uuid = string(raw[:UUIDLength])

	// Extract encrypted portion (IV + ciphertext + HMAC)
	encryptedPortion := raw[UUIDLength:]

	// Decrypt using the existing crypto package
	decrypted := crypto.AesDecrypt(key, encryptedPortion)
	if len(decrypted) == 0 {
		return "", nil, ErrDecryption
	}

	// Parse JSON body
	body = make(map[string]interface{})
	if err := json.Unmarshal(decrypted, &body); err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrJSONUnmarshal, err)
	}

	return uuid, body, nil
}

// EncryptAgentResponse encrypts a response to send to a Poseidon agent.
//
// Message format (before base64 encode):
//
//	UUID[36 bytes] + IV[16 bytes] + AES-256-CBC(JSON) + HMAC-SHA256[32 bytes]
//
// Parameters:
//   - uuid: the 36-byte agent UUID to prepend
//   - body: the response body (will be JSON-serialized)
//   - psk: base64-encoded 32-byte pre-shared key
//
// Returns:
//   - encrypted: base64-encoded encrypted message
//   - err: any error that occurred
func EncryptAgentResponse(uuid string, body interface{}, psk string) (string, error) {
	// Decode the PSK from base64
	key, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPSK, err)
	}

	// Serialize body to JSON
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrJSONMarshal, err)
	}

	// Encrypt using the existing crypto package (returns IV + ciphertext + HMAC)
	encrypted := crypto.AesEncrypt(key, jsonBytes)
	if len(encrypted) == 0 {
		return "", ErrDecryption
	}

	// Prepend UUID
	message := append([]byte(uuid), encrypted...)

	// Base64 encode the final message
	return base64.StdEncoding.EncodeToString(message), nil
}
