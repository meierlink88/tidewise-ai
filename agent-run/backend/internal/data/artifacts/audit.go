package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

const terminalAuditDefaultTimeWindowHours = 48

func WriteTerminalAudit(root string, execution agentrun.Execution) (map[string]string, error) {
	if root == "" {
		root = "data"
	}
	if execution.ID == "" || (execution.Status != agentrun.StatusFailed && execution.Status != agentrun.StatusSkipped) {
		return nil, fmt.Errorf("terminal failed or skipped Execution is required")
	}
	runRoot := filepath.Join(root, "runs", execution.ID)
	paths := map[string]string{
		"documents":  filepath.Join(root, "documents"),
		"index":      filepath.Join(root, "indexes", "dedup-index.tsv"),
		"candidates": filepath.Join(runRoot, "candidates.jsonl"),
		"summary":    filepath.Join(runRoot, "summary.md"),
		"manifest":   filepath.Join(runRoot, "manifest.json"),
	}
	if payload, err := os.ReadFile(paths["manifest"]); err == nil {
		var existing struct {
			ExecutionID     string `json:"execution_id"`
			ExecutionStatus string `json:"execution_status"`
			PromptSHA256    string `json:"prompt_sha256"`
			StopReason      string `json:"stop_reason"`
			ErrorCode       string `json:"error_code"`
			ErrorSummary    string `json:"error_summary"`
		}
		if json.Unmarshal(payload, &existing) != nil ||
			existing.ExecutionID != execution.ID ||
			existing.ExecutionStatus != string(execution.Status) ||
			existing.PromptSHA256 != execution.PromptSHA256 ||
			existing.StopReason != execution.StopReason ||
			existing.ErrorCode != execution.ErrorCode ||
			existing.ErrorSummary != execution.ErrorSummary {
			return nil, fmt.Errorf("terminal audit manifest conflicts with Execution")
		}
		return paths, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	stagingParent := filepath.Join(root, ".staging")
	if err := os.MkdirAll(stagingParent, 0o755); err != nil {
		return nil, err
	}
	stageRoot, err := os.MkdirTemp(stagingParent, "audit-"+execution.ID+"-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stageRoot)
	stagedCandidates := filepath.Join(stageRoot, "candidates.jsonl")
	stagedSummary := filepath.Join(stageRoot, "summary.md")
	stagedManifest := filepath.Join(stageRoot, "manifest.json")
	if err := atomicWrite(stagedCandidates, nil, 0o644); err != nil {
		return nil, err
	}

	completedAt := execution.CreatedAt.UTC()
	if execution.CompletedAt != nil {
		completedAt = execution.CompletedAt.UTC()
	}
	counts := terminalCandidateCounts(execution.CandidateCounts)
	windowEnd := execution.CreatedAt.UTC()
	windowStart := windowEnd.Add(-terminalAuditDefaultTimeWindowHours * time.Hour)
	payload := manifestPayload{
		Schema: "collector_artifact_manifest.v1", ExecutionID: execution.ID,
		ExecutionStatus: string(execution.Status),
		AgentKey:        "collector", AgentVersion: execution.AgentVersion,
		PromptSHA256: execution.PromptSHA256, PromptBytes: execution.PromptBytes,
		StartedAt:       execution.CreatedAt.UTC().Format(time.RFC3339),
		CompletedAt:     completedAt.Format(time.RFC3339),
		TimeWindowHours: terminalAuditDefaultTimeWindowHours,
		WindowStart:     windowStart.Format(time.RFC3339),
		WindowEnd:       windowEnd.Format(time.RFC3339),
		StopReason:      execution.StopReason, ErrorCode: execution.ErrorCode, ErrorSummary: execution.ErrorSummary,
		ConnectorCounts: make(map[string]int), CandidateCounts: counts,
		ContentLevels: map[string]int{
			string(collector.LevelFullText): 0, string(collector.LevelSummary): 0,
			string(collector.LevelSnippet): 0, string(collector.LevelTitle): 0,
		},
		ResultsPending: counts["results_pending"], Artifacts: paths,
	}
	for _, invocation := range execution.Invocations {
		outcome := connectorOutcome{
			Connector: invocation.ConnectorKey, Status: string(invocation.Status),
			ResultCount: invocation.ResultCount, ErrorCode: invocation.ErrorCode,
			ErrorSummary: invocation.ErrorSummary,
		}
		payload.ConnectorOutcomes = append(payload.ConnectorOutcomes, outcome)
		payload.ConnectorCounts[invocation.ConnectorKey] = invocation.ResultCount
		if invocation.Status != agentrun.InvocationPending && invocation.Status != agentrun.InvocationNotInvoked {
			payload.ConnectorsAttempted++
		}
		if invocation.Status == agentrun.InvocationCompleted {
			payload.ConnectorsCompleted++
		}
		if invocation.Status == agentrun.InvocationFailed {
			payload.ConnectorsFailed++
		}
		if invocation.ErrorCode != "" {
			payload.ConnectorFailures = append(payload.ConnectorFailures, connectorFailure{
				Connector: invocation.ConnectorKey, Code: invocation.ErrorCode, Summary: invocation.ErrorSummary,
			})
		}
	}
	if err := writeSummary(stagedSummary, payload); err != nil {
		return nil, err
	}
	if err := writeManifest(stagedManifest, payload); err != nil {
		return nil, err
	}
	for _, pair := range [][2]string{
		{stagedCandidates, paths["candidates"]},
		{stagedSummary, paths["summary"]},
		{stagedManifest, paths["manifest"]},
	} {
		if err := atomicCopy(pair[0], pair[1], 0o644); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

func terminalCandidateCounts(input map[string]int) map[string]int {
	output := map[string]int{
		"raw_results": 0, "merged_results": 0, "results_terminal": 0,
		"results_pending": 0, "accepted": 0, "known_url": 0,
		"out_of_window": 0, "invalid_result": 0,
		"exact_duplicate": 0, "near_duplicate": 0,
	}
	for key := range output {
		if value, exists := input[key]; exists {
			output[key] = value
		}
	}
	return output
}
