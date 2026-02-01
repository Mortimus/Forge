// Package main is the entry point for the Forge service.
// It initializes configuration, clients, and starts the main orchestration loop.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flag"
	"fmt"

	"github.com/mortimus/forge/internal/clients/jules"
	"github.com/mortimus/forge/internal/config"
	"github.com/mortimus/forge/internal/orchestrator"
	"github.com/mortimus/forge/internal/persistence"
	"github.com/mortimus/forge/internal/stats"
	"github.com/mortimus/forge/internal/version"
)

// main initializes the application and starts the orchestrator.
// It listens for SIGINT and SIGTERM to perform a graceful shutdown.
func main() {
	// Default config path
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	listSources := flag.Bool("list-sources", false, "List all available Jules sources and exit")
	flag.Parse()

	if *listSources {
		if err := runListSources(context.Background(), *configFile); err != nil {
			log.Fatalf("Failed to list sources: %v", err)
		}
		return
	}

	if err := run(context.Background(), nil, *configFile); err != nil && err != context.Canceled {
		log.Fatalf("Failed: %v", err)
	}
}

// runListSources lists all available Jules sources to stdout.
func runListSources(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	// For listing sources, we just need the Jules API Key.
	// We can iterate sources.
	client := jules.NewClient(cfg.JulesAPIKey, cfg.CheckInterval())
	sources, err := client.ListSources(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Discovered %d Jules sources:\n", len(sources))
	for _, s := range sources {
		fmt.Printf(" - Name: %s\n", s.Name)
		fmt.Printf("   ID:   %s\n", s.ID)
		fmt.Printf("   Repo: %s/%s\n", s.GithubRepo.Owner, s.GithubRepo.Repo)
	}
	return nil
}

// run is the main execution function that sets up clients and starts the orchestrator.
func run(ctx context.Context, cfg *config.Config, configPath string) error {
	if cfg == nil {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			return err
		}
	}

	log.Printf("Forge Service Starting... Version: %s", version.Version)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			cancel()
		case <-ctx.Done():
		}
	}()


	julesClient := jules.NewClient(cfg.JulesAPIKey, 1*time.Second)
	statsCollector := stats.New()
	persistenceManager := persistence.NewManager(cfg.StateFilePath)

	// Orchestrator creates its own per-repo instances of GitHub client
	orch := orchestrator.New(cfg, julesClient, statsCollector, persistenceManager)
	return orch.Run(ctx)
}
