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
