package utils

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/config"
)

var (
	// debug is read from config
	debug = config.Debug
	// SeededRand is used when generating a random value for EKE
	SeededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func init() {
	if debug {
		fmt.Println("Debug mode enabled")
	}
}

func PrintDebug(msg string) {
	if debug {
		log.Print(msg)
	}
}

func GenerateSessionID() string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 20)
	for i := range b {
		b[i] = letterBytes[SeededRand.Intn(len(letterBytes))]
	}
	return string(b)
}

func RandomNumInRange(limit int) int {
	return SeededRand.Intn(limit)
}
