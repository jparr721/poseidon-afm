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
