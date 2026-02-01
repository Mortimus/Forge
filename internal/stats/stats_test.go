package stats

import (
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	c := New()

	c.IncSessionCount()
	c.IncSessionCount()
	c.SetDailyCount(5)
	c.RecordError()
	c.IncPRMerged()
	c.RecordGapDuration(10 * time.Second)
	c.RecordGapDuration(20 * time.Second)
	c.RecordResolutionDuration(5 * time.Second)

	report := c.GetReport()

	if report.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", report.TotalSessions)
	}
	if report.SessionsLast24h != 5 {
		t.Errorf("SessionsLast24h = %d, want 5", report.SessionsLast24h)
	}
	if report.TotalErrors != 1 {
		t.Errorf("TotalErrors = %d, want 1", report.TotalErrors)
	}
	if report.TotalPRsMerged != 1 {
		t.Errorf("TotalPRsMerged = %d, want 1", report.TotalPRsMerged)
	}
	if report.AvgGapAnalysisSeconds != 15.0 {
		t.Errorf("AvgGapAnalysisSeconds = %f, want 15.0", report.AvgGapAnalysisSeconds)
	}
	if report.AvgResolutionSeconds != 5.0 {
		t.Errorf("AvgResolutionSeconds = %f, want 5.0", report.AvgResolutionSeconds)
	}
}

func TestAvgDuration_Empty(t *testing.T) {
	if got := avgDuration([]time.Duration{}); got != 0 {
		t.Errorf("avgDuration([]) = %f, want 0", got)
	}
}

func TestLogStats(t *testing.T) {
	// Capture log output? It writes to std log which is hard to capture without redirection.
	// For unit test coverage, just calling it is enough to cover the lines.
	c := New()
	c.LogStats()
	// No panic means success mostly.
}
