// Package prompts manages the loading of prompt templates.
// It supports both embedded templates and external overrides.
package prompts

import (
	"embed"
	"fmt"
	"os"
)

//go:embed templates/*
var templatesFS embed.FS

// GetTemplate reads a template file. It prefers the externalPath if provided.
// If externalPath is empty, it falls back to the embedded filesystem using the name.
func GetTemplate(name string, externalPath string) (string, error) {
	// 1. Check External Override
	if externalPath != "" {
		content, err := os.ReadFile(externalPath)
		if err != nil {
			return "", fmt.Errorf("failed to read external template %s: %w", externalPath, err)
		}
		return string(content), nil
	}

	// 2. Fallback to Embedded
	content, err := templatesFS.ReadFile("templates/" + name)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
