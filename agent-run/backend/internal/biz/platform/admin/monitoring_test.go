package admin

import (
	"testing"
	"time"
)

func TestMonitoringDurationUsesFrozenGenerationTime(t *testing.T) {
	startedAt := time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC)
	generatedAt := startedAt.Add(90 * time.Second)

	if got := monitoringDurationMs(&startedAt, nil, generatedAt); got == nil || *got != 90_000 {
		t.Fatalf("running duration = %v, want 90000ms", got)
	}
	completedAt := startedAt.Add(30 * time.Second)
	if got := monitoringDurationMs(&startedAt, &completedAt, generatedAt); got == nil || *got != 30_000 {
		t.Fatalf("completed duration = %v, want 30000ms", got)
	}
	if got := monitoringDurationMs(nil, nil, generatedAt); got != nil {
		t.Fatalf("duration without start = %v, want nil", got)
	}
}
