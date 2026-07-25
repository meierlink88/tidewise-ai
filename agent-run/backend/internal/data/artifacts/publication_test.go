package artifacts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type memoryPublicationRepository struct {
	reference         *agentrun.PublicationReference
	completion        *agentrun.ExecutionCompletion
	commitCalls       int
	unknownCommitOnce bool
	prepareError      error
	unknownPrepare    bool
}

func (r *memoryPublicationRepository) PreparePublication(_ context.Context, reference agentrun.PublicationReference) error {
	if r.unknownPrepare {
		copy := reference
		r.reference = &copy
		return errors.New("injected unknown prepare result")
	}
	if r.prepareError != nil {
		return r.prepareError
	}
	if r.reference != nil &&
		(r.reference.ExecutionID != reference.ExecutionID ||
			r.reference.PlanPath != reference.PlanPath ||
			r.reference.PlanSHA256 != reference.PlanSHA256) {
		return errors.New("publication identity conflict")
	}
	copy := reference
	r.reference = &copy
	return nil
}

func TestUnknownPrepareResultPreservesAndReconcilesPendingPlan(t *testing.T) {
	root := t.TempDir()
	repository := &memoryPublicationRepository{unknownPrepare: true}
	runID := "prepare-unknown"
	_, err := (File{
		Root: root, NearDuplicateRadius: 3, Publications: repository,
	}).Materialize(context.Background(), collector.Request{
		RunID: runID, Prompt: "collect",
		CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
			Content: "direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if !errors.Is(err, ErrPublicationPending) {
		t.Fatalf("Materialize error = %v, want ErrPublicationPending", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".pending", runID, "plan.json")); statErr != nil {
		t.Fatalf("unknown prepare deleted durable plan: %v", statErr)
	}
	repository.unknownPrepare = false
	if err := ReconcilePreparedPublications(context.Background(), root, repository); err != nil {
		t.Fatal(err)
	}
	if repository.completion == nil {
		t.Fatal("unknown prepare result was not reconciled")
	}
}

func TestStartupCleanupRemovesOnlyUnreferencedPendingDirectory(t *testing.T) {
	root := t.TempDir()
	orphan := filepath.Join(root, ".pending", "orphan", "plan.json")
	if err := os.MkdirAll(filepath.Dir(orphan), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphan, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReconcilePreparedPublications(context.Background(), root, &memoryPublicationRepository{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(orphan)); !os.IsNotExist(err) {
		t.Fatalf("orphaned pending directory remains: %v", err)
	}
}

func TestPrepareFailureRemovesUnreferencedPendingPayload(t *testing.T) {
	root := t.TempDir()
	repository := &memoryPublicationRepository{prepareError: errors.New("injected prepare failure")}
	runID := "prepare-failed"
	_, err := (File{
		Root: root, NearDuplicateRadius: 3, Publications: repository,
	}).Materialize(context.Background(), collector.Request{
		RunID: runID, Prompt: "collect",
		CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{"tavily": {
		Results: []collector.Candidate{{
			Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
			Content: "direct result", ContentLevel: collector.LevelSnippet,
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "prepare Artifact publication") {
		t.Fatalf("Materialize error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".pending", runID)); !os.IsNotExist(statErr) {
		t.Fatalf("unprepared pending payload remains: %v", statErr)
	}
}

func (r *memoryPublicationRepository) ListPreparedPublications(context.Context) ([]agentrun.PublicationReference, error) {
	if r.reference == nil || r.completion != nil {
		return nil, nil
	}
	return []agentrun.PublicationReference{*r.reference}, nil
}

func (r *memoryPublicationRepository) CommitPreparedPublication(
	_ context.Context,
	reference agentrun.PublicationReference,
	completion agentrun.ExecutionCompletion,
) error {
	r.commitCalls++
	if r.reference == nil || r.reference.PlanSHA256 != reference.PlanSHA256 {
		return errors.New("publication was not prepared")
	}
	copy := completion
	r.completion = &copy
	if r.unknownCommitOnce && r.commitCalls == 1 {
		return errors.New("injected unknown commit result")
	}
	return nil
}

func TestPreparedPublicationReconcilesAfterIndexBoundaryWithoutRecollecting(t *testing.T) {
	root := t.TempDir()
	repository := &memoryPublicationRepository{}
	publishSteps := 0
	materializer := File{
		Root: root, NearDuplicateRadius: 3, Publications: repository,
		BeforePublish: func(kind string) error {
			publishSteps++
			if kind == "index" {
				return errors.New("injected process interruption")
			}
			return nil
		},
	}
	request := collector.Request{
		RunID: "recoverable", Prompt: "secret full collection Prompt", CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC),
		TimeWindowHours: 48,
	}
	_, err := materializer.Materialize(context.Background(), request, map[string]collector.ConnectorRun{
		"tavily": {
			Connector: "tavily",
			Results: []collector.Candidate{{
				Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
				Content: "direct result", ContentLevel: collector.LevelSnippet,
			}},
		},
	})
	if !errors.Is(err, ErrPublicationPending) {
		t.Fatalf("Materialize error = %v, want ErrPublicationPending", err)
	}
	if repository.reference == nil || repository.completion != nil {
		t.Fatalf("repository state reference=%#v completion=%#v", repository.reference, repository.completion)
	}
	if publishSteps == 0 {
		t.Fatal("fault injection did not reach publication")
	}
	for _, path := range []string{
		filepath.Join(root, "runs", request.RunID, "candidates.jsonl"),
		filepath.Join(root, "runs", request.RunID, "summary.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("pre-index Artifact %s was not published: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "indexes", "dedup-index.tsv")); !os.IsNotExist(err) {
		t.Fatalf("index was published before injected boundary: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "runs", request.RunID, "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("manifest was published before index: %v", err)
	}
	if _, err := os.Stat(repository.reference.PlanPath); err != nil {
		t.Fatalf("durable plan missing: %v", err)
	}
	planPayload, err := os.ReadFile(repository.reference.PlanPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(planPayload), request.Prompt) {
		t.Fatalf("publication plan leaked Prompt: %s", planPayload)
	}

	if err := ReconcilePreparedPublications(context.Background(), root, repository); err != nil {
		t.Fatal(err)
	}
	if repository.completion == nil ||
		repository.completion.Status != agentrun.StatusSucceeded ||
		repository.completion.StopReason != "connectors_completed" ||
		repository.completion.CandidateCounts["results_pending"] != 0 {
		t.Fatalf("completion = %#v", repository.completion)
	}
	for _, path := range []string{
		filepath.Join(root, "indexes", "dedup-index.tsv"),
		filepath.Join(root, "runs", request.RunID, "manifest.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("reconciled Artifact %s missing: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Dir(repository.reference.PlanPath)); !os.IsNotExist(err) {
		t.Fatalf("pending publication was not cleaned: %v", err)
	}
}

func TestPreparedPublicationRecoversEveryOrderedFileBoundary(t *testing.T) {
	for _, boundary := range []string{"document", "candidates", "summary", "index", "manifest"} {
		t.Run(boundary, func(t *testing.T) {
			root := t.TempDir()
			repository := &memoryPublicationRepository{}
			failed := false
			materializer := File{
				Root: root, NearDuplicateRadius: 3, Publications: repository,
				BeforePublish: func(kind string) error {
					if kind == boundary && !failed {
						failed = true
						return errors.New("injected boundary")
					}
					return nil
				},
			}
			_, err := materializer.Materialize(context.Background(), collector.Request{
				RunID: "boundary-" + boundary, Prompt: "collect",
				CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), TimeWindowHours: 48,
			}, map[string]collector.ConnectorRun{"tavily": {
				Results: []collector.Candidate{{
					Connector: "tavily", Title: "Policy", URL: "https://example.com/" + boundary,
					Content: "direct result", ContentLevel: collector.LevelSnippet,
				}},
			}})
			if !errors.Is(err, ErrPublicationPending) || !failed {
				t.Fatalf("Materialize error = %v, failed=%v", err, failed)
			}
			if err := ReconcilePreparedPublications(context.Background(), root, repository); err != nil {
				t.Fatal(err)
			}
			if repository.completion == nil {
				t.Fatal("publication did not commit after reconciliation")
			}
			if _, err := os.Stat(repository.completion.Artifacts["manifest"]); err != nil {
				t.Fatalf("manifest missing: %v", err)
			}
		})
	}
}

func TestPreparedPublicationRejectsConflictingExistingTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, ".pending", "run", "payload", "summary.md")
	target := filepath.Join(root, "runs", "run", "summary.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("planned"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("different"), 0o644); err != nil {
		t.Fatal(err)
	}
	item, err := buildPublicationItem("summary", source, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := publishImmutableItem(item); err == nil {
		t.Fatal("conflicting target was accepted")
	}
}

func TestPublicationRetriesUnknownCommitResultWithoutDowngradingExecution(t *testing.T) {
	root := t.TempDir()
	repository := &memoryPublicationRepository{unknownCommitOnce: true}
	completedAt := time.Date(2026, 7, 22, 9, 10, 11, 123456789, time.UTC)
	result, err := (File{
		Root: root, NearDuplicateRadius: 3, Publications: repository,
		Now: func() time.Time {
			return completedAt
		},
	}).Materialize(context.Background(), collector.Request{
		RunID: "unknown-commit", Prompt: "collect",
		CollectedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), TimeWindowHours: 48,
	}, map[string]collector.ConnectorRun{
		"tavily": {
			Connector: "tavily",
			Results: []collector.Candidate{{
				Connector: "tavily", Title: "Policy", URL: "https://example.com/policy",
				Content: "direct result", ContentLevel: collector.LevelSnippet,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || repository.commitCalls != 2 || repository.completion == nil {
		t.Fatalf("result=%#v commit_calls=%d completion=%#v", result, repository.commitCalls, repository.completion)
	}
	if !repository.completion.CompletedAt.Equal(completedAt) {
		t.Fatalf("database completion time = %s, want %s", repository.completion.CompletedAt, completedAt)
	}
	if _, err := os.Stat(result.Manifest); err != nil {
		t.Fatalf("manifest missing after unknown commit retry: %v", err)
	}
	manifest, err := os.ReadFile(result.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), `"completed_at": "`+completedAt.Format(time.RFC3339Nano)+`"`) {
		t.Fatalf("manifest completion time differs from database completion:\n%s", manifest)
	}
}

func TestWriteTerminalAuditIsSafeCompleteAndIdempotent(t *testing.T) {
	root := t.TempDir()
	completedAt := time.Date(2026, 7, 22, 8, 1, 0, 0, time.UTC)
	execution := agentrun.Execution{
		ID: "failed-run", AgentVersion: "collector.v1",
		Prompt: "secret full Prompt", PromptSHA256: strings.Repeat("a", 64), PromptBytes: 18,
		Status: agentrun.StatusFailed, StopReason: "agent_or_tool_limit",
		ErrorCode: "planning_failed", ErrorSummary: "Query planning failed",
		CreatedAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), CompletedAt: &completedAt,
		Invocations: []agentrun.ConnectorInvocation{{
			ConnectorKey: "tavily", Status: agentrun.InvocationNotInvoked,
			ErrorCode: "not_invoked", ErrorSummary: "Connector was not invoked because query planning failed",
		}, {
			ConnectorKey: "parallel_search", Status: agentrun.InvocationFailed,
			ErrorCode: "connector_failed", ErrorSummary: "Connector request failed",
		}},
	}
	paths, err := WriteTerminalAudit(root, execution)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := WriteTerminalAudit(root, execution)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, replayed) {
		t.Fatalf("replayed paths = %#v, want %#v", replayed, paths)
	}
	for _, key := range []string{"candidates", "summary", "manifest"} {
		if _, err := os.Stat(paths[key]); err != nil {
			t.Fatalf("missing %s: %v", key, err)
		}
	}
	manifest, err := os.ReadFile(paths["manifest"])
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	for _, want := range []string{
		`"stop_reason": "agent_or_tool_limit"`,
		`"error_code": "planning_failed"`,
		`"status": "not_invoked"`,
		`"connectors_failed": 1`,
		`"time_window_hours": 48`,
		`"window_start": "2026-07-20T08:00:00Z"`,
		`"window_end": "2026-07-22T08:00:00Z"`,
		`"results_pending": 0`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("manifest missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, execution.Prompt) {
		t.Fatalf("terminal audit leaked Prompt: %s", text)
	}
}

func TestAuditPollutionReportsHashesWithoutMutatingArtifacts(t *testing.T) {
	root := t.TempDir()
	documentPath := filepath.Join(root, "documents", "2026", "07", "22", "polluted.md")
	ledgerPath := filepath.Join(root, "runs", "run", "candidates.jsonl")
	summaryPath := filepath.Join(root, "runs", "run", "summary.md")
	manifestPath := filepath.Join(root, "runs", "run", "manifest.json")
	indexPath := filepath.Join(root, "indexes", "dedup-index.tsv")
	for _, path := range []string{documentPath, ledgerPath, summaryPath, manifestPath, indexPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	document := []byte("---\nsource_external_id: \"2.434241e+06\"\n---\n\n股 份 1. 8 0亿元\n")
	ledger := []byte("{\"url\":\"https://www.cls.cn/detail/2.434241e+06\"}\n")
	if err := os.WriteFile(documentPath, document, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledgerPath, ledger, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(summaryPath, []byte("# summary\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(indexHeader), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := AuditPollution(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Documents != 1 || report.Ledgers != 1 || report.IndexRows != 0 || len(report.Files) != 5 || len(report.Findings) != 2 {
		t.Fatalf("report = %#v", report)
	}
	for _, identity := range report.Files {
		if identity.Path == "" || len(identity.SHA256) != 64 || identity.Kind == "" {
			t.Fatalf("file identity = %#v", identity)
		}
	}
	if len(report.Findings[0].SHA256) != 64 || len(report.Findings[1].SHA256) != 64 {
		t.Fatalf("findings = %#v", report.Findings)
	}
	gotDocument, _ := os.ReadFile(documentPath)
	gotLedger, _ := os.ReadFile(ledgerPath)
	if string(gotDocument) != string(document) || string(gotLedger) != string(ledger) {
		t.Fatal("read-only pollution audit mutated an Artifact")
	}
}
