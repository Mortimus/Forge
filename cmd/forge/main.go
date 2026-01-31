// Package main is the entry point for the Forge service.
// It initializes configuration, clients, and starts the main orchestration loop.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mortimus/forge/internal/clients/github"
	"github.com/mortimus/forge/internal/clients/jules"
	"github.com/mortimus/forge/internal/config"
	"github.com/mortimus/forge/internal/orchestrator"
	"github.com/mortimus/forge/internal/stats"
	"github.com/mortimus/forge/internal/version"
)

// main initializes the application and starts the orchestrator.
// It listens for SIGINT and SIGTERM to perform a graceful shutdown.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Forge Service Starting... Version: %s", version.Version)
	log.Printf("Target Repo: %s", cfg.GithubRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// 1. Clients
	ghClient, err := github.NewClient(ctx, cfg.GithubPAT, cfg.GithubRepo)
	if err != nil {
		log.Fatalf("Failed to create GitHub Client: %v", err)
	}

	julesClient := jules.NewClient(cfg.JulesAPIKey)
	statsCollector := stats.New()

	// 2. Orchestrator
	orch := orchestrator.New(cfg, ghClient, julesClient, statsCollector)

	// 3. Run
	if err := orch.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Orchestrator failed: %v", err)
	}

	log.Println("Forge Service Stopped")
}
