package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/mortimus/forge/internal/clients/github"
	"github.com/mortimus/forge/internal/clients/jules"
	"github.com/mortimus/forge/internal/config"
	"github.com/mortimus/forge/internal/persistence"
	"github.com/mortimus/forge/internal/prompts"
	"github.com/mortimus/forge/internal/stats"
)

// Orchestrator manages the lifecycle of automated coding sessions for multiple repositories.
type Orchestrator struct {
	cfg   *config.Config
	jules jules.ClientInterface
	stats *stats.Collector
	pm    *persistence.Manager

	// State
	mu                sync.Mutex
	activeSessions    map[string]*ActiveSession // Key: SessionID
	activeRepos       map[string]*RepoContext   // Key: RepoName (Owner/Repo)
	backoffMultiplier int                       // Current backoff multiplier (0 = no backoff)
}

// RepoContext holds the runtime state and configuration for a specific repository.
type RepoContext struct {
	Config     config.RepositoryConfig
	GH         github.ClientInterface
	SourceName string
	DailyCount int
	LastReset  time.Time

	// Cache
	LastSpecCheck time.Time
	CachedSpecs   []string
}

// ActiveSession represents a currently running Jules session.
type ActiveSession struct {
	ID        string
	Repo      string
	Type      SessionType
	PRURL     string
	StartTime time.Time
	State              string // Jules Session State
	Handled            bool   // Whether completion has been handled
	LastAutomatedState string // Last state where an automated action was taken
}

// SessionType defines the purpose of a Jules session.
type SessionType string

const (
	TypeGapAnalysis SessionType = "GAP_ANALYSIS"
	TypeResolution  SessionType = "RESOLUTION"
)

// New creates a new Orchestrator.
func New(cfg *config.Config, jClient jules.ClientInterface, collector *stats.Collector, pm *persistence.Manager) *Orchestrator {
	return &Orchestrator{
		cfg:            cfg,
		jules:          jClient,
		stats:          collector,
		pm:             pm,
		activeSessions: make(map[string]*ActiveSession),
		activeRepos:    make(map[string]*RepoContext),
	}
}

// Run starts the main control loops.
func (o *Orchestrator) Run(ctx context.Context) error {
	// Initialize Repositories
	for _, repoCfg := range o.cfg.Repositories {
		gh, err := github.NewClient(ctx, repoCfg.GithubPAT, repoCfg.GithubRepo, 1*time.Second)
		if err != nil {
			return fmt.Errorf("failed to create client for %s: %w", repoCfg.GithubRepo, err)
		}
        
        sourceName, err := o.resolveSourceName(ctx, repoCfg.GithubRepo)
        if err != nil {
             log.Printf("Warning: Failed to resolve source for %s: %v. Skipping.", repoCfg.GithubRepo, err)
             continue
        }

		o.activeRepos[repoCfg.GithubRepo] = &RepoContext{
			Config:     repoCfg,
			GH:         gh,
			SourceName: sourceName,
			LastReset:  time.Now(),
		}
	}

	if len(o.activeRepos) == 0 {
		return fmt.Errorf("no valid repositories configured")
	}

	// Load State (Global)
	if o.pm != nil {
		o.debugLog("[INIT] Loading persistence state...")
		o.loadState()
	}

	// Start Global Monitor Loop
	go o.monitorLoop(ctx)

	// Start Repo Loops
	var wg sync.WaitGroup
	for _, rc := range o.activeRepos {
		wg.Add(1)
		go func(rc *RepoContext) {
			defer wg.Done()
			o.repoLoop(ctx, rc)
		}(rc)
	}

	log.Printf("Orchestrator started with %d repositories.", len(o.activeRepos))
	wg.Wait()
    
    // Final save on exit
    o.saveState()
	return nil
}

func (o *Orchestrator) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(o.cfg.CheckInterval())
	defer ticker.Stop()

    o.debugLog("[MONITOR] Initial session sync...")
    o.syncActiveSessions(ctx)
    o.debugLog("[MONITOR] Initial sync complete. Waiting %v until next check.", o.cfg.CheckInterval())

	for {
		select {
		case <-ctx.Done():
			o.debugLog("[MONITOR] Context cancelled, stopping monitor loop.")
			return
		case <-ticker.C:
			o.debugLog("[MONITOR] Starting global session sync cycle")
			backoff := o.getBackoffDuration()
			if backoff > 0 {
				o.debugLog("[MONITOR] Backoff active, waiting %v", backoff)
				select {
				case <-ctx.Done():
					o.debugLog("[MONITOR] Context cancelled during backoff, stopping monitor loop.")
					return
				case <-time.After(backoff):
					o.debugLog("[MONITOR] Backoff finished")
				}
			}
			o.syncActiveSessions(ctx)
			o.debugLog("[MONITOR] Cycle complete, waiting %v until next check", o.cfg.CheckInterval())
		}
	}
}

func (o *Orchestrator) repoLoop(ctx context.Context, rc *RepoContext) {
	o.debugLog("[%s] Starting repository loop...", rc.Config.GithubRepo)
	ticker := time.NewTicker(o.cfg.CheckInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			o.debugLog("[%s] Context cancelled, stopping repo loop.", rc.Config.GithubRepo)
			return
		case <-ticker.C:
			o.debugLog("[%s] Starting repo processing cycle", rc.Config.GithubRepo)
			backoff := o.getBackoffDuration()
			if backoff > 0 {
				o.debugLog("[%s] Backoff active, waiting %v", rc.Config.GithubRepo, backoff)
				select {
				case <-ctx.Done():
					o.debugLog("[%s] Context cancelled during backoff, stopping repo loop.", rc.Config.GithubRepo)
					return
				case <-time.After(backoff):
					o.debugLog("[%s] Backoff finished", rc.Config.GithubRepo)
				}
			}
			o.processRepo(ctx, rc)
			o.debugLog("[%s] Cycle complete, waiting %v until next check", rc.Config.GithubRepo, o.cfg.CheckInterval())
		}
	}
}

func (o *Orchestrator) syncActiveSessions(ctx context.Context) {
	o.debugLog("[SYNC] Fetching active sessions from Jules API...")
	sessions, err := o.jules.ListSessions(ctx)
	if err != nil {
		log.Printf("[SYNC] Error listing Jules sessions: %v", err)
		// API error -> increase backoff
		o.increaseBackoff()
		return
	}
	o.debugLog("[SYNC] Successfully fetched %d sessions from Jules API.", len(sessions))

	o.mu.Lock()
	defer o.mu.Unlock()

	existing := make(map[string]*ActiveSession)
	for id, s := range o.activeSessions {
		existing[id] = s
	}

	for _, s := range sessions {
        var repoName string
        
        for rName, rc := range o.activeRepos {
            if s.SourceContext.Source == rc.SourceName {
                repoName = rName
                break
            }
        }
        
        if repoName == "" {
             for rName := range o.activeRepos {
                 if strings.Contains(strings.ToLower(s.SourceContext.Source), strings.ToLower(rName)) {
                     repoName = rName
                     break
                 }
             }
        }

        if repoName == "" {
            o.debugLog("[SYNC] Skipping session %s: No matching repository found for source '%s'", s.Name, s.SourceContext.Source)
            continue
        }

		if current, ok := existing[s.Name]; ok {
			if current.State != s.State {
				o.debugLog("[SYNC] Updating session %s state from %s to %s (Repo: %s)", s.Name, current.State, s.State, repoName)
				current.State = s.State
			}
		} else {
             isTerminal := isTerminalState(s.State)
             if !isTerminal {
                  o.debugLog("[SYNC] Adding new active session %s (State: %s, Repo: %s)", s.Name, s.State, repoName)
                  o.activeSessions[s.Name] = &ActiveSession{
                      ID: s.Name,
                      Repo: repoName,
                      Type: TypeResolution, 
                      StartTime: time.Now(),
                      State: s.State,
                  }
             } else {
                 o.debugLog("[SYNC] Skipping new session %s in terminal state %s", s.Name, s.State)
             }
		}
	}
    
    o.debugLog("[SYNC] State synchronized: %d active sessions", len(o.activeSessions))
    o.saveStateInternal()
}

func (o *Orchestrator) processRepo(ctx context.Context, rc *RepoContext) {
	if time.Since(rc.LastReset) > 24*time.Hour {
         rc.DailyCount = 0
         rc.LastReset = time.Now()
         o.debugLog("[%s] Daily count reset to 0.", rc.Config.GithubRepo)
    }

    o.mu.Lock()
    var mySession *ActiveSession
    for _, s := range o.activeSessions {
        if s.Repo == rc.Config.GithubRepo {
             mySession = s
             break
        }
    }
    o.mu.Unlock()

    if mySession != nil {
        o.debugLog("[%s] Active session %s found (State: %s). Processing...", rc.Config.GithubRepo, mySession.ID, mySession.State)
        if isTerminalState(mySession.State) {
             o.mu.Lock()
             if !mySession.Handled {
                 mySession.Handled = true
                 o.mu.Unlock()
                 o.handleCompletion(ctx, rc, mySession)
                 
                 o.mu.Lock()
                 delete(o.activeSessions, mySession.ID)
                 o.debugLog("[%s] Removed completed session %s from active sessions.", rc.Config.GithubRepo, mySession.ID)
             } else {
                 o.debugLog("[%s] Session %s is in terminal state %s but already handled. Removing.", rc.Config.GithubRepo, mySession.ID, mySession.State)
                 delete(o.activeSessions, mySession.ID)
             }
             o.mu.Unlock()
             return
        }
        
        // Automated Interactions for Non-Terminal States
        o.handleAutomatedInteractions(ctx, rc, mySession)
        return 
    }

    o.findNewWork(ctx, rc)
}

func (o *Orchestrator) handleAutomatedInteractions(ctx context.Context, rc *RepoContext, sess *ActiveSession) {
	o.mu.Lock()
	if sess.LastAutomatedState == sess.State {
		o.mu.Unlock()
		o.debugLog("[%s] Session %s in state %s already handled, skipping automated interaction.", rc.Config.GithubRepo, sess.ID, sess.State)
		return
	}
	state := sess.State
	o.mu.Unlock()

	switch state {
	case "AWAITING_PLAN_APPROVAL":
		log.Printf("[%s] Automatically approving plan for session %s", rc.Config.GithubRepo, sess.ID)
		if err := o.jules.ApprovePlan(ctx, sess.ID); err != nil {
			log.Printf("[%s] Failed to approve plan: %v", rc.Config.GithubRepo, err)
			return
		}
		o.mu.Lock()
		sess.LastAutomatedState = state
		o.mu.Unlock()
		o.debugLog("[%s] Approved plan for session %s.", rc.Config.GithubRepo, sess.ID)

	case "AWAITING_USER_FEEDBACK":
		log.Printf("[%s] Automatically sending 'best judgement' to session %s", rc.Config.GithubRepo, sess.ID)
		msg := "Use your best judgement to continue working on the task and keep our automation up and running."
		if err := o.jules.SendMessage(ctx, sess.ID, msg); err != nil {
			log.Printf("[%s] Failed to send message: %v", rc.Config.GithubRepo, err)
			return
		}
		o.mu.Lock()
		sess.LastAutomatedState = state
		o.mu.Unlock()
		o.debugLog("[%s] Sent 'best judgement' message to session %s.", rc.Config.GithubRepo, sess.ID)
	default:
		o.debugLog("[%s] No automated interaction defined for state %s for session %s.", rc.Config.GithubRepo, state, sess.ID)
	}
}

func (o *Orchestrator) findNewWork(ctx context.Context, rc *RepoContext) {
	o.debugLog("[%s] Checking for new work opportunities...", rc.Config.GithubRepo)

	// 1. Check for open PRs
	o.debugLog("[%s] Discovery: Listing open Pull Requests", rc.Config.GithubRepo)
	prs, err := rc.GH.ListOpenPullRequests(ctx)
	if err != nil {
		log.Printf("[%s] Failed to list open PRs: %v", rc.Config.GithubRepo, err)
		return
	}
	if len(prs) > 0 {
		o.debugLog("[%s] Blocking: Found %d open PR(s). Skipping new work.", rc.Config.GithubRepo, len(prs))
		return
	}
	o.debugLog("[%s] Discovery: No open PRs found.", rc.Config.GithubRepo)

	// 2. Refresh Specs (if needed)
	var files []string
	if time.Since(rc.LastSpecCheck) > 10*time.Minute || len(rc.CachedSpecs) == 0 {
		o.debugLog("[%s] Discovery: Scanning spec directory %s", rc.Config.GithubRepo, rc.Config.SpecPath)
		files, err = rc.GH.ListFiles(ctx, rc.Config.SpecPath)
		if err != nil {
			o.debugLog("[%s] Failed to list specs: %v", rc.Config.GithubRepo, err)
			return
		}
		rc.CachedSpecs = files
		rc.LastSpecCheck = time.Now()
		o.debugLog("[%s] Discovery: Found %d spec(s) and updated cache.", rc.Config.GithubRepo, len(rc.CachedSpecs))
	} else {
		o.debugLog("[%s] Discovery: Using cached specs (%d specs). Next refresh in %v.", rc.Config.GithubRepo, len(rc.CachedSpecs), (10*time.Minute - time.Since(rc.LastSpecCheck)).Round(time.Second))
		files = rc.CachedSpecs
	}

	var specPaths []string
	for _, filename := range files {
		if strings.HasSuffix(filename, ".md") {
			path := fmt.Sprintf("%s/%s", rc.Config.SpecPath, filename)
			specPaths = append(specPaths, path)
		}
	}

	if len(specPaths) == 0 {
		o.debugLog("[%s] Discovery: No specs found. Skipping.", rc.Config.GithubRepo)
		return
	}
	o.debugLog("[%s] Discovery: Found %d markdown spec files.", rc.Config.GithubRepo, len(specPaths))

	// 3. Look for Implementation Plan
	o.debugLog("[%s] Discovery: Checking for existing Implementation Plan at %s", rc.Config.GithubRepo, rc.Config.ImplPlanPath)
	planContent, err := rc.GH.GetFileContent(ctx, rc.Config.ImplPlanPath)
	planExists := err == nil && planContent != ""

	if !planExists {
		o.debugLog("[%s] Starting Gap Analysis: No implementation plan found.", rc.Config.GithubRepo)
		o.startGapAnalysis(ctx, rc, specPaths)
		return
	}

	if strings.Contains(planContent, "Status: Completed") {
		o.startGapAnalysis(ctx, rc, specPaths)
		return
	}

	o.startResolution(ctx, rc, planContent, specPaths)
}

func (o *Orchestrator) startGapAnalysis(ctx context.Context, rc *RepoContext, specFiles []string) {
	tmplStr, err := prompts.GetTemplate("gap_analysis.md", rc.Config.GapAnalysisTemplatePath)
	if err != nil {
		log.Printf("[%s] Failed to load template: %v", rc.Config.GithubRepo, err)
		return
	}

	data := struct {
		SystemPrompt           string
		AgentsMemory           string
		SpecFiles              []string
		ImplementationPlanPath string
	}{
		SystemPrompt:           o.getSystemPrompt(ctx, rc),
		AgentsMemory:           o.getAgentsMemory(ctx, rc),
		SpecFiles:              specFiles,
		ImplementationPlanPath: rc.Config.ImplPlanPath,
	}

	fullPrompt, err := o.renderTemplate(tmplStr, data)
	if err != nil {
		log.Printf("[%s] Prompt render error: %v", rc.Config.GithubRepo, err)
		return
	}

	if err := o.checkRateLimit(); err != nil {
		log.Printf("[%s] Rate limit active: %v", rc.Config.GithubRepo, err)
		return
	}

	sessionTitle := fmt.Sprintf("Forge Gap Analysis: %d Specs (%s)", len(specFiles), rc.Config.GithubRepo)
	sess, err := o.jules.CreateSession(ctx, sessionTitle, fullPrompt, rc.SourceName, "main")
	if err != nil {
		log.Printf("[%s] Session create error: %v", rc.Config.GithubRepo, err)
		o.stats.RecordError()
		// API error -> increase backoff
		o.increaseBackoff()
		return
	}

	o.resetBackoff()
	o.recordSessionStart()

	o.mu.Lock()
	o.activeSessions[sess.Name] = &ActiveSession{
		ID:        sess.Name,
		Repo:      rc.Config.GithubRepo,
		Type:      TypeGapAnalysis,
		StartTime: time.Now(),
		State:     "ACTIVE",
	}
	o.mu.Unlock()

	o.stats.IncSessionCount()
	o.saveState()
	log.Printf("[%s] Started Gap Analysis %s", rc.Config.GithubRepo, sess.Name)
}

func (o *Orchestrator) startResolution(ctx context.Context, rc *RepoContext, plan string, specFiles []string) {
	tmplStr, err := prompts.GetTemplate("resolution.md", rc.Config.ResolutionTemplatePath)
	if err != nil {
		log.Printf("[%s] Template error: %v", rc.Config.GithubRepo, err)
		return
	}

	data := struct {
		SystemPrompt           string
		AgentsMemory           string
		PlanContent            string
		SpecFiles              []string
		ImplementationPlanPath string
	}{
		SystemPrompt:           o.getSystemPrompt(ctx, rc),
		AgentsMemory:           o.getAgentsMemory(ctx, rc),
		PlanContent:            plan,
		SpecFiles:              specFiles,
		ImplementationPlanPath: rc.Config.ImplPlanPath,
	}

	fullPrompt, err := o.renderTemplate(tmplStr, data)
	if err != nil {
		log.Printf("[%s] Render error: %v", rc.Config.GithubRepo, err)
		return
	}

	if err := o.checkRateLimit(); err != nil {
		log.Printf("[%s] Rate limit active: %v", rc.Config.GithubRepo, err)
		return
	}

	sessionTitle := fmt.Sprintf("Forge Resolution (%s)", rc.Config.GithubRepo)
	sess, err := o.jules.CreateSession(ctx, sessionTitle, fullPrompt, rc.SourceName, "main")
	if err != nil {
		log.Printf("[%s] Session error: %v", rc.Config.GithubRepo, err)
		o.stats.RecordError()
		// API error -> increase backoff
		o.increaseBackoff()
		return
	}

	o.resetBackoff()
	o.recordSessionStart()

	o.mu.Lock()
	o.activeSessions[sess.Name] = &ActiveSession{
		ID:        sess.Name,
		Repo:      rc.Config.GithubRepo,
		Type:      TypeResolution,
		StartTime: time.Now(),
		State:     "ACTIVE",
	}
	o.mu.Unlock()

	o.stats.IncSessionCount()
	o.saveState()
	log.Printf("[%s] Started Resolution %s", rc.Config.GithubRepo, sess.Name)
}

func (o *Orchestrator) handleCompletion(ctx context.Context, rc *RepoContext, sess *ActiveSession) {
	log.Printf("[%s] Completing session %s (%s)", rc.Config.GithubRepo, sess.ID, sess.State)

	defer func() {
		if !o.cfg.AutoDeleteSessions {
			o.debugLog("[%s] Auto-delete sessions is disabled. Skipping deletion of session %s.", rc.Config.GithubRepo, sess.ID)
			return
		}
		log.Printf("[%s] Deleting session %s", rc.Config.GithubRepo, sess.ID)
		if err := o.jules.DeleteSession(ctx, sess.ID); err != nil {
			log.Printf("[%s] Failed to delete session %s: %v", rc.Config.GithubRepo, sess.ID, err)
		}
	}()

	fullSess, err := o.jules.GetSession(ctx, sess.ID)
	if err != nil {
		log.Printf("Error fetching session %s: %v", sess.ID, err)
		return
	}

	var prOutput *jules.PullRequestOutput
	for _, output := range fullSess.Outputs {
		if output.PullRequest != nil {
			prOutput = output.PullRequest
			break
		}
	}

	if prOutput == nil {
		if sess.State == "FAILED" && rc.Config.ImplPlanPath != "" {
			 log.Printf("[%s] Session failed. Cleaning plan.", rc.Config.GithubRepo)
			 rc.GH.DeleteFile(ctx, rc.Config.ImplPlanPath, "Cleanup after failure")
		}
		return
	}
	
	sess.PRURL = prOutput.URL
	o.stats.SetLastPR(prOutput.URL)
	rc.DailyCount++ 
	
	parts := strings.Split(prOutput.URL, "/")
	prNumStr := parts[len(parts)-1]
	var prNum int
	fmt.Sscanf(prNumStr, "%d", &prNum)

	if !rc.Config.AutoMerge {
		log.Printf("[%s] PR %d created (Auto-Merge off).", rc.Config.GithubRepo, prNum)
		o.stats.IncPRMerged()
		return
	}

	// Fetch PR for branch name
	githubPR, _ := rc.GH.GetPR(ctx, prNum)

	if err := rc.GH.MergePR(ctx, prNum); err != nil {
		log.Printf("[%s] Merge failed for PR %d: %v", rc.Config.GithubRepo, prNum, err)
		o.stats.RecordError()
		return
	}

	o.stats.IncPRMerged()
	log.Printf("[%s] Merged PR %d", rc.Config.GithubRepo, prNum)

	if githubPR != nil && githubPR.Head != nil && githubPR.Head.Ref != nil {
		branch := *githubPR.Head.Ref
		if branch != "main" && branch != "master" {
			rc.GH.DeleteBranch(ctx, branch)
		}
	}

	if rc.Config.ImplPlanPath != "" && sess.Type == TypeResolution {
		rc.GH.DeleteFile(ctx, rc.Config.ImplPlanPath, "Plan completed")
	}
}

func (o *Orchestrator) resolveSourceName(ctx context.Context, repoName string) (string, error) {
	sources, err := o.jules.ListSources(ctx)
	if err != nil {
		return "", err
	}
	targetRepo := strings.ToLower(repoName)
	for _, s := range sources {
		matchID := strings.ToLower(fmt.Sprintf("%s/%s", s.GithubRepo.Owner, s.GithubRepo.Repo))
		if strings.Contains(strings.ToLower(s.Name), targetRepo) || strings.HasSuffix(matchID, targetRepo) {
			return s.Name, nil
		}
	}
	return "", fmt.Errorf("source not found for %s", repoName)
}

func (o *Orchestrator) getAgentsMemory(ctx context.Context, rc *RepoContext) string {
	content, err := rc.GH.GetFileContent(ctx, rc.Config.AgentsPromptPath)
	if err != nil {
		return "No memory file found."
	}
	return content
}

func (o *Orchestrator) getSystemPrompt(ctx context.Context, rc *RepoContext) string {
    if content, err := rc.GH.GetFileContent(ctx, rc.Config.SystemPromptPath); err == nil {
        return content
    }
    if content, err := os.ReadFile(rc.Config.SystemPromptPath); err == nil {
		return string(content)
	}
	return ""
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

func (o *Orchestrator) debugLog(format string, args ...interface{}) {
	if o.cfg.Debug {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func (o *Orchestrator) saveState() {
	if o.pm == nil {
		return
	}
	o.mu.Lock()
    o.saveStateInternal()
	o.mu.Unlock()
}

func (o *Orchestrator) saveStateInternal() {
	state := &persistence.State{
		LifetimeSessions: o.stats.GetTotalSessions(),
		ActiveSessions:   make(map[string]persistence.SessionMetadata),
		Repositories:     make(map[string]persistence.RepoState),
		LastPR:           o.stats.GetLastPR(),
	}

	for _, rc := range o.activeRepos {
		state.Repositories[rc.Config.GithubRepo] = persistence.RepoState{
			DailyCount: rc.DailyCount,
			LastReset:  rc.LastReset.Format(time.RFC3339),
		}
	}
	for id, sess := range o.activeSessions {
		state.ActiveSessions[id] = persistence.SessionMetadata{
			ID:        sess.ID,
			Repo:      sess.Repo,
			Type:      string(sess.Type),
			PRURL:     sess.PRURL,
			StartTime: sess.StartTime.Format(time.RFC3339),
		}
	}
	if err := o.pm.Save(state); err != nil {
		log.Printf("[ERROR] State save failed: %v", err)
	}
}

func (o *Orchestrator) loadState() {
	state, err := o.pm.Load()
	if err != nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()

	for id, meta := range state.ActiveSessions {
		startTime, _ := time.Parse(time.RFC3339, meta.StartTime)
		o.activeSessions[id] = &ActiveSession{
			ID:        meta.ID,
			Repo:      meta.Repo,
			Type:      SessionType(meta.Type),
			PRURL:     meta.PRURL,
			StartTime: startTime,
		}
	}
	for alias, repoState := range state.Repositories {
		if rc, ok := o.activeRepos[alias]; ok {
			rc.DailyCount = repoState.DailyCount
			if t, err := time.Parse(time.RFC3339, repoState.LastReset); err == nil {
				rc.LastReset = t
			}
		}
	}

	o.stats.SetTotalSessions(state.LifetimeSessions)
	o.stats.SetLastPR(state.LastPR)
	o.debugLog("Loaded state: %d active sessions, %d repos", len(o.activeSessions), len(state.Repositories))
}


func (o *Orchestrator) checkRateLimit() error {
	if o.pm == nil {
		return nil
	}
	state, err := o.pm.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	var activeTimestamps []string

	// Prune old timestamps
	for _, tsStr := range state.SessionTimestamps {
		ts, err := time.Parse(time.RFC3339, tsStr)
		if err != nil {
			continue // Skip malformed
		}
		if ts.After(cutoff) {
			activeTimestamps = append(activeTimestamps, tsStr)
		}
	}

	// Update state if we pruned
	if len(activeTimestamps) != len(state.SessionTimestamps) {
		state.SessionTimestamps = activeTimestamps
		if err := o.pm.Save(state); err != nil {
			log.Printf("Failed to save pruned timestamps: %v", err)
		}
	}

	// Check global limit
	if len(activeTimestamps) >= o.cfg.MaxSessionsPerDay {
		o.increaseBackoff()
		return fmt.Errorf("global rate limit reached (%d/%d in last 24h). Next slot available at %s", 
			len(activeTimestamps), o.cfg.MaxSessionsPerDay, "unknown")
	}
	
	// If we are passing the checks, we can reset backoff partialy or fully? 
	// Actually, if we are NOT limited, we shouldn't necessarily reset immediately unless we successfully do an action.
	// But simply *checking* and passing means we aren't limited. Limit errors cause backoff. Passing doesn't necessarily mean success yet. 
	// I'll leave reset for successful operations.

	return nil
}

func (o *Orchestrator) recordSessionStart() {
	if o.pm == nil {
		return
	}
	state, err := o.pm.Load()
	if err != nil {
		log.Printf("Failed to load state to record session: %v", err)
		return
	}

	state.SessionTimestamps = append(state.SessionTimestamps, time.Now().Format(time.RFC3339))
	if err := o.pm.Save(state); err != nil {
		log.Printf("Failed to save session timestamp: %v", err)
	}
}

func (o *Orchestrator) getBackoffDuration() time.Duration {
	const baseDuration = 5 * time.Minute
	if o.backoffMultiplier == 0 {
		return 0
	}
	// Exponential backoff: 5m, 10m, 20m, 40m, 80m...
	// Cap at some reasonable max, e.g., 6 hours
	duration := baseDuration * time.Duration(1<<uint(o.backoffMultiplier-1))
	maxDuration := 6 * time.Hour
	if duration > maxDuration {
		return maxDuration
	}
	return duration
}

func (o *Orchestrator) increaseBackoff() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.backoffMultiplier++
	log.Printf("[Backpressure] Increasing backoff. Level: %d. Next sleep: %v", o.backoffMultiplier, o.getBackoffDuration())
}

func (o *Orchestrator) resetBackoff() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.backoffMultiplier > 0 {
		o.backoffMultiplier = 0
		log.Printf("[Backpressure] Backoff reset. Streaming resuming normally.")
	}
}

func isTerminalState(state string) bool {
    return state == "COMPLETED" || state == "FAILED" || state == "CANCELLED" || state == "SUCCEEDED"
}

