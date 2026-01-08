# Unified Config System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace fragmented ldflags/env var configuration with a unified JSON-driven build system that generates config.go files and cross-compiles agents.

**Architecture:** A Go builder tool at `cmd/builder/` reads JSON config, generates `config.go` via text/template, copies to both module locations, and executes `go build` with appropriate GOOS/GOARCH/tags.

**Tech Stack:** Go 1.24+, text/template, os/exec for build invocation

---

## Task 1: Create Builder CLI Entry Point

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/main.go`

**Step 1: Create the builder directory and main.go**

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	configPath := flag.String("config", "", "Path to JSON config file (required)")
	output := flag.String("output", "", "Output path for binary (overrides config)")
	validate := flag.Bool("validate", false, "Validate config only, don't build")
	dryRun := flag.Bool("dry-run", false, "Show what would happen without building")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "error: --config is required")
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		cfg.Build.Output = *output
	}

	if err := ValidateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}

	if *validate {
		fmt.Println("Config is valid")
		os.Exit(0)
	}

	if *dryRun {
		PrintDryRun(cfg)
		os.Exit(0)
	}

	if err := Build(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 2: Verify file compiles**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -o /dev/null ./cmd/builder/...`
Expected: Compilation errors for undefined functions (LoadConfig, ValidateConfig, etc.) - this is expected, we'll add them next.

**Step 3: Commit**

```bash
git add cmd/builder/main.go
git commit -m "feat(builder): add CLI entry point skeleton"
```

---

## Task 2: Create Config Types

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/types.go`

**Step 1: Create types.go with all config structs**

```go
package main

// Config is the top-level configuration structure
type Config struct {
	UUID     string       `json:"uuid"`
	Debug    bool         `json:"debug"`
	Build    BuildConfig  `json:"build"`
	Profiles []string     `json:"profiles"`
	Egress   EgressConfig `json:"egress,omitempty"`
	UIClient *UIConfig    `json:"uiClient,omitempty"`

	HTTP        *HTTPConfig        `json:"http,omitempty"`
	Websocket   *WebsocketConfig   `json:"websocket,omitempty"`
	TCP         *TCPConfig         `json:"tcp,omitempty"`
	DNS         *DNSConfig         `json:"dns,omitempty"`
	DynamicHTTP *DynamicHTTPConfig `json:"dynamichttp,omitempty"`
	HTTPx       *HTTPxConfig       `json:"httpx,omitempty"`
}

type BuildConfig struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Output string `json:"output,omitempty"`
	Mode   string `json:"mode,omitempty"`
	Garble bool   `json:"garble,omitempty"`
	Static bool   `json:"static,omitempty"`
	CGO    bool   `json:"cgo,omitempty"`
}

type EgressConfig struct {
	Order           []string `json:"order,omitempty"`
	Failover        string   `json:"failover,omitempty"`
	FailedThreshold int      `json:"failedThreshold,omitempty"`
	BackoffDelay    int      `json:"backoffDelay,omitempty"`
	BackoffBase     int      `json:"backoffBase,omitempty"`
}

type UIConfig struct {
	BaseURL      string `json:"baseUrl"`
	CheckinPath  string `json:"checkinPath,omitempty"`
	PollPath     string `json:"pollPath,omitempty"`
	PollInterval int    `json:"pollInterval,omitempty"`
	HTTPTimeout  int    `json:"httpTimeout,omitempty"`
}

type HTTPConfig struct {
	CallbackHost           string            `json:"callbackHost"`
	CallbackPort           int               `json:"callbackPort"`
	AesPsk                 string            `json:"aesPsk"`
	Killdate               string            `json:"killdate"`
	Interval               int               `json:"interval"`
	Jitter                 int               `json:"jitter"`
	PostUri                string            `json:"postUri"`
	GetUri                 string            `json:"getUri"`
	QueryPathName          string            `json:"queryPathName,omitempty"`
	EncryptedExchangeCheck *bool             `json:"encryptedExchangeCheck,omitempty"`
	Headers                map[string]string `json:"headers,omitempty"`
	Proxy                  *ProxyConfig      `json:"proxy,omitempty"`
}

type ProxyConfig struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	User   string `json:"user,omitempty"`
	Pass   string `json:"pass,omitempty"`
	Bypass bool   `json:"bypass,omitempty"`
}

type WebsocketConfig struct {
	CallbackHost           string `json:"callbackHost"`
	CallbackPort           int    `json:"callbackPort"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	Interval               int    `json:"interval"`
	Jitter                 int    `json:"jitter"`
	Endpoint               string `json:"endpoint"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
	DomainFront            string `json:"domainFront,omitempty"`
	TaskingType            string `json:"taskingType,omitempty"`
	UserAgent              string `json:"userAgent,omitempty"`
}

type TCPConfig struct {
	Port                   int    `json:"port"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
}

type DNSConfig struct {
	Domains                []string `json:"domains"`
	AesPsk                 string   `json:"aesPsk"`
	Killdate               string   `json:"killdate"`
	Interval               int      `json:"interval"`
	Jitter                 int      `json:"jitter"`
	Server                 string   `json:"server,omitempty"`
	DomainRotation         string   `json:"domainRotation,omitempty"`
	FailoverThreshold      int      `json:"failoverThreshold,omitempty"`
	RecordType             string   `json:"recordType,omitempty"`
	MaxQueryLength         int      `json:"maxQueryLength,omitempty"`
	MaxSubdomainLength     int      `json:"maxSubdomainLength,omitempty"`
	EncryptedExchangeCheck *bool    `json:"encryptedExchangeCheck,omitempty"`
}

type DynamicHTTPConfig struct {
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	Interval               int    `json:"interval"`
	Jitter                 int    `json:"jitter"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
	RawC2Config            string `json:"rawC2Config"`
}

type HTTPxConfig struct {
	CallbackDomains        []string `json:"callbackDomains"`
	AesPsk                 string   `json:"aesPsk"`
	Killdate               string   `json:"killdate"`
	Interval               int      `json:"interval"`
	Jitter                 int      `json:"jitter"`
	DomainRotationMethod   string   `json:"domainRotationMethod,omitempty"`
	FailoverThreshold      int      `json:"failoverThreshold,omitempty"`
	EncryptedExchangeCheck *bool    `json:"encryptedExchangeCheck,omitempty"`
	RawC2Config            string   `json:"rawC2Config"`
}
```

**Step 2: Verify file compiles**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -o /dev/null ./cmd/builder/...`
Expected: Still fails (missing other functions), but no syntax errors in types.go

**Step 3: Commit**

```bash
git add cmd/builder/types.go
git commit -m "feat(builder): add config type definitions"
```

---

## Task 3: Create Config Loader

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/loader.go`

**Step 1: Create loader.go**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadConfig reads and parses a JSON config file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	// Build defaults
	if cfg.Build.Output == "" {
		cfg.Build.Output = "./agent"
	}
	if cfg.Build.Mode == "" {
		cfg.Build.Mode = "default"
	}

	// Egress defaults
	if len(cfg.Egress.Order) == 0 {
		cfg.Egress.Order = cfg.Profiles
	}
	if cfg.Egress.Failover == "" {
		cfg.Egress.Failover = "failover"
	}
	if cfg.Egress.FailedThreshold == 0 {
		cfg.Egress.FailedThreshold = 10
	}
	if cfg.Egress.BackoffDelay == 0 {
		cfg.Egress.BackoffDelay = 5
	}
	if cfg.Egress.BackoffBase == 0 {
		cfg.Egress.BackoffBase = 1
	}

	// UI Client defaults
	if cfg.UIClient != nil {
		if cfg.UIClient.CheckinPath == "" {
			cfg.UIClient.CheckinPath = "/checkin"
		}
		if cfg.UIClient.PollPath == "" {
			cfg.UIClient.PollPath = "/poll"
		}
		if cfg.UIClient.PollInterval == 0 {
			cfg.UIClient.PollInterval = 5
		}
		if cfg.UIClient.HTTPTimeout == 0 {
			cfg.UIClient.HTTPTimeout = 30
		}
	}

	// Profile defaults (EncryptedExchangeCheck defaults to true)
	trueVal := true
	if cfg.HTTP != nil {
		if cfg.HTTP.QueryPathName == "" {
			cfg.HTTP.QueryPathName = "q"
		}
		if cfg.HTTP.EncryptedExchangeCheck == nil {
			cfg.HTTP.EncryptedExchangeCheck = &trueVal
		}
	}
	if cfg.Websocket != nil {
		if cfg.Websocket.EncryptedExchangeCheck == nil {
			cfg.Websocket.EncryptedExchangeCheck = &trueVal
		}
		if cfg.Websocket.TaskingType == "" {
			cfg.Websocket.TaskingType = "Push"
		}
	}
	if cfg.TCP != nil {
		if cfg.TCP.EncryptedExchangeCheck == nil {
			cfg.TCP.EncryptedExchangeCheck = &trueVal
		}
	}
	if cfg.DNS != nil {
		if cfg.DNS.EncryptedExchangeCheck == nil {
			cfg.DNS.EncryptedExchangeCheck = &trueVal
		}
		if cfg.DNS.DomainRotation == "" {
			cfg.DNS.DomainRotation = "fail-over"
		}
		if cfg.DNS.FailoverThreshold == 0 {
			cfg.DNS.FailoverThreshold = 3
		}
		if cfg.DNS.RecordType == "" {
			cfg.DNS.RecordType = "TXT"
		}
	}
	if cfg.DynamicHTTP != nil {
		if cfg.DynamicHTTP.EncryptedExchangeCheck == nil {
			cfg.DynamicHTTP.EncryptedExchangeCheck = &trueVal
		}
	}
	if cfg.HTTPx != nil {
		if cfg.HTTPx.EncryptedExchangeCheck == nil {
			cfg.HTTPx.EncryptedExchangeCheck = &trueVal
		}
		if cfg.HTTPx.DomainRotationMethod == "" {
			cfg.HTTPx.DomainRotationMethod = "fail-over"
		}
		if cfg.HTTPx.FailoverThreshold == 0 {
			cfg.HTTPx.FailoverThreshold = 3
		}
	}
}
```

**Step 2: Verify file compiles**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -o /dev/null ./cmd/builder/...`
Expected: Still fails (missing ValidateConfig, Build, PrintDryRun)

**Step 3: Commit**

```bash
git add cmd/builder/loader.go
git commit -m "feat(builder): add config loader with defaults"
```

---

## Task 4: Create Config Validator

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/validate.go`

**Step 1: Create validate.go**

```go
package main

import (
	"fmt"
	"strings"
	"time"
)

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	// Required global fields
	if cfg.UUID == "" {
		return fmt.Errorf("uuid is required")
	}

	// Build validation
	if err := validateBuild(&cfg.Build); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	// Must have at least one profile
	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("profiles: at least one profile is required")
	}

	// Validate each selected profile has its config
	for _, profile := range cfg.Profiles {
		if err := validateProfile(cfg, profile); err != nil {
			return err
		}
	}

	return nil
}

func validateBuild(b *BuildConfig) error {
	validOS := map[string]bool{"windows": true, "linux": true, "darwin": true}
	if !validOS[b.OS] {
		return fmt.Errorf("os must be one of: windows, linux, darwin (got %q)", b.OS)
	}

	validArch := map[string]bool{"amd64": true, "arm64": true}
	if !validArch[b.Arch] {
		return fmt.Errorf("arch must be one of: amd64, arm64 (got %q)", b.Arch)
	}

	validMode := map[string]bool{"default": true, "c-archive": true, "c-shared": true}
	if !validMode[b.Mode] {
		return fmt.Errorf("mode must be one of: default, c-archive, c-shared (got %q)", b.Mode)
	}

	// Static only works on Linux
	if b.Static && b.OS != "linux" {
		return fmt.Errorf("static linking is only supported on linux")
	}

	return nil
}

func validateProfile(cfg *Config, profile string) error {
	switch profile {
	case "http":
		if cfg.HTTP == nil {
			return fmt.Errorf("http config is required when 'http' profile is selected")
		}
		return validateHTTP(cfg.HTTP)
	case "websocket":
		if cfg.Websocket == nil {
			return fmt.Errorf("websocket config is required when 'websocket' profile is selected")
		}
		return validateWebsocket(cfg.Websocket)
	case "tcp":
		if cfg.TCP == nil {
			return fmt.Errorf("tcp config is required when 'tcp' profile is selected")
		}
		return validateTCP(cfg.TCP)
	case "dns":
		if cfg.DNS == nil {
			return fmt.Errorf("dns config is required when 'dns' profile is selected")
		}
		return validateDNS(cfg.DNS)
	case "dynamichttp":
		if cfg.DynamicHTTP == nil {
			return fmt.Errorf("dynamichttp config is required when 'dynamichttp' profile is selected")
		}
		return validateDynamicHTTP(cfg.DynamicHTTP)
	case "httpx":
		if cfg.HTTPx == nil {
			return fmt.Errorf("httpx config is required when 'httpx' profile is selected")
		}
		return validateHTTPx(cfg.HTTPx)
	default:
		return fmt.Errorf("unknown profile: %q", profile)
	}
}

func validateHTTP(h *HTTPConfig) error {
	if h.CallbackHost == "" {
		return fmt.Errorf("http.callbackHost is required")
	}
	if h.CallbackPort == 0 {
		return fmt.Errorf("http.callbackPort is required")
	}
	if h.AesPsk == "" {
		return fmt.Errorf("http.aesPsk is required")
	}
	if err := validateKilldate(h.Killdate, "http"); err != nil {
		return err
	}
	if h.PostUri == "" {
		return fmt.Errorf("http.postUri is required")
	}
	if h.GetUri == "" {
		return fmt.Errorf("http.getUri is required")
	}
	if h.Jitter < 0 || h.Jitter > 100 {
		return fmt.Errorf("http.jitter must be between 0 and 100")
	}
	return nil
}

func validateWebsocket(w *WebsocketConfig) error {
	if w.CallbackHost == "" {
		return fmt.Errorf("websocket.callbackHost is required")
	}
	if w.CallbackPort == 0 {
		return fmt.Errorf("websocket.callbackPort is required")
	}
	if w.AesPsk == "" {
		return fmt.Errorf("websocket.aesPsk is required")
	}
	if err := validateKilldate(w.Killdate, "websocket"); err != nil {
		return err
	}
	if w.Endpoint == "" {
		return fmt.Errorf("websocket.endpoint is required")
	}
	if w.Jitter < 0 || w.Jitter > 100 {
		return fmt.Errorf("websocket.jitter must be between 0 and 100")
	}
	validTasking := map[string]bool{"Push": true, "Poll": true}
	if !validTasking[w.TaskingType] {
		return fmt.Errorf("websocket.taskingType must be Push or Poll")
	}
	return nil
}

func validateTCP(t *TCPConfig) error {
	if t.Port == 0 {
		return fmt.Errorf("tcp.port is required")
	}
	if t.AesPsk == "" {
		return fmt.Errorf("tcp.aesPsk is required")
	}
	if err := validateKilldate(t.Killdate, "tcp"); err != nil {
		return err
	}
	return nil
}

func validateDNS(d *DNSConfig) error {
	if len(d.Domains) == 0 {
		return fmt.Errorf("dns.domains is required (at least one domain)")
	}
	if d.AesPsk == "" {
		return fmt.Errorf("dns.aesPsk is required")
	}
	if err := validateKilldate(d.Killdate, "dns"); err != nil {
		return err
	}
	if d.Jitter < 0 || d.Jitter > 100 {
		return fmt.Errorf("dns.jitter must be between 0 and 100")
	}
	validRotation := map[string]bool{"fail-over": true, "round-robin": true, "random": true}
	if !validRotation[d.DomainRotation] {
		return fmt.Errorf("dns.domainRotation must be fail-over, round-robin, or random")
	}
	validRecord := map[string]bool{"A": true, "AAAA": true, "TXT": true}
	if !validRecord[d.RecordType] {
		return fmt.Errorf("dns.recordType must be A, AAAA, or TXT")
	}
	return nil
}

func validateDynamicHTTP(d *DynamicHTTPConfig) error {
	if d.AesPsk == "" {
		return fmt.Errorf("dynamichttp.aesPsk is required")
	}
	if err := validateKilldate(d.Killdate, "dynamichttp"); err != nil {
		return err
	}
	if d.RawC2Config == "" {
		return fmt.Errorf("dynamichttp.rawC2Config is required")
	}
	if d.Jitter < 0 || d.Jitter > 100 {
		return fmt.Errorf("dynamichttp.jitter must be between 0 and 100")
	}
	return nil
}

func validateHTTPx(h *HTTPxConfig) error {
	if len(h.CallbackDomains) == 0 {
		return fmt.Errorf("httpx.callbackDomains is required (at least one domain)")
	}
	if h.AesPsk == "" {
		return fmt.Errorf("httpx.aesPsk is required")
	}
	if err := validateKilldate(h.Killdate, "httpx"); err != nil {
		return err
	}
	if h.RawC2Config == "" {
		return fmt.Errorf("httpx.rawC2Config is required")
	}
	if h.Jitter < 0 || h.Jitter > 100 {
		return fmt.Errorf("httpx.jitter must be between 0 and 100")
	}
	validRotation := map[string]bool{"fail-over": true, "round-robin": true, "random": true}
	if !validRotation[h.DomainRotationMethod] {
		return fmt.Errorf("httpx.domainRotationMethod must be fail-over, round-robin, or random")
	}
	return nil
}

func validateKilldate(killdate, profile string) error {
	if killdate == "" {
		return fmt.Errorf("%s.killdate is required", profile)
	}
	_, err := time.Parse("2006-01-02", killdate)
	if err != nil {
		return fmt.Errorf("%s.killdate must be in YYYY-MM-DD format", profile)
	}
	return nil
}

// PrintDryRun shows what the build would do
func PrintDryRun(cfg *Config) {
	fmt.Println("=== DRY RUN ===")
	fmt.Printf("Target: %s/%s\n", cfg.Build.OS, cfg.Build.Arch)
	fmt.Printf("Output: %s\n", getOutputPath(cfg))
	fmt.Printf("Profiles: %s\n", strings.Join(cfg.Profiles, ", "))
	fmt.Printf("CGO: %v\n", cfg.Build.CGO)
	fmt.Printf("Garble: %v\n", cfg.Build.Garble)
	fmt.Printf("Static: %v\n", cfg.Build.Static)
	fmt.Println("\nConfig files will be written to:")
	fmt.Println("  - pkg/config/config.go")
	fmt.Println("\nBuild command:")
	fmt.Printf("  GOOS=%s GOARCH=%s go build -tags=%q -o %s .\n",
		cfg.Build.OS, cfg.Build.Arch, strings.Join(cfg.Profiles, ","), getOutputPath(cfg))
}

func getOutputPath(cfg *Config) string {
	output := cfg.Build.Output
	if cfg.Build.OS == "windows" && !strings.HasSuffix(output, ".exe") {
		output += ".exe"
	}
	return output
}
```

**Step 2: Verify file compiles**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -o /dev/null ./cmd/builder/...`
Expected: Still fails (missing Build function)

**Step 3: Commit**

```bash
git add cmd/builder/validate.go
git commit -m "feat(builder): add config validation and dry-run"
```

---

## Task 5: Create Config Template

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/templates/config.go.tmpl`

**Step 1: Create template directory and file**

```bash
mkdir -p .worktrees/unified-config/poseidon/poseidon/agent_code/cmd/builder/templates
```

Then create the template file:

```go
// Code generated by poseidon builder. DO NOT EDIT.
package config

// Global Settings
var (
	UUID  = "{{.UUID}}"
	Debug = {{.Debug}}
)

// Build Info
var (
	BuildOS   = "{{.Build.OS}}"
	BuildArch = "{{.Build.Arch}}"
)

// Egress Settings
var (
	EgressOrder       = []string{ {{- range $i, $v := .Egress.Order}}{{if $i}}, {{end}}"{{$v}}"{{end -}} }
	EgressFailover    = "{{.Egress.Failover}}"
	FailedThreshold   = {{.Egress.FailedThreshold}}
	BackoffDelay      = {{.Egress.BackoffDelay}}
	BackoffBase       = {{.Egress.BackoffBase}}
)

// UI Client Settings
var (
	UIBaseURL      = "{{if .UIClient}}{{.UIClient.BaseURL}}{{end}}"
	UICheckinPath  = "{{if .UIClient}}{{.UIClient.CheckinPath}}{{end}}"
	UIPollPath     = "{{if .UIClient}}{{.UIClient.PollPath}}{{end}}"
	UIPollInterval = {{if .UIClient}}{{.UIClient.PollInterval}}{{else}}5{{end}}
	UIHTTPTimeout  = {{if .UIClient}}{{.UIClient.HTTPTimeout}}{{else}}30{{end}}
)

// HTTP Profile
var (
	HTTPCallbackHost      = "{{if .HTTP}}{{.HTTP.CallbackHost}}{{end}}"
	HTTPCallbackPort      = {{if .HTTP}}{{.HTTP.CallbackPort}}{{else}}0{{end}}
	HTTPAesPsk            = "{{if .HTTP}}{{.HTTP.AesPsk}}{{end}}"
	HTTPKilldate          = "{{if .HTTP}}{{.HTTP.Killdate}}{{end}}"
	HTTPInterval          = {{if .HTTP}}{{.HTTP.Interval}}{{else}}0{{end}}
	HTTPJitter            = {{if .HTTP}}{{.HTTP.Jitter}}{{else}}0{{end}}
	HTTPPostUri           = "{{if .HTTP}}{{.HTTP.PostUri}}{{end}}"
	HTTPGetUri            = "{{if .HTTP}}{{.HTTP.GetUri}}{{end}}"
	HTTPQueryPathName     = "{{if .HTTP}}{{.HTTP.QueryPathName}}{{end}}"
	HTTPEncryptedExchange = {{if .HTTP}}{{if .HTTP.EncryptedExchangeCheck}}{{deref .HTTP.EncryptedExchangeCheck}}{{else}}true{{end}}{{else}}true{{end}}
	HTTPHeaders           = map[string]string{ {{- if .HTTP}}{{range $k, $v := .HTTP.Headers}}"{{$k}}": "{{$v}}", {{end}}{{end -}} }
	HTTPProxyHost         = "{{if .HTTP}}{{if .HTTP.Proxy}}{{.HTTP.Proxy.Host}}{{end}}{{end}}"
	HTTPProxyPort         = {{if .HTTP}}{{if .HTTP.Proxy}}{{.HTTP.Proxy.Port}}{{else}}0{{end}}{{else}}0{{end}}
	HTTPProxyUser         = "{{if .HTTP}}{{if .HTTP.Proxy}}{{.HTTP.Proxy.User}}{{end}}{{end}}"
	HTTPProxyPass         = "{{if .HTTP}}{{if .HTTP.Proxy}}{{.HTTP.Proxy.Pass}}{{end}}{{end}}"
	HTTPProxyBypass       = {{if .HTTP}}{{if .HTTP.Proxy}}{{.HTTP.Proxy.Bypass}}{{else}}false{{end}}{{else}}false{{end}}
)

// Websocket Profile
var (
	WebsocketCallbackHost      = "{{if .Websocket}}{{.Websocket.CallbackHost}}{{end}}"
	WebsocketCallbackPort      = {{if .Websocket}}{{.Websocket.CallbackPort}}{{else}}0{{end}}
	WebsocketAesPsk            = "{{if .Websocket}}{{.Websocket.AesPsk}}{{end}}"
	WebsocketKilldate          = "{{if .Websocket}}{{.Websocket.Killdate}}{{end}}"
	WebsocketInterval          = {{if .Websocket}}{{.Websocket.Interval}}{{else}}0{{end}}
	WebsocketJitter            = {{if .Websocket}}{{.Websocket.Jitter}}{{else}}0{{end}}
	WebsocketEndpoint          = "{{if .Websocket}}{{.Websocket.Endpoint}}{{end}}"
	WebsocketEncryptedExchange = {{if .Websocket}}{{if .Websocket.EncryptedExchangeCheck}}{{deref .Websocket.EncryptedExchangeCheck}}{{else}}true{{end}}{{else}}true{{end}}
	WebsocketDomainFront       = "{{if .Websocket}}{{.Websocket.DomainFront}}{{end}}"
	WebsocketTaskingType       = "{{if .Websocket}}{{.Websocket.TaskingType}}{{end}}"
	WebsocketUserAgent         = "{{if .Websocket}}{{.Websocket.UserAgent}}{{end}}"
)

// TCP Profile
var (
	TCPPort              = {{if .TCP}}{{.TCP.Port}}{{else}}0{{end}}
	TCPAesPsk            = "{{if .TCP}}{{.TCP.AesPsk}}{{end}}"
	TCPKilldate          = "{{if .TCP}}{{.TCP.Killdate}}{{end}}"
	TCPEncryptedExchange = {{if .TCP}}{{if .TCP.EncryptedExchangeCheck}}{{deref .TCP.EncryptedExchangeCheck}}{{else}}true{{end}}{{else}}true{{end}}
)

// DNS Profile
var (
	DNSDomains            = []string{ {{- if .DNS}}{{range $i, $v := .DNS.Domains}}{{if $i}}, {{end}}"{{$v}}"{{end}}{{end -}} }
	DNSAesPsk             = "{{if .DNS}}{{.DNS.AesPsk}}{{end}}"
	DNSKilldate           = "{{if .DNS}}{{.DNS.Killdate}}{{end}}"
	DNSInterval           = {{if .DNS}}{{.DNS.Interval}}{{else}}0{{end}}
	DNSJitter             = {{if .DNS}}{{.DNS.Jitter}}{{else}}0{{end}}
	DNSServer             = "{{if .DNS}}{{.DNS.Server}}{{end}}"
	DNSDomainRotation     = "{{if .DNS}}{{.DNS.DomainRotation}}{{end}}"
	DNSFailoverThreshold  = {{if .DNS}}{{.DNS.FailoverThreshold}}{{else}}3{{end}}
	DNSRecordType         = "{{if .DNS}}{{.DNS.RecordType}}{{end}}"
	DNSMaxQueryLength     = {{if .DNS}}{{.DNS.MaxQueryLength}}{{else}}253{{end}}
	DNSMaxSubdomainLength = {{if .DNS}}{{.DNS.MaxSubdomainLength}}{{else}}63{{end}}
	DNSEncryptedExchange  = {{if .DNS}}{{if .DNS.EncryptedExchangeCheck}}{{deref .DNS.EncryptedExchangeCheck}}{{else}}true{{end}}{{else}}true{{end}}
)

// DynamicHTTP Profile
var (
	DynamicHTTPAesPsk             = "{{if .DynamicHTTP}}{{.DynamicHTTP.AesPsk}}{{end}}"
	DynamicHTTPKilldate           = "{{if .DynamicHTTP}}{{.DynamicHTTP.Killdate}}{{end}}"
	DynamicHTTPInterval           = {{if .DynamicHTTP}}{{.DynamicHTTP.Interval}}{{else}}0{{end}}
	DynamicHTTPJitter             = {{if .DynamicHTTP}}{{.DynamicHTTP.Jitter}}{{else}}0{{end}}
	DynamicHTTPEncryptedExchange  = {{if .DynamicHTTP}}{{if .DynamicHTTP.EncryptedExchangeCheck}}{{deref .DynamicHTTP.EncryptedExchangeCheck}}{{else}}true{{end}}{{else}}true{{end}}
	DynamicHTTPRawC2Config        = `{{if .DynamicHTTP}}{{.DynamicHTTP.RawC2Config}}{{end}}`
)

// HTTPx Profile
var (
	HTTPxCallbackDomains       = []string{ {{- if .HTTPx}}{{range $i, $v := .HTTPx.CallbackDomains}}{{if $i}}, {{end}}"{{$v}}"{{end}}{{end -}} }
	HTTPxAesPsk                = "{{if .HTTPx}}{{.HTTPx.AesPsk}}{{end}}"
	HTTPxKilldate              = "{{if .HTTPx}}{{.HTTPx.Killdate}}{{end}}"
	HTTPxInterval              = {{if .HTTPx}}{{.HTTPx.Interval}}{{else}}0{{end}}
	HTTPxJitter                = {{if .HTTPx}}{{.HTTPx.Jitter}}{{else}}0{{end}}
	HTTPxDomainRotationMethod  = "{{if .HTTPx}}{{.HTTPx.DomainRotationMethod}}{{end}}"
	HTTPxFailoverThreshold     = {{if .HTTPx}}{{.HTTPx.FailoverThreshold}}{{else}}3{{end}}
	HTTPxEncryptedExchange     = {{if .HTTPx}}{{if .HTTPx.EncryptedExchangeCheck}}{{deref .HTTPx.EncryptedExchangeCheck}}{{else}}true{{end}}{{else}}true{{end}}
	HTTPxRawC2Config           = `{{if .HTTPx}}{{.HTTPx.RawC2Config}}{{end}}`
)
```

**Step 2: Commit**

```bash
git add cmd/builder/templates/config.go.tmpl
git commit -m "feat(builder): add config.go template"
```

---

## Task 6: Create Generator

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/generate.go`

**Step 1: Create generate.go**

```go
package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// GenerateConfig generates the config.go file from the template
func GenerateConfig(cfg *Config, outputPath string) error {
	// Create template with helper functions
	funcMap := template.FuncMap{
		"deref": func(b *bool) bool {
			if b == nil {
				return true
			}
			return *b
		},
	}

	tmpl, err := template.New("config.go.tmpl").Funcs(funcMap).ParseFS(templateFS, "templates/config.go.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
```

**Step 2: Verify file compiles**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -o /dev/null ./cmd/builder/...`
Expected: Still fails (missing Build function)

**Step 3: Commit**

```bash
git add cmd/builder/generate.go
git commit -m "feat(builder): add config generator with embedded template"
```

---

## Task 7: Create Build Executor

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/build.go`

**Step 1: Create build.go**

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Build generates config and compiles the agent
func Build(cfg *Config) error {
	// Get the agent_code directory (where we run go build)
	builderDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// The builder lives in cmd/builder, so agent_code is two levels up
	agentCodeDir := filepath.Dir(filepath.Dir(builderDir))

	// If we're already in agent_code, adjust
	if filepath.Base(builderDir) == "agent_code" {
		agentCodeDir = builderDir
	}

	// Generate config.go to pkg/config/
	configPath := filepath.Join(agentCodeDir, "pkg", "config", "config.go")
	fmt.Printf("Generating config: %s\n", configPath)
	if err := GenerateConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Build the agent
	output := getOutputPath(cfg)
	if !filepath.IsAbs(output) {
		output = filepath.Join(agentCodeDir, output)
	}

	tags := strings.Join(cfg.Profiles, ",")

	// Prepare build command
	args := []string{"build"}

	// Add build mode flags
	switch cfg.Build.Mode {
	case "c-archive":
		args = append(args, "-buildmode=c-archive")
	case "c-shared":
		args = append(args, "-buildmode=c-shared")
	}

	// Add tags
	args = append(args, fmt.Sprintf("-tags=%s", tags))

	// Add output
	args = append(args, "-o", output)

	// Add ldflags for static linking
	if cfg.Build.Static {
		args = append(args, `-ldflags=-extldflags "-static"`)
	}

	// Add source directory
	args = append(args, ".")

	// Set environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOOS=%s", cfg.Build.OS))
	env = append(env, fmt.Sprintf("GOARCH=%s", cfg.Build.Arch))
	if cfg.Build.CGO {
		env = append(env, "CGO_ENABLED=1")
	} else {
		env = append(env, "CGO_ENABLED=0")
	}

	// Choose build command
	buildCmd := "go"
	if cfg.Build.Garble {
		buildCmd = "garble"
		args = append([]string{"-literals", "-tiny", "-seed=random"}, args...)
	}

	fmt.Printf("Running: %s %s\n", buildCmd, strings.Join(args, " "))
	fmt.Printf("GOOS=%s GOARCH=%s CGO_ENABLED=%v\n", cfg.Build.OS, cfg.Build.Arch, cfg.Build.CGO)

	cmd := exec.Command(buildCmd, args...)
	cmd.Dir = agentCodeDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("\nBuild successful: %s\n", output)
	return nil
}
```

**Step 2: Verify builder compiles**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -o /dev/null ./cmd/builder/...`
Expected: SUCCESS - no errors

**Step 3: Commit**

```bash
git add cmd/builder/build.go
git commit -m "feat(builder): add build executor with cross-compilation support"
```

---

## Task 8: Create Config Package for Agent

**Files:**
- Create: `poseidon/poseidon/agent_code/pkg/config/config.example.go`

**Step 1: Create pkg/config directory and example file**

```go
//go:build ignore

// This file shows the structure of the generated config.go
// The actual config.go is generated by the builder tool and should not be committed.
// Copy this file to config.go and modify for local testing.

package config

// Global Settings
var (
	UUID  = "00000000-0000-0000-0000-000000000000"
	Debug = true
)

// Build Info
var (
	BuildOS   = "linux"
	BuildArch = "amd64"
)

// Egress Settings
var (
	EgressOrder       = []string{"http"}
	EgressFailover    = "failover"
	FailedThreshold   = 10
	BackoffDelay      = 5
	BackoffBase       = 1
)

// UI Client Settings
var (
	UIBaseURL      = "http://localhost:11111"
	UICheckinPath  = "/checkin"
	UIPollPath     = "/poll"
	UIPollInterval = 5
	UIHTTPTimeout  = 30
)

// HTTP Profile
var (
	HTTPCallbackHost      = "https://localhost:443"
	HTTPCallbackPort      = 443
	HTTPAesPsk            = ""
	HTTPKilldate          = "2099-12-31"
	HTTPInterval          = 10
	HTTPJitter            = 20
	HTTPPostUri           = "/data"
	HTTPGetUri            = "/news"
	HTTPQueryPathName     = "q"
	HTTPEncryptedExchange = true
	HTTPHeaders           = map[string]string{}
	HTTPProxyHost         = ""
	HTTPProxyPort         = 0
	HTTPProxyUser         = ""
	HTTPProxyPass         = ""
	HTTPProxyBypass       = false
)

// Websocket Profile
var (
	WebsocketCallbackHost      = ""
	WebsocketCallbackPort      = 0
	WebsocketAesPsk            = ""
	WebsocketKilldate          = ""
	WebsocketInterval          = 0
	WebsocketJitter            = 0
	WebsocketEndpoint          = ""
	WebsocketEncryptedExchange = true
	WebsocketDomainFront       = ""
	WebsocketTaskingType       = "Push"
	WebsocketUserAgent         = ""
)

// TCP Profile
var (
	TCPPort              = 0
	TCPAesPsk            = ""
	TCPKilldate          = ""
	TCPEncryptedExchange = true
)

// DNS Profile
var (
	DNSDomains            = []string{}
	DNSAesPsk             = ""
	DNSKilldate           = ""
	DNSInterval           = 0
	DNSJitter             = 0
	DNSServer             = ""
	DNSDomainRotation     = "fail-over"
	DNSFailoverThreshold  = 3
	DNSRecordType         = "TXT"
	DNSMaxQueryLength     = 253
	DNSMaxSubdomainLength = 63
	DNSEncryptedExchange  = true
)

// DynamicHTTP Profile
var (
	DynamicHTTPAesPsk            = ""
	DynamicHTTPKilldate          = ""
	DynamicHTTPInterval          = 0
	DynamicHTTPJitter            = 0
	DynamicHTTPEncryptedExchange = true
	DynamicHTTPRawC2Config       = ``
)

// HTTPx Profile
var (
	HTTPxCallbackDomains      = []string{}
	HTTPxAesPsk               = ""
	HTTPxKilldate             = ""
	HTTPxInterval             = 0
	HTTPxJitter               = 0
	HTTPxDomainRotationMethod = "fail-over"
	HTTPxFailoverThreshold    = 3
	HTTPxEncryptedExchange    = true
	HTTPxRawC2Config          = ``
)
```

**Step 2: Update .gitignore to exclude generated config.go**

Add to `poseidon/poseidon/agent_code/.gitignore` (create if needed):

```
# Generated config
pkg/config/config.go
```

**Step 3: Commit**

```bash
git add pkg/config/config.example.go
git add .gitignore  # or create poseidon/poseidon/agent_code/.gitignore
git commit -m "feat(config): add config package with example file"
```

---

## Task 9: Create Test Config JSON

**Files:**
- Create: `poseidon/poseidon/agent_code/cmd/builder/testdata/http-only.json`

**Step 1: Create testdata directory and test config**

```bash
mkdir -p cmd/builder/testdata
```

```json
{
  "uuid": "test-uuid-1234",
  "debug": true,
  "build": {
    "os": "linux",
    "arch": "amd64",
    "output": "./test-agent"
  },
  "profiles": ["http"],
  "egress": {
    "order": ["http"],
    "failover": "failover",
    "failedThreshold": 10
  },
  "http": {
    "callbackHost": "https://test.example.com",
    "callbackPort": 443,
    "aesPsk": "dGVzdC1rZXktYmFzZTY0",
    "killdate": "2099-12-31",
    "interval": 10,
    "jitter": 20,
    "postUri": "/api/data",
    "getUri": "/api/status",
    "queryPathName": "q",
    "encryptedExchangeCheck": true,
    "headers": {
      "User-Agent": "Mozilla/5.0 Test Agent"
    }
  }
}
```

**Step 2: Verify builder works with test config**

Run:
```bash
cd .worktrees/unified-config/poseidon/poseidon/agent_code
go run ./cmd/builder --config cmd/builder/testdata/http-only.json --dry-run
```

Expected: Dry run output showing build parameters

**Step 3: Commit**

```bash
git add cmd/builder/testdata/http-only.json
git commit -m "test(builder): add test config for http-only build"
```

---

## Task 10: Update profile.go to Use Config Package

**Files:**
- Modify: `poseidon/poseidon/agent_code/pkg/profiles/profile.go`

**Step 1: Update imports and remove ldflags variables**

Replace the package-level variables section (lines 23-35) with config imports:

```go
package profiles

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/config"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/responses"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/functions"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

// UUID is now read from config package
var UUID = config.UUID

// these are internal representations and other variables
var (
	// currentConnectionID is which fallback profile we're currently running
	currentConnectionID = 0
	// failedConnectionCounts mapping of C2 profile to failed egress connection counts
	failedConnectionCounts map[string]int
	// failedConnectionCountThreshold is how many failed attempts before rotating c2 profiles
	failedConnectionCountThreshold = config.FailedThreshold
	// egressOrder the priority for starting and running egress profiles
	egressOrder = config.EgressOrder
	// egress_failover is read from config
	egress_failover = config.EgressFailover
	// backoff settings
	backoffDelay   = config.BackoffDelay
	backoffSeconds = config.BackoffBase
	// MythicID is the callback UUID once this payload finishes staging
	MythicID = ""

	// availableC2Profiles map of C2 profile name to instance of that profile
	availableC2Profiles = make(map[string]structs.Profile)
)
```

**Step 2: Update Initialize() function**

Remove the base64 decoding since we now read directly from config:

```go
// Initialize parses the connection order information and threshold counts from config
func Initialize() {
	// egressOrder is already set from config package
	failedConnectionCounts = make(map[string]int)
	for _, key := range egressOrder {
		failedConnectionCounts[key] = 0
	}
	utils.PrintDebug(fmt.Sprintf("Initial Egress order: %v", egressOrder))
}
```

**Step 3: Verify build still works**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -tags="http" -o /dev/null .`
Expected: Build fails because pkg/config/config.go doesn't exist yet - this is expected

**Step 4: Create a minimal config.go for testing**

Copy config.example.go to config.go temporarily:

```bash
cp pkg/config/config.example.go pkg/config/config.go
# Remove the //go:build ignore line from config.go
```

**Step 5: Verify build works**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -tags="http" -o /dev/null .`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add pkg/profiles/profile.go pkg/config/config.go
git commit -m "refactor(profiles): use config package for global settings"
```

---

## Task 11: Update http.go to Use Config Package

**Files:**
- Modify: `poseidon/poseidon/agent_code/pkg/profiles/http.go`

**Step 1: Remove the ldflags variable and base64 decoding**

At line 31, remove:
```go
var http_initial_config string
```

**Step 2: Update init() function to read from config**

Replace the existing init() function (lines 184-244) with:

```go
func init() {
	// Read directly from config package instead of decoding base64
	killDateString := fmt.Sprintf("%sT00:00:00.000Z", config.HTTPKilldate)
	killDateTime, err := time.Parse("2006-01-02T15:04:05.000Z", killDateString)
	if err != nil {
		utils.PrintDebug(fmt.Sprintf("error parsing killdate, using far future: %v\n", err))
		killDateTime = time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	}

	profile := C2HTTP{
		BaseURL:               parseURLAndPort(config.HTTPCallbackHost, uint(config.HTTPCallbackPort)),
		PostURI:               config.HTTPPostUri,
		ProxyUser:             config.HTTPProxyUser,
		ProxyPass:             config.HTTPProxyPass,
		Key:                   config.HTTPAesPsk,
		Killdate:              killDateTime,
		ShouldStop:            true,
		stoppedChannel:        make(chan bool, 1),
		interruptSleepChannel: make(chan bool, 1),
	}

	profile.Interval = config.HTTPInterval
	if profile.Interval < 0 {
		profile.Interval = 0
	}

	profile.Jitter = config.HTTPJitter
	if profile.Jitter < 0 {
		profile.Jitter = 0
	}

	profile.HeaderList = config.HTTPHeaders

	if config.HTTPProxyHost != "" && len(config.HTTPProxyHost) > 3 {
		profile.ProxyURL = parseURLAndPort(config.HTTPProxyHost, uint(config.HTTPProxyPort))
		if config.HTTPProxyUser != "" && config.HTTPProxyPass != "" {
			profile.ProxyUser = config.HTTPProxyUser
			profile.ProxyPass = config.HTTPProxyPass
		}
	}

	profile.ProxyBypass = config.HTTPProxyBypass
	profile.ExchangingKeys = config.HTTPEncryptedExchange

	RegisterAvailableC2Profile(&profile)
}
```

**Step 3: Add config import**

Add to imports:
```go
"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/config"
```

**Step 4: Remove HTTPInitialConfig struct and UnmarshalJSON method**

Remove lines 33-116 (the HTTPInitialConfig struct and its methods) - they're no longer needed.

**Step 5: Verify build works**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -tags="http" -o /dev/null .`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add pkg/profiles/http.go
git commit -m "refactor(http): use config package instead of ldflags"
```

---

## Task 12: Update websocket.go to Use Config Package

**Files:**
- Modify: `poseidon/poseidon/agent_code/pkg/profiles/websocket.go`

**Step 1: Read websocket.go to understand current structure**

Look for:
- `var websocket_initial_config string`
- The `init()` function
- Import statements

**Step 2: Remove ldflags variable and update init()**

Similar pattern to http.go - remove base64 decoding and read from config:

```go
func init() {
	killDateString := fmt.Sprintf("%sT00:00:00.000Z", config.WebsocketKilldate)
	killDateTime, err := time.Parse("2006-01-02T15:04:05.000Z", killDateString)
	if err != nil {
		killDateTime = time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	}

	profile := C2Websocket{
		// ... read from config.Websocket* variables
	}
	RegisterAvailableC2Profile(&profile)
}
```

**Step 3: Add config import**

**Step 4: Remove WebsocketInitialConfig struct**

**Step 5: Verify build**

Run: `cd .worktrees/unified-config/poseidon/poseidon/agent_code && go build -tags="websocket" -o /dev/null .`

**Step 6: Commit**

```bash
git add pkg/profiles/websocket.go
git commit -m "refactor(websocket): use config package instead of ldflags"
```

---

## Task 13: Update tcp.go to Use Config Package

Same pattern as Task 11-12. Read from `config.TCP*` variables.

---

## Task 14: Update dns.go to Use Config Package

Same pattern. Read from `config.DNS*` variables.

---

## Task 15: Update dynamichttp.go to Use Config Package

Same pattern. Read from `config.DynamicHTTP*` variables.

---

## Task 16: Update httpx.go to Use Config Package

Same pattern. Read from `config.HTTPx*` variables.

---

## Task 17: Update utils.go to Use Config Package

**Files:**
- Modify: `poseidon/poseidon/agent_code/pkg/utils/utils.go`

**Step 1: Remove debugString and update init()**

```go
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
// ... rest unchanged
```

**Step 2: Verify build**

**Step 3: Commit**

```bash
git add pkg/utils/utils.go
git commit -m "refactor(utils): use config.Debug instead of ldflags"
```

---

## Task 18: Update main.go to Use Config Package

**Files:**
- Modify: `poseidon/main.go`

**Step 1: Update to read from config**

```go
package main

import (
	"context"
	"log"
	"net/http"
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

	tasks.Initialize()
	responses.Initialize(func() chan structs.MythicMessage { return nil })
	files.Initialize()
	p2p.Initialize()

	client := buildClient()

	if err := client.Run(ctx); err != nil {
		log.Fatalf("uiclient exited with error: %v", err)
	}
}

func buildClient() *pollclient.Client {
	httpClient := &http.Client{Timeout: time.Duration(config.UIHTTPTimeout) * time.Second}

	return pollclient.New(pollclient.Config{
		BaseURL:     config.UIBaseURL,
		CheckinPath: config.UICheckinPath,
		PollPath:    config.UIPollPath,
		Interval:    time.Duration(config.UIPollInterval) * time.Second,
		HTTPClient:  httpClient,
	})
}
```

**Step 2: Commit**

```bash
git add poseidon/main.go
git commit -m "refactor(main): use config package instead of env vars"
```

---

## Task 19: Clean Up Makefile

**Files:**
- Modify: `poseidon/poseidon/agent_code/Makefile`

**Step 1: Simplify Makefile**

Remove all ldflags injection. The new build process is:
1. Generate config using builder
2. Run `go build -tags="profile"`

Update Makefile to just have simple targets:

```makefile
BINARY_NAME=poseidon

# Build targets - use the builder tool for full builds
# These are for quick local testing only

build_http:
	go build -o ${BINARY_NAME}_http -tags="http" .

build_websocket:
	go build -o ${BINARY_NAME}_websocket -tags="websocket" .

build_tcp:
	go build -o ${BINARY_NAME}_tcp -tags="tcp" .

build_dns:
	go build -o ${BINARY_NAME}_dns -tags="dns" .

# Full build using builder tool
build:
	@echo "Use: go run ./cmd/builder --config <config.json>"
	@echo "Example: go run ./cmd/builder --config cmd/builder/testdata/http-only.json"

clean:
	rm -f ${BINARY_NAME}_*
	rm -f pkg/config/config.go
```

**Step 2: Commit**

```bash
git add Makefile
git commit -m "refactor(makefile): simplify after config unification"
```

---

## Task 20: End-to-End Test

**Step 1: Run full build with builder**

```bash
cd .worktrees/unified-config/poseidon/poseidon/agent_code
go run ./cmd/builder --config cmd/builder/testdata/http-only.json
```

Expected: Binary created at ./test-agent

**Step 2: Verify binary exists**

```bash
ls -la test-agent
```

**Step 3: Test cross-compilation**

Create a Windows test:

```bash
# Create windows config
cat > cmd/builder/testdata/http-windows.json << 'EOF'
{
  "uuid": "test-windows-uuid",
  "debug": false,
  "build": {
    "os": "windows",
    "arch": "amd64",
    "output": "./test-agent-win"
  },
  "profiles": ["http"],
  "http": {
    "callbackHost": "https://test.example.com",
    "callbackPort": 443,
    "aesPsk": "dGVzdC1rZXktYmFzZTY0",
    "killdate": "2099-12-31",
    "interval": 10,
    "jitter": 20,
    "postUri": "/api/data",
    "getUri": "/api/status"
  }
}
EOF

go run ./cmd/builder --config cmd/builder/testdata/http-windows.json
```

Expected: test-agent-win.exe created

**Step 4: Commit test configs**

```bash
git add cmd/builder/testdata/
git commit -m "test(builder): add cross-compilation test configs"
```

---

## Task 21: Final Cleanup and Documentation

**Step 1: Update .gitignore at project root**

Add entries for generated config files and test binaries.

**Step 2: Update CLAUDE.md with new build instructions**

Add section about using the builder tool.

**Step 3: Final commit**

```bash
git add -A
git commit -m "docs: update build instructions for unified config system"
```

---

## Summary

After completing all tasks, you will have:

1. **Builder tool** at `cmd/builder/` that:
   - Parses JSON config
   - Validates all fields
   - Generates `config.go` from template
   - Cross-compiles agent binaries

2. **Config package** at `pkg/config/` that:
   - Provides all config values as package variables
   - Is generated at build time (not committed)
   - Has an example file for reference

3. **Migrated profiles** that:
   - Import config package directly
   - No longer use ldflags/base64 injection
   - Work identically to before

4. **Simplified Makefile** that:
   - Removed all ldflags complexity
   - Points users to builder tool
