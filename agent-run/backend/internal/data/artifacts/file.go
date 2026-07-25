package artifacts

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector/materialization"
)

const indexHeader = "document_id\tpublished_at\turl_sha256\tcontent_sha256\tsimhash64\tdocument_path\n"

type File struct {
	Root                string
	NearDuplicateRadius int
	Publications        PublicationRepository
	BeforePublish       func(string) error
	Now                 func() time.Time
}

type indexRecord struct {
	DocumentID, PublishedAt, URLHash, ContentHash, SimHash, Path string
}

type candidateLedgerEntry struct {
	Title            string                         `json:"title"`
	URL              string                         `json:"url"`
	Connectors       []string                       `json:"connectors"`
	PrimaryConnector string                         `json:"primary_connector"`
	ContentLevel     string                         `json:"content_level"`
	PublishedAtHint  string                         `json:"published_at_hint,omitempty"`
	Disposition      collector.CandidateDisposition `json:"disposition"`
	Reason           string                         `json:"reason"`
	ArtifactPath     string                         `json:"artifact_path,omitempty"`
	ContentSHA256    string                         `json:"content_sha256,omitempty"`
}

type connectorFailure struct {
	Connector string `json:"connector"`
	Code      string `json:"code"`
	Summary   string `json:"summary"`
}

type connectorOutcome struct {
	Connector    string `json:"connector"`
	Status       string `json:"status"`
	ResultCount  int    `json:"result_count"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorSummary string `json:"error_summary,omitempty"`
}

type manifestPayload struct {
	Schema              string              `json:"schema"`
	ExecutionID         string              `json:"execution_id"`
	ExecutionStatus     string              `json:"execution_status"`
	AgentKey            string              `json:"agent_key"`
	AgentVersion        string              `json:"agent_version"`
	PromptSHA256        string              `json:"prompt_sha256"`
	PromptBytes         int                 `json:"prompt_bytes"`
	StartedAt           string              `json:"started_at"`
	CompletedAt         string              `json:"completed_at"`
	TimeWindowHours     int                 `json:"time_window_hours"`
	WindowStart         string              `json:"window_start"`
	WindowEnd           string              `json:"window_end"`
	StopReason          string              `json:"stop_reason"`
	ErrorCode           string              `json:"error_code,omitempty"`
	ErrorSummary        string              `json:"error_summary,omitempty"`
	ConnectorsAttempted int                 `json:"connectors_attempted"`
	ConnectorsCompleted int                 `json:"connectors_completed"`
	ConnectorsFailed    int                 `json:"connectors_failed"`
	ConnectorOutcomes   []connectorOutcome  `json:"connector_outcomes"`
	ConnectorCounts     map[string]int      `json:"connector_counts"`
	ConnectorFailures   []connectorFailure  `json:"connector_failures"`
	CandidateCounts     map[string]int      `json:"candidate_counts"`
	ContentLevels       map[string]int      `json:"content_levels"`
	ResultsPending      int                 `json:"results_pending"`
	Artifacts           map[string]string   `json:"artifacts"`
	Accepted            []map[string]string `json:"accepted"`
}

func (f File) Materialize(ctx context.Context, request collector.Request, runs map[string]collector.ConnectorRun) (*collector.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := f.Root
	if root == "" {
		root = "data"
	}
	documentsRoot := filepath.Join(root, "documents")
	indexPath := filepath.Join(root, "indexes", "dedup-index.tsv")
	runRoot := filepath.Join(root, "runs", request.RunID)
	if _, err := os.Stat(runRoot); err == nil {
		return nil, fmt.Errorf("run Artifact directory already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	stagingParent := filepath.Join(root, ".staging")
	if err := os.MkdirAll(stagingParent, 0o755); err != nil {
		return nil, err
	}
	var stageRoot string
	var err error
	if f.Publications == nil {
		stageRoot, err = os.MkdirTemp(stagingParent, request.RunID+"-")
		if err != nil {
			return nil, err
		}
	} else {
		stageRoot = filepath.Join(root, ".pending", request.RunID)
		if _, err := os.Stat(stageRoot); err == nil {
			return nil, fmt.Errorf("Artifact publication is already pending")
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		if err := os.MkdirAll(stageRoot, 0o755); err != nil {
			return nil, err
		}
	}
	publicationPrepared := false
	defer func() {
		if f.Publications == nil || !publicationPrepared {
			_ = os.RemoveAll(stageRoot)
		}
	}()
	stageDocumentsRoot := filepath.Join(stageRoot, "documents")
	stageRunRoot := filepath.Join(stageRoot, "run")
	stageIndexPath := filepath.Join(stageRoot, "dedup-index.tsv")
	if err := os.MkdirAll(stageRunRoot, 0o755); err != nil {
		return nil, err
	}

	previousIndexSHA256 := ""
	if digest, hashErr := fileSHA256(indexPath); hashErr == nil {
		previousIndexSHA256 = digest
	} else if !os.IsNotExist(hashErr) {
		return nil, hashErr
	}
	records, err := loadIndex(indexPath)
	if errors.Is(err, os.ErrNotExist) {
		records, err = collectDocumentRecords(documentsRoot)
	}
	if err != nil {
		return nil, err
	}
	existing := make([]materialization.ExistingRecord, 0, len(records))
	for _, record := range records {
		existing = append(existing, materialization.ExistingRecord{
			URLHash: record.URLHash, ContentHash: record.ContentHash, SimHash: record.SimHash,
		})
	}
	processor := materialization.NewProcessor(runs, existing, f.NearDuplicateRadius, request)
	ledger := make([]candidateLedgerEntry, 0, len(processor.Merged))
	acceptedDocuments := make([]string, 0, len(processor.Merged))
	stagedDocuments := make(map[string]string, len(processor.Merged))
	for _, item := range processor.Merged {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		decision, err := processor.Decide(item)
		if err != nil {
			return nil, err
		}
		state := decision.Disposition
		record, stagedPath, err := writeAccepted(decision, request.CollectedAt, stageDocumentsRoot)
		if err != nil {
			return nil, err
		}
		path := ""
		if state == collector.DispositionAccepted {
			relative, err := filepath.Rel(stageDocumentsRoot, stagedPath)
			if err != nil {
				return nil, err
			}
			path = filepath.Join(documentsRoot, relative)
			if _, err := os.Stat(path); err == nil {
				return nil, fmt.Errorf("accepted document path already exists")
			} else if !os.IsNotExist(err) {
				return nil, err
			}
			stagedDocuments[path] = stagedPath
		}
		switch state {
		case collector.DispositionAccepted:
			record.Path = path
			records = append(records, record)
			acceptedDocuments = append(acceptedDocuments, path)
		}
		entry := candidateLedgerEntry{
			Title: strings.TrimSpace(decision.Candidate.Title), URL: decision.Candidate.URL,
			Connectors: append([]string(nil), decision.Candidate.Connectors...), PrimaryConnector: decision.Candidate.PrimaryConnector,
			ContentLevel: string(decision.Candidate.ContentLevel), PublishedAtHint: decision.Candidate.PublishedAtHint,
			Disposition: state, Reason: materialization.Reason(state),
		}
		if state == collector.DispositionAccepted {
			entry.ArtifactPath = path
			entry.ContentSHA256 = record.ContentHash
		}
		ledger = append(ledger, entry)
	}
	stopReason, err := processor.Finish()
	if err != nil {
		return nil, err
	}
	result := &collector.Result{
		RunID: request.RunID, StopReason: stopReason,
		Documents: documentsRoot, AcceptedDocuments: acceptedDocuments, Index: indexPath,
		Candidates: filepath.Join(runRoot, "candidates.jsonl"),
		Manifest:   filepath.Join(runRoot, "manifest.json"),
		Summary:    filepath.Join(runRoot, "summary.md"), Stats: processor.Stats,
	}
	stageResult := *result
	stageResult.Candidates = filepath.Join(stageRunRoot, "candidates.jsonl")
	stageResult.Summary = filepath.Join(stageRunRoot, "summary.md")
	stageResult.Manifest = filepath.Join(stageRunRoot, "manifest.json")
	stageResult.Index = stageIndexPath
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := writeIndex(stageIndexPath, records); err != nil {
		return nil, err
	}
	if err := writeCandidateLedger(stageResult.Candidates, ledger); err != nil {
		return nil, err
	}
	completedAt := f.now()
	payload, err := buildManifest(result, request, runs, stagedDocuments, completedAt)
	if err != nil {
		return nil, err
	}
	if err := writeSummary(stageResult.Summary, payload); err != nil {
		return nil, err
	}
	if err := writeManifest(stageResult.Manifest, payload); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.Publications == nil {
		if err := publishArtifacts(ctx, stageResult, *result, stagedDocuments, indexPath, runRoot); err != nil {
			return nil, err
		}
		return result, nil
	}
	reference, plan, err := preparePublicationPlan(
		root, *result, stageResult, stagedDocuments, previousIndexSHA256,
		materialization.TerminalStatus(*result), completedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := retryPublicationCall(ctx, func() error {
		return f.Publications.PreparePublication(ctx, reference)
	}); err != nil {
		checkContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		references, checkErr := f.Publications.ListPreparedPublications(checkContext)
		cancel()
		for _, prepared := range references {
			if prepared.ExecutionID == reference.ExecutionID &&
				prepared.PlanPath == reference.PlanPath &&
				prepared.PlanSHA256 == reference.PlanSHA256 {
				publicationPrepared = true
				return nil, fmt.Errorf("%w: prepare result was unknown", ErrPublicationPending)
			}
		}
		if checkErr != nil {
			publicationPrepared = true
			return nil, fmt.Errorf("%w: prepare result could not be confirmed", ErrPublicationPending)
		}
		return nil, fmt.Errorf("prepare Artifact publication: %w", err)
	}
	publicationPrepared = true
	if err := publishPreparedPlan(ctx, root, plan, f.BeforePublish); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPublicationPending, err)
	}
	completion, err := planCompletion(plan)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPublicationPending, err)
	}
	if err := retryPublicationCall(ctx, func() error {
		return f.Publications.CommitPreparedPublication(ctx, reference, completion)
	}); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPublicationPending, err)
	}
	_ = os.RemoveAll(stageRoot)
	return result, nil
}

func (f File) now() time.Time {
	if f.Now != nil {
		return f.Now().UTC()
	}
	return time.Now().UTC()
}

func writeAccepted(decision materialization.Decision, collectedAt time.Time, documentsRoot string) (indexRecord, string, error) {
	if decision.Disposition != collector.DispositionAccepted {
		return indexRecord{}, "", nil
	}
	item := decision.Candidate
	path := filepath.Join(
		documentsRoot,
		decision.PublishedAt.Format("2006/01/02"),
		fmt.Sprintf("%s--%s--%s.md",
			decision.PublishedAt.UTC().Format("20060102T150405Z"),
			slug(item.Title),
			decision.URLHash[:8],
		),
	)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return indexRecord{}, "", err
	}
	markdown := render(
		item,
		decision.DocumentID,
		decision.ContentHash,
		decision.PublishedText,
		collectedAt.UTC().Format(time.RFC3339),
		decision.TimeBasis,
	)
	if err := atomicWrite(path, []byte(markdown), 0o644); err != nil {
		return indexRecord{}, "", err
	}
	return indexRecord{
		decision.DocumentID,
		decision.PublishedText,
		decision.URLHash,
		decision.ContentHash,
		decision.SimHash,
		path,
	}, path, nil
}

func render(item collector.Candidate, documentID, contentHash, published, collected, timeBasis string) string {
	publishedYAML := "null"
	if published != "" {
		publishedYAML = strconv.Quote(published)
	}
	var connectorLines strings.Builder
	for _, name := range item.Connectors {
		fmt.Fprintf(&connectorLines, "  - %s\n", strconv.Quote(name))
	}
	return fmt.Sprintf("---\nschema_version: \"connector_result_md.v1\"\ndocument_id: %s\ntitle: %s\nsource_name: %s\nsource_type: %s\nsource_url: %s\nsource_external_id: %s\nconnectors:\n%sprimary_connector: %s\ncontent_origin: \"connector_response\"\ncontent_level: %s\npublished_at: %s\ncollected_at: %s\ntime_basis: %s\nlanguage: %s\ncontent_sha256: %s\nquality_status: \"accepted\"\nsupersedes_document_id: null\n---\n\n# %s\n\n%s\n", strconv.Quote(documentID), strconv.Quote(item.Title), yamlNullable(item.SourceName), yamlNullable(item.SourceType), strconv.Quote(item.URL), yamlNullable(item.SourceExternalID), connectorLines.String(), strconv.Quote(item.PrimaryConnector), strconv.Quote(string(item.ContentLevel)), publishedYAML, strconv.Quote(collected), strconv.Quote(timeBasis), strconv.Quote(language(item.Title+item.Content)), strconv.Quote(contentHash), item.Title, item.Content)
}

func writeSummary(path string, manifest manifestPayload) error {
	counts := manifest.CandidateCounts
	content := fmt.Sprintf("# Collector Run %s\n\n- execution_status: %s\n- time_window_hours: %d\n- window_start: %s\n- window_end: %s\n- stop_reason: %s\n- connectors_attempted: %d\n- connectors_completed: %d\n- connectors_failed: %d\n- raw_results: %d\n- merged_results: %d\n- results_terminal: %d\n- results_pending: %d\n- accepted: %d\n- known_url: %d\n- out_of_window: %d\n- invalid_result: %d\n- exact_duplicate: %d\n- near_duplicate: %d\n- full_text: %d\n- summary: %d\n- snippet: %d\n- title_only: %d\n- documents: %s\n- index: %s\n- candidates: %s\n- summary_artifact: %s\n- manifest: %s\n",
		manifest.ExecutionID, manifest.ExecutionStatus, manifest.TimeWindowHours, manifest.WindowStart, manifest.WindowEnd,
		manifest.StopReason, manifest.ConnectorsAttempted, manifest.ConnectorsCompleted, manifest.ConnectorsFailed,
		counts["raw_results"], counts["merged_results"], counts["results_terminal"], counts["results_pending"],
		counts["accepted"], counts["known_url"], counts["out_of_window"], counts["invalid_result"],
		counts["exact_duplicate"], counts["near_duplicate"], manifest.ContentLevels[string(collector.LevelFullText)],
		manifest.ContentLevels[string(collector.LevelSummary)], manifest.ContentLevels[string(collector.LevelSnippet)],
		manifest.ContentLevels[string(collector.LevelTitle)], manifest.Artifacts["documents"], manifest.Artifacts["index"],
		manifest.Artifacts["candidates"], manifest.Artifacts["summary"], manifest.Artifacts["manifest"])
	if len(manifest.ConnectorOutcomes) > 0 {
		content += "- connector_outcomes:\n"
		for _, outcome := range manifest.ConnectorOutcomes {
			content += fmt.Sprintf("  - %s: %s (%d)", outcome.Connector, outcome.Status, outcome.ResultCount)
			if outcome.ErrorCode != "" {
				content += fmt.Sprintf(", %s: %s", outcome.ErrorCode, outcome.ErrorSummary)
			}
			content += "\n"
		}
	}
	if len(manifest.Accepted) > 0 {
		content += "- accepted_artifacts:\n"
		for _, accepted := range manifest.Accepted {
			content += fmt.Sprintf("  - %s sha256:%s\n", accepted["path"], accepted["sha256"])
		}
	}
	return atomicWrite(path, []byte(content), 0o644)
}

func writeCandidateLedger(path string, entries []candidateLedgerEntry) error {
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetEscapeHTML(false)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("encode Candidate ledger: %w", err)
		}
	}
	return atomicWrite(path, []byte(builder.String()), 0o644)
}

func buildManifest(result *collector.Result, request collector.Request, runs map[string]collector.ConnectorRun, stagedDocuments map[string]string, completedAt time.Time) (manifestPayload, error) {
	promptHash := sha256.Sum256([]byte(request.Prompt))
	window := materialization.WindowFor(request)
	audit := materialization.BuildRunAudit(*result, runs)
	payload := manifestPayload{
		Schema: "collector_artifact_manifest.v1", ExecutionID: request.RunID,
		ExecutionStatus: string(audit.ExecutionStatus),
		AgentKey:        collector.AgentKey, AgentVersion: collector.AgentVersion,
		PromptSHA256: hex.EncodeToString(promptHash[:]), PromptBytes: len([]byte(request.Prompt)),
		StartedAt: request.CollectedAt.UTC().Format(time.RFC3339), CompletedAt: completedAt.UTC().Format(time.RFC3339Nano),
		TimeWindowHours: window.Hours,
		WindowStart:     window.Start.Format(time.RFC3339), WindowEnd: window.End.Format(time.RFC3339),
		StopReason: result.StopReason, ErrorCode: audit.ErrorCode, ErrorSummary: audit.ErrorSummary,
		ConnectorsAttempted: audit.ConnectorsAttempted, ConnectorsCompleted: audit.ConnectorsCompleted,
		ConnectorsFailed: audit.ConnectorsFailed, ConnectorCounts: result.Stats.ConnectorCounts,
		CandidateCounts: audit.CandidateCounts, ResultsPending: result.Stats.ResultsPending,
		ContentLevels: audit.ContentLevels,
		Artifacts:     map[string]string{"documents": result.Documents, "candidates": result.Candidates, "summary": result.Summary, "index": result.Index, "manifest": result.Manifest},
	}
	for _, item := range audit.ConnectorOutcomes {
		payload.ConnectorOutcomes = append(payload.ConnectorOutcomes, connectorOutcome{
			Connector: item.Connector, Status: item.Status, ResultCount: item.ResultCount,
			ErrorCode: item.ErrorCode, ErrorSummary: item.ErrorSummary,
		})
		if item.ErrorCode != "" {
			payload.ConnectorFailures = append(payload.ConnectorFailures, connectorFailure{
				Connector: item.Connector, Code: item.ErrorCode, Summary: item.ErrorSummary,
			})
		}
	}
	for _, path := range result.AcceptedDocuments {
		content, err := os.ReadFile(stagedDocuments[path])
		if err != nil {
			return manifestPayload{}, fmt.Errorf("hash accepted Artifact: %w", err)
		}
		sum := sha256.Sum256(content)
		payload.Accepted = append(payload.Accepted, map[string]string{"path": path, "sha256": hex.EncodeToString(sum[:])})
	}
	return payload, nil
}

func writeManifest(path string, payload manifestPayload) error {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	encoded = append(encoded, '\n')
	return atomicWrite(path, encoded, 0o644)
}

func publishArtifacts(ctx context.Context, staged, final collector.Result, stagedDocuments map[string]string, indexPath, runRoot string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	previousIndex, indexErr := os.ReadFile(indexPath)
	indexExisted := indexErr == nil
	if indexErr != nil && !os.IsNotExist(indexErr) {
		return indexErr
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(runRoot, 0o755); err != nil {
		return err
	}
	published := make([]string, 0, len(stagedDocuments)+3)
	indexPublished := false
	rollback := func(cause error) error {
		for _, path := range published {
			_ = os.Remove(path)
		}
		_ = os.RemoveAll(runRoot)
		if indexPublished {
			if indexExisted {
				if err := atomicWrite(indexPath, previousIndex, 0o644); err != nil {
					return fmt.Errorf("%w; restore dedup index: %v", cause, err)
				}
			} else {
				_ = os.Remove(indexPath)
			}
		}
		return cause
	}

	documentPaths := make([]string, 0, len(stagedDocuments))
	for path := range stagedDocuments {
		documentPaths = append(documentPaths, path)
	}
	sort.Strings(documentPaths)
	for _, path := range documentPaths {
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return rollback(err)
		}
		if err := os.Rename(stagedDocuments[path], path); err != nil {
			return rollback(err)
		}
		published = append(published, path)
	}
	for _, pair := range [][2]string{
		{staged.Candidates, final.Candidates},
		{staged.Summary, final.Summary},
	} {
		if err := ctx.Err(); err != nil {
			return rollback(err)
		}
		if err := os.Rename(pair[0], pair[1]); err != nil {
			return rollback(err)
		}
		published = append(published, pair[1])
	}
	if err := ctx.Err(); err != nil {
		return rollback(err)
	}
	if err := os.Rename(staged.Index, indexPath); err != nil {
		return rollback(err)
	}
	indexPublished = true
	if err := ctx.Err(); err != nil {
		return rollback(err)
	}
	if err := os.Rename(staged.Manifest, final.Manifest); err != nil {
		return rollback(err)
	}
	return nil
}

func loadIndex(path string) ([]indexRecord, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []indexRecord
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read dedup index header: %w", err)
		}
		return nil, fmt.Errorf("invalid dedup index: missing header")
	}
	if scanner.Text()+"\n" != indexHeader {
		return nil, fmt.Errorf("invalid dedup index: unexpected header")
	}
	documentIDs := make(map[string]struct{})
	paths := make(map[string]struct{})
	lineNumber := 1
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 6 {
			return nil, fmt.Errorf("invalid dedup index row %d: expected 6 columns", lineNumber)
		}
		record := indexRecord{parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]}
		if !strings.HasPrefix(record.DocumentID, "sha256:") || !validHex(record.DocumentID[len("sha256:"):], sha256.Size*2) {
			return nil, fmt.Errorf("invalid dedup index row %d: document ID", lineNumber)
		}
		if record.PublishedAt != "" && materialization.ParseTime(record.PublishedAt).IsZero() {
			return nil, fmt.Errorf("invalid dedup index row %d: published_at", lineNumber)
		}
		if !validHex(record.URLHash, sha256.Size*2) {
			return nil, fmt.Errorf("invalid dedup index row %d: URL hash", lineNumber)
		}
		if !validHex(record.ContentHash, sha256.Size*2) {
			return nil, fmt.Errorf("invalid dedup index row %d: content hash", lineNumber)
		}
		if !validHex(record.SimHash, 16) {
			return nil, fmt.Errorf("invalid dedup index row %d: SimHash", lineNumber)
		}
		if strings.TrimSpace(record.Path) == "" {
			return nil, fmt.Errorf("invalid dedup index row %d: document path", lineNumber)
		}
		if information, err := os.Stat(record.Path); err != nil || !information.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid dedup index row %d: document path is not resolvable", lineNumber)
		}
		if _, exists := documentIDs[record.DocumentID]; exists {
			return nil, fmt.Errorf("invalid dedup index row %d: duplicate document ID", lineNumber)
		}
		cleanPath := filepath.Clean(record.Path)
		if _, exists := paths[cleanPath]; exists {
			return nil, fmt.Errorf("invalid dedup index row %d: duplicate document path", lineNumber)
		}
		documentIDs[record.DocumentID] = struct{}{}
		paths[cleanPath] = struct{}{}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dedup index: %w", err)
	}
	return records, nil
}

func validHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func writeIndex(path string, records []indexRecord) error {
	var builder strings.Builder
	builder.WriteString(indexHeader)
	for _, record := range records {
		fmt.Fprintf(&builder, "%s\t%s\t%s\t%s\t%s\t%s\n", record.DocumentID, record.PublishedAt, record.URLHash, record.ContentHash, record.SimHash, record.Path)
	}
	return atomicWrite(path, []byte(builder.String()), 0o644)
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func slug(value string) string {
	var output []rune
	dash := false
	for _, char := range strings.ToLower(value) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			output = append(output, char)
			dash = false
		} else if !dash && len(output) > 0 {
			output = append(output, '-')
			dash = true
		}
		if len(output) >= 64 {
			break
		}
	}
	result := strings.Trim(string(output), "-")
	if result == "" {
		return "connector-result"
	}
	return result
}

func yamlNullable(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	return strconv.Quote(value)
}

func language(value string) string {
	for _, char := range value {
		if unicode.In(char, unicode.Han) {
			return "zh"
		}
	}
	return "en"
}
