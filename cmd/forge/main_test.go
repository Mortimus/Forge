package main

import (
	"context"
	"testing"
)

func TestRun_Config(t *testing.T) {
	// Should fail likely due to missing config file if we pass a path that doesn't exist
    // run() now takes a path string.
    
    err := run(context.Background(), nil, "non_existent.yaml")
    if err == nil {
        t.Error("expected error for missing config")
    }
}

// TestRun using a valid config is hard without mocking the entire world inside Run()
// We'll trust unit tests for individual components.
