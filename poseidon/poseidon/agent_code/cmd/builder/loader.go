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
