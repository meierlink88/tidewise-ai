package agentrun

import (
	"slices"
	"testing"
)

func TestMonitoringStateForStatusUsesExistingRuntimeEnums(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind   MonitoringKind
		status string
		want   MonitoringState
		ok     bool
	}{
		{MonitoringCollector, "partially_succeeded", MonitoringStateSuccess, true},
		{MonitoringCollector, "materializing", MonitoringStateRunning, true},
		{MonitoringCollector, "failed", MonitoringStateFailure, true},
		{MonitoringCollector, "skipped", "", false},
		{MonitoringArtifactExtraction, "no_events", MonitoringStateSuccess, true},
		{MonitoringArtifactExtraction, "retry_wait", MonitoringStateRunning, true},
		{MonitoringArtifactExtraction, "blocked", MonitoringStateFailure, true},
		{MonitoringSemantic, "succeeded", MonitoringStateSuccess, true},
		{MonitoringSemantic, "pending", MonitoringStateRunning, true},
		{MonitoringSemantic, "failed", MonitoringStateFailure, true},
		{MonitoringSemantic, "skipped", "", false},
	}
	for _, test := range tests {
		test := test
		t.Run(string(test.kind)+"/"+test.status, func(t *testing.T) {
			t.Parallel()
			got, ok := MonitoringStateForStatus(test.kind, test.status)
			if got != test.want || ok != test.ok {
				t.Fatalf("state = %q, ok = %v; want %q, %v", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestMonitoringStatusesExcludesSkippedAndReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()
	statuses, ok := MonitoringStatuses(MonitoringCollector, MonitoringStateAll)
	if !ok || slices.Contains(statuses, "skipped") {
		t.Fatalf("statuses = %#v, ok = %v", statuses, ok)
	}
	statuses[0] = "mutated"
	again, _ := MonitoringStatuses(MonitoringCollector, MonitoringStateSuccess)
	if slices.Contains(again, "mutated") {
		t.Fatal("MonitoringStatuses returned shared mutable state")
	}
}
