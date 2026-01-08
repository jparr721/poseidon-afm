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
