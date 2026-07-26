package artifacts

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type eventReaderExecutions struct {
	execution agentrun.Execution
}

func (f eventReaderExecutions) GetExecution(context.Context, string) (agentrun.Execution, error) {
	return f.execution, nil
}

func TestEventReaderReadsOnlyVerifiedManifestArtifacts(t *testing.T) {
	root := t.TempDir()
	executionID := "11111111-1111-4111-8111-111111111111"
	collectedAt := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	result, err := (File{Root: root}).Materialize(context.Background(), collector.Request{
		RunID: executionID, Prompt: "collect", CollectedAt: collectedAt, TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{
		"source": {
			Connector: "source",
			Results: []collector.Candidate{{
				Connector: "source", Title: "某公司宣布扩产",
				URL: "https://example.com/news/1", SourceName: "Example",
				SourceType: "news", Content: "某公司于7月26日宣布新增一条产线。",
				ContentLevel: collector.LevelFullText, PublishedAtHint: "2026-07-26T07:00:00Z",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reader := EventReader{
		Root: root,
		Executions: eventReaderExecutions{execution: agentrun.Execution{
			ID: executionID, AgentKey: collector.AgentKey, AgentVersion: collector.AgentVersion,
			Status: agentrun.StatusSucceeded, Artifacts: map[string]string{"manifest": result.Manifest},
		}},
	}
	artifacts, err := reader.Read(context.Background(), []string{executionID})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	artifact := artifacts[0]
	if artifact.CollectorExecutionID != executionID || artifact.ArtifactID != artifact.DocumentID ||
		!strings.Contains(artifact.Body, "新增一条产线") || artifact.ContentSHA256 == "" {
		t.Fatalf("artifact = %#v", artifact)
	}

	if err := os.WriteFile(result.AcceptedDocuments[0], []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := reader.Read(context.Background(), []string{executionID}); err == nil ||
		!strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("tampered Artifact error = %v", err)
	}
}

func TestEventReaderRejectsUnregisteredManifestPath(t *testing.T) {
	reader := EventReader{
		Root: t.TempDir(),
		Executions: eventReaderExecutions{execution: agentrun.Execution{
			ID: "11111111-1111-4111-8111-111111111111", AgentKey: collector.AgentKey,
			AgentVersion: collector.AgentVersion, Status: agentrun.StatusSucceeded,
			Artifacts: map[string]string{"manifest": "/tmp/other-manifest.json"},
		}},
	}
	if _, err := reader.Read(context.Background(), []string{"11111111-1111-4111-8111-111111111111"}); err == nil ||
		!strings.Contains(err.Error(), "Manifest identity") {
		t.Fatalf("unregistered Manifest error = %v", err)
	}
}
