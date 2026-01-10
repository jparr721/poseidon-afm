// Package main provides a standalone mock AFM server for manual testing.
//
// Usage:
//
//	go run ./cmd/mockafm [flags]
//	  -port       Port to listen on (default: 11111)
//	  -psk        Pre-shared key, base64-encoded (default: generates random)
//	  -operation  Operation ID for URL path (default: "test-operation")
//
// The server prints connection info on startup and runs until interrupted.
package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/testing/mockafm"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 11111, "Port to listen on")
	psk := flag.String("psk", "", "Pre-shared key (base64-encoded 32 bytes, generates random if empty)")
	operationID := flag.String("operation", "test-operation", "Operation ID for URL path")
	flag.Parse()

	// Generate PSK if not provided
	actualPSK := *psk
	if actualPSK == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating PSK: %v\n", err)
			os.Exit(1)
		}
		actualPSK = base64.StdEncoding.EncodeToString(key)
	}

	// Create server config
	config := mockafm.ServerConfig{
		PSK:         actualPSK,
		OperationID: *operationID,
	}

	// Create and start server
	server := mockafm.NewServer(config)
	if err := server.Start(*port); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	// Print connection info
	fmt.Println("Mock AFM Server Started")
	fmt.Println("=======================")
	fmt.Printf("Address:     %s\n", server.GetAddr())
	fmt.Printf("URL:         %s\n", server.GetURL())
	fmt.Printf("Operation:   %s\n", *operationID)
	fmt.Printf("PSK:         %s\n", actualPSK)
	fmt.Println()
	fmt.Println("Agent config values:")
	fmt.Printf("  callbackHost: http://127.0.0.1\n")
	fmt.Printf("  callbackPort: %d\n", *port)
	fmt.Printf("  postUri:      /api/v1/operations/%s/agent\n", *operationID)
	fmt.Printf("  getUri:       /api/v1/operations/%s/agent\n", *operationID)
	fmt.Printf("  aesPsk:       %s\n", actualPSK)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	if err := server.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping server: %v\n", err)
	}
	fmt.Println("Server stopped.")
}
