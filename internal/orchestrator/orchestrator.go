// Package orchestrator implements the core logic of the Ralph service.
// It manages the OODA loop: observing GitHub state, deciding on actions, and acting via Jules.
package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/mortimus/ralph/internal/clients/github"
	"github.com/mortimus/ralph/internal/clients/jules"
	"github.com/mortimus/ralph/internal/config"
	"github.com/mortimus/ralph/internal/prompts"
	"github.com/mortimus/ralph/internal/stats"
)

// Orchestrator manages the lifecycle of automated coding sessions.
// It coordinates between GitHub (state/code) and Jules (intelligence/action).
type Orchestrator struct {
	cfg   *config.Config
	gh    *github.Client
	jules *jules.Client
	stats *stats.Collector

	sourceName string // Verified Jules Source Name

	// State
	mu             sync.Mutex
	activeSessions map[string]*ActiveSession // Key: SessionID (Jules Resource Name)
	dailyCount     int
	lastReset      time.Time
}

// ActiveSession represents a currently running Jules session managed by this instance.
type ActiveSession struct {
	ID        string
	Type      SessionType
	StartTime time.Time
}

// SessionType defines the purpose of a Jules session.
type SessionType string

const (
	// TypeGapAnalysis sessions generate an implementation plan from specs.
	TypeGapAnalysis SessionType = "GAP_ANALYSIS"
	// TypeResolution sessions write code based on an approved plan.
	TypeResolution  SessionType = "RESOLUTION"
)

// New creates a new Orchestrator with the given dependencies.
func New(cfg *config.Config, gh *github.Client, jClient *jules.Client, collector *stats.Collector) *Orchestrator {
	return &Orchestrator{
		cfg:            cfg,
		gh:             gh,
		jules:          jClient,
		stats:          collector,
		activeSessions: make(map[string]*ActiveSession),
		lastReset:      time.Now(),
	}
}

// Run starts the main control loop.
// It blocks until the context is canceled.
func (o *Orchestrator) Run(ctx context.Context) error {
	// 1. Resolve Source Name
	if err := o.resolveSourceName(ctx); err != nil {
		o.stats.RecordError()
		return err
	}
	log.Printf("Resolved Jules Source: %s", o.sourceName)

	ticker := time.NewTicker(o.cfg.CheckInterval())
	defer ticker.Stop()

	log.Println("Orchestrator Loop Started (Stateless Bot Mode)")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			o.tick(ctx)
		}
	}
}

func (o *Orchestrator) resolveSourceName(ctx context.Context) error {
	sources, err := o.jules.ListSources(ctx)
	if err != nil {
		return fmt.Errorf("failed to list sources: %w", err)
	}

	targetRepo := strings.ToLower(o.cfg.GithubRepo) // owner/repo
	for _, s := range sources {
		// id is likely "github/owner/repo"
		matchID := strings.ToLower(fmt.Sprintf("%s/%s", s.GithubRepo.Owner, s.GithubRepo.Repo))
		if strings.Contains(strings.ToLower(s.Name), targetRepo) || strings.HasSuffix(matchID, targetRepo) {
			o.sourceName = s.Name
			return nil
		}
	}
	return fmt.Errorf("source for repo %s not found in Jules", o.cfg.GithubRepo)
}

func (o *Orchestrator) tick(ctx context.Context) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// 1. Reset Daily Limit if needed
	if time.Since(o.lastReset) > 24*time.Hour {
		o.dailyCount = 0
		o.lastReset = time.Now()
		o.stats.SetDailyCount(0)
		log.Println("Daily limit counter reset")
	}

	o.stats.LogStats()

	// 2. Check Active Sessions
	o.checkActiveSessions(ctx)

	// 3. Find New Work
	// Ensure SEQUENTIAL execution: Only if NO active sessions.
	if len(o.activeSessions) == 0 && o.dailyCount < o.cfg.MaxSessionsPerDay {
		o.findNewWork(ctx)
	}
}

func (o *Orchestrator) checkActiveSessions(ctx context.Context) {
	// Iterate over copy of keys to allow deletion
	keys := make([]string, 0, len(o.activeSessions))
	for k := range o.activeSessions {
		keys = append(keys, k)
	}

	for _, sessionID := range keys {
		sessionMeta := o.activeSessions[sessionID]

		sess, err := o.jules.GetSession(ctx, sessionID)
		if err != nil {
			log.Printf("Error getting session %s: %v", sessionID, err)
			o.stats.RecordError()
			continue
		}

		// Check for PR
		for _, output := range sess.Outputs {
			if output.PullRequest != nil {
				log.Printf("Session %s generated PR: %s", sessionID, output.PullRequest.URL)
				o.handleSessionCompletion(ctx, sessionMeta, output.PullRequest)
				delete(o.activeSessions, sessionID)

				o.dailyCount++
				o.stats.SetDailyCount(o.dailyCount)

				duration := time.Since(sessionMeta.StartTime)
				if sessionMeta.Type == TypeGapAnalysis {
					o.stats.RecordGapDuration(duration)
				} else {
					o.stats.RecordResolutionDuration(duration)
				}

				break // Session done
			}
		}
	}
}

func (o *Orchestrator) handleSessionCompletion(ctx context.Context, meta *ActiveSession, pr *jules.PullRequestOutput) {
	// 1. Merge PR
	parts := strings.Split(pr.URL, "/")
	prNumStr := parts[len(parts)-1]
	var prNum int
	fmt.Sscanf(prNumStr, "%d", &prNum)

	err := o.gh.MergePR(ctx, prNum)
	if err != nil {
		log.Printf("Failed to merge PR %d: %v.", prNum, err)
		o.stats.RecordError()
	} else {
		o.stats.IncPRMerged()
		log.Printf("Merged PR %d (Type: %s)", prNum, meta.Type)
	}
}

func (o *Orchestrator) findNewWork(ctx context.Context) {
	// 1. Check for Specs (Remote)
	files, err := o.gh.ListFiles(ctx, o.cfg.SpecPath)
	if err != nil {
		// Log rarely to avoid spam or if error is other than Not Found
		// If 404, valid state (no specs yet).
		return
	}

	var specContents []string
	hasSpecs := false
	for _, filename := range files {
		if strings.HasSuffix(filename, ".md") {
			hasSpecs = true
			path := fmt.Sprintf("%s/%s", o.cfg.SpecPath, filename)
			content, err := o.gh.GetFileContent(ctx, path)
			if err == nil {
				specContents = append(specContents, fmt.Sprintf("File: %s\n%s", filename, content))
			}
		}
	}

	if !hasSpecs {
		return // No specs to process
	}

	// 2. Check Plan (Remote)
	planContent, err := o.gh.GetFileContent(ctx, o.cfg.ImplPlanPath)
	planExists := err == nil && planContent != ""

	// 3. Decision Logic
	if !planExists {
		o.startGapAnalysis(ctx, specContents)
		return
	}

	if strings.Contains(planContent, "Status: Approved") {
		o.startResolution(ctx, planContent, specContents)
		return
	}
}

func (o *Orchestrator) getAgentsMemory(ctx context.Context) string {
	content, err := o.gh.GetFileContent(ctx, o.cfg.AgentsPromptPath)
	if err != nil {
		return "No memory file found."
	}
	return content
}

func (o *Orchestrator) getSystemPrompt(ctx context.Context) string {
	content, err := o.gh.GetFileContent(ctx, o.cfg.SystemPromptPath)
	if err != nil {
		return ""
	}
	return content
}

func (o *Orchestrator) startGapAnalysis(ctx context.Context, specs []string) {
	tmplStr, err := prompts.GetTemplate("gap_analysis.md", o.cfg.GapAnalysisTemplatePath)
	if err != nil {
		log.Printf("Failed to load prompt template: %v", err)
		return
	}

	data := struct {
		SystemPrompt           string
		AgentsMemory           string
		SpecContent            string
		ImplementationPlanPath string
	}{
		SystemPrompt:           o.getSystemPrompt(ctx),
		AgentsMemory:           o.getAgentsMemory(ctx),
		SpecContent:            strings.Join(specs, "\n---\n"),
		ImplementationPlanPath: o.cfg.ImplPlanPath,
	}

	fullPrompt, err := o.renderTemplate(tmplStr, data)
	if err != nil {
		log.Printf("Failed to render prompt: %v", err)
		return
	}

	sessionTitle := fmt.Sprintf("Ralph Gap Analysis: %d Specs", len(specs))
	sess, err := o.jules.CreateSession(ctx, sessionTitle, fullPrompt, o.sourceName, "main")
	if err != nil {
		log.Printf("Failed to create Jules session: %v", err)
		o.stats.RecordError()
		return
	}

	o.activeSessions[sess.Name] = &ActiveSession{
		ID:        sess.Name,
		Type:      TypeGapAnalysis,
		StartTime: time.Now(),
	}
	o.stats.IncSessionCount()
	log.Printf("Started Gap Analysis Session %s", sess.Name)
}

func (o *Orchestrator) startResolution(ctx context.Context, plan string, specs []string) {
	tmplStr, err := prompts.GetTemplate("resolution.md", o.cfg.ResolutionTemplatePath)
	if err != nil {
		log.Printf("Failed to load prompt template: %v", err)
		return
	}

	data := struct {
		SystemPrompt           string
		AgentsMemory           string
		PlanContent            string
		SpecContent            string
		ImplementationPlanPath string
	}{
		SystemPrompt:           o.getSystemPrompt(ctx),
		AgentsMemory:           o.getAgentsMemory(ctx),
		PlanContent:            plan,
		SpecContent:            strings.Join(specs, "\n---\n"),
		ImplementationPlanPath: o.cfg.ImplPlanPath,
	}

	fullPrompt, err := o.renderTemplate(tmplStr, data)
	if err != nil {
		log.Printf("Failed to render prompt: %v", err)
		return
	}

	sessionTitle := "Ralph Resolution"
	sess, err := o.jules.CreateSession(ctx, sessionTitle, fullPrompt, o.sourceName, "main")
	if err != nil {
		log.Printf("Failed to create Jules session: %v", err)
		o.stats.RecordError()
		return
	}

	o.activeSessions[sess.Name] = &ActiveSession{
		ID:        sess.Name,
		Type:      TypeResolution,
		StartTime: time.Now(),
	}
	o.stats.IncSessionCount()
	log.Printf("Started Resolution Session %s", sess.Name)
}

func (o *Orchestrator) renderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("prompt").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
