// Package stats provides functionality for collecting and reporting application metrics.
package stats

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Collector accumulates statistics about the application's performance and activity.
// It is safe for concurrent use.
type Collector struct {
	mu sync.Mutex

	Sessions24h int
	Errors      int

	GapAnalysisDurations []time.Duration
	ResolutionDurations  []time.Duration

	// Sliding window for 24h reset handled by orchestrator,
	// but here we just store the count provided by caller or increment?
	// Let's make this simple: Orchestrator manages the logic of "what is 24h",
	// this struct just holds data for display.

	TotalSessions int
	PRsMerged     int

	Repository       string
	ActiveSessionIDs []string
	LastPR           string
	StartTime        time.Time
}

// Report represents a snapshot of the current statistics, suitable for JSON output.
type Report struct {
	SessionsLast24h       int      `json:"sessions_last_24h"`
	TotalErrors           int      `json:"total_errors"`
	AvgGapAnalysisSeconds float64  `json:"avg_gap_analysis_seconds"`
	AvgResolutionSeconds  float64  `json:"avg_resolution_seconds"`
	TotalGapSessions      int      `json:"total_gap_sessions"`
	TotalResolutionSess   int      `json:"total_resolution_sessions"`
	TotalSessions         int      `json:"total_sessions"`
	TotalPRsMerged        int      `json:"total_prs_merged"`
	Repository            string   `json:"repository"`
	ActiveSessions        int      `json:"active_sessions"`
	ActiveSessionIDs      []string `json:"active_session_ids"`
	Uptime                string   `json:"uptime"`
}

// New creates a new Collector instance.
func New() *Collector {
	return &Collector{
		GapAnalysisDurations: make([]time.Duration, 0),
		ResolutionDurations:  make([]time.Duration, 0),
		StartTime:            time.Now(),
	}
}

// RecordError increments the total error count.
func (c *Collector) RecordError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Errors++
}

// SetDailyCount updates the count of sessions in the last 24 hours.
func (c *Collector) SetDailyCount(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Sessions24h = count
}

// IncSessionCount increments the total lifetime session count.
func (c *Collector) IncSessionCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TotalSessions++
}

// IncPRMerged increments the count of Pull Requests merged.
func (c *Collector) IncPRMerged() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.PRsMerged++
}

// SetRepository sets the name of the repository being monitored.
func (c *Collector) SetRepository(repo string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Repository = repo
}

// SetActiveSessions sets the list of currently active Jules session IDs.
func (c *Collector) SetActiveSessions(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ActiveSessionIDs = ids
}

// RecordGapDuration adds a duration measurement for a Gap Analysis session.
func (c *Collector) RecordGapDuration(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.GapAnalysisDurations = append(c.GapAnalysisDurations, d)
}

// RecordResolutionDuration adds a duration measurement for a Resolution session.
func (c *Collector) RecordResolutionDuration(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ResolutionDurations = append(c.ResolutionDurations, d)
}

// GetTotalSessions returns the lifetime session count.
func (c *Collector) GetTotalSessions() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.TotalSessions
}

// SetTotalSessions sets the lifetime session count (used for recovery).
func (c *Collector) SetTotalSessions(count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TotalSessions = count
}

// GetLastPR returns the URL of the most recent PR.
func (c *Collector) GetLastPR() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.LastPR
}

// SetLastPR sets the URL of the most recent PR (used for recovery).
func (c *Collector) SetLastPR(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastPR = url
}

// GetReport returns a snapshot of the current statistics.
func (c *Collector) GetReport() Report {
	c.mu.Lock()
	defer c.mu.Unlock()

	return Report{
		SessionsLast24h:       c.Sessions24h,
		TotalErrors:           c.Errors,
		AvgGapAnalysisSeconds: avgDuration(c.GapAnalysisDurations),
		AvgResolutionSeconds:  avgDuration(c.ResolutionDurations),
		TotalGapSessions:      len(c.GapAnalysisDurations),
		TotalResolutionSess:   len(c.ResolutionDurations),
		TotalSessions:         c.TotalSessions,
		TotalPRsMerged:        c.PRsMerged,
		Repository:            c.Repository,
		ActiveSessions:        len(c.ActiveSessionIDs),
		ActiveSessionIDs:      c.ActiveSessionIDs,
		Uptime:                time.Since(c.StartTime).Round(time.Second).String(),
	}
}

// LogStats logs the current statistics report as JSON to the standard logger.
func (c *Collector) LogStats() {
	report := c.GetReport()
	b, _ := json.MarshalIndent(report, "", "  ")
	log.Printf("Current Stats:\n%s", string(b))
}

func avgDuration(ds []time.Duration) float64 {
	if len(ds) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	return sum.Seconds() / float64(len(ds))
}
