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
