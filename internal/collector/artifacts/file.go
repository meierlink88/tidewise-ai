package artifacts

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/text/unicode/norm"
)

const indexHeader = "document_id\tpublished_at\turl_sha256\tcontent_sha256\tsimhash64\tdocument_path\n"

var tokenPattern = regexp.MustCompile(`[\pL\pN_]+`)
var excessNewlinePattern = regexp.MustCompile(`\n{3,}`)

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

	stats := collector.Stats{
		ConnectorCounts: make(map[string]int), ConnectorErrors: make(map[string]string),
		ContentLevels: map[collector.ContentLevel]int{
			collector.LevelFullText: 0, collector.LevelSummary: 0,
			collector.LevelSnippet: 0, collector.LevelTitle: 0,
		},
	}
	var candidates []collector.Candidate
	for _, name := range orderedRunNames(runs) {
		run := runs[name]
		stats.ConnectorCounts[name] = len(run.Results)
		if run.ErrorCode != "" {
			stats.ConnectorErrors[name] = run.ErrorCode + ": " + run.ErrorSummary
		}
		candidates = append(candidates, run.Results...)
	}
	stats.RawResults = len(candidates)
	merged := merge(candidates)
	stats.MergedResults = len(merged)
	stats.ResultsPending = len(merged)

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
	windowStart := request.CollectedAt.Add(-time.Duration(request.TimeWindowHours) * time.Hour)
	ledger := make([]candidateLedgerEntry, 0, len(merged))
	acceptedDocuments := make([]string, 0, len(merged))
	stagedDocuments := make(map[string]string, len(merged))
	for _, item := range merged {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		stats.ContentLevels[item.ContentLevel]++
		state, record, stagedPath, err := f.materializeOne(item, request.CollectedAt, windowStart, stageDocumentsRoot, records)
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
			stats.Accepted++
			record.Path = path
			records = append(records, record)
			acceptedDocuments = append(acceptedDocuments, path)
		case collector.DispositionKnownURL:
			stats.KnownURL++
		case collector.DispositionOutOfWindow:
			stats.OutOfWindow++
		case collector.DispositionInvalidResult:
			stats.InvalidResult++
		case collector.DispositionExactDuplicate:
			stats.ExactDuplicate++
		case collector.DispositionNearDuplicate:
			stats.NearDuplicate++
		}
		stats.ResultsTerminal++
		stats.ResultsPending--
		entry := candidateLedgerEntry{
			Title: strings.TrimSpace(item.Title), URL: canonicalURL(item.URL),
			Connectors: append([]string(nil), item.Connectors...), PrimaryConnector: item.PrimaryConnector,
			ContentLevel: string(item.ContentLevel), PublishedAtHint: item.PublishedAtHint,
			Disposition: state, Reason: dispositionReason(state),
		}
		if state == collector.DispositionAccepted {
			entry.ArtifactPath = path
			entry.ContentSHA256 = record.ContentHash
		}
		ledger = append(ledger, entry)
	}
	if stats.MergedResults != stats.ResultsTerminal+stats.ResultsPending {
		return nil, fmt.Errorf("Candidate conservation failed")
	}
	stopReason := "connectors_completed"
	if len(stats.ConnectorErrors) > 0 {
		stopReason = "completed_with_connector_failures"
	}
	result := &collector.Result{
		RunID: request.RunID, StopReason: stopReason,
		Documents: documentsRoot, AcceptedDocuments: acceptedDocuments, Index: indexPath,
		Candidates: filepath.Join(runRoot, "candidates.jsonl"),
		Manifest:   filepath.Join(runRoot, "manifest.json"),
		Summary:    filepath.Join(runRoot, "summary.md"), Stats: stats,
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
		terminalStatus(*result), completedAt,
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

func (f File) materializeOne(item collector.Candidate, collectedAt, windowStart time.Time, documentsRoot string, records []indexRecord) (collector.CandidateDisposition, indexRecord, string, error) {
	item.Title = strings.TrimSpace(item.Title)
	item.URL = canonicalURL(item.URL)
	item.Content = strings.TrimSpace(item.Content)
	if item.Content == "" {
		item.Content = item.Title
	}
	if item.Title == "" || item.URL == "" || !validLevel(item.ContentLevel) {
		return collector.DispositionInvalidResult, indexRecord{}, "", nil
	}
	published := parseTime(item.PublishedAtHint)
	if !published.IsZero() && (published.Before(windowStart) || published.After(collectedAt)) {
		return collector.DispositionOutOfWindow, indexRecord{}, "", nil
	}
	urlHash := hash(item.URL)
	for _, record := range records {
		if record.URLHash == urlHash {
			return collector.DispositionKnownURL, indexRecord{}, "", nil
		}
	}
	body := normalizeBody("# " + item.Title + "\n\n" + item.Content)
	contentHash := hash(body)
	fingerprint := simhash(body)
	for _, record := range records {
		if record.ContentHash == contentHash {
			return collector.DispositionExactDuplicate, indexRecord{}, "", nil
		}
		if f.NearDuplicateRadius > 0 && hamming(record.SimHash, fingerprint) <= f.NearDuplicateRadius {
			return collector.DispositionNearDuplicate, indexRecord{}, "", nil
		}
	}
	stamp := collectedAt
	publishedText := ""
	timeBasis := "collected_at"
	if !published.IsZero() {
		stamp = published
		publishedText = published.Format(time.RFC3339)
		timeBasis = "published_at"
	}
	documentID := "sha256:" + hash(item.URL+"\n"+contentHash)
	path := filepath.Join(documentsRoot, stamp.Format("2006/01/02"), fmt.Sprintf("%s--%s--%s.md", stamp.UTC().Format("20060102T150405Z"), slug(item.Title), urlHash[:8]))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", indexRecord{}, "", err
	}
	markdown := render(item, documentID, contentHash, publishedText, collectedAt.UTC().Format(time.RFC3339), timeBasis)
	if err := atomicWrite(path, []byte(markdown), 0o644); err != nil {
		return "", indexRecord{}, "", err
	}
	return collector.DispositionAccepted, indexRecord{documentID, publishedText, urlHash, contentHash, fingerprint, path}, path, nil
}

func dispositionReason(disposition collector.CandidateDisposition) string {
	switch disposition {
	case collector.DispositionAccepted:
		return "accepted"
	case collector.DispositionKnownURL:
		return "canonical_url_already_indexed"
	case collector.DispositionOutOfWindow:
		return "published_at_outside_time_window"
	case collector.DispositionInvalidResult:
		return "missing_title_or_url_or_content_level"
	case collector.DispositionExactDuplicate:
		return "content_sha256_already_indexed"
	case collector.DispositionNearDuplicate:
		return "simhash_within_radius"
	default:
		return "unknown_disposition"
	}
}

func merge(items []collector.Candidate) []collector.Candidate {
	groups := make(map[string][]collector.Candidate)
	var invalid int
	for _, item := range items {
		key := canonicalURL(item.URL)
		if key == "" {
			key = fmt.Sprintf("__invalid_%d", invalid)
			invalid++
		}
		groups[key] = append(groups[key], item)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	merged := make([]collector.Candidate, 0, len(keys))
	for _, key := range keys {
		group := groups[key]
		best := group[0]
		connectorSet := make(map[string]struct{})
		for _, item := range group {
			connectorSet[item.Connector] = struct{}{}
			if richer(item, best) {
				best = item
			}
		}
		best.URL = key
		best.PrimaryConnector = best.Connector
		for name := range connectorSet {
			best.Connectors = append(best.Connectors, name)
		}
		sort.Strings(best.Connectors)
		merged = append(merged, best)
	}
	return merged
}

func richer(candidate, current collector.Candidate) bool {
	if candidateRank, currentRank := rank(candidate.ContentLevel), rank(current.ContentLevel); candidateRank != currentRank {
		return candidateRank > currentRank
	}
	if candidateLength, currentLength := nonWhitespaceRunes(candidate.Content), nonWhitespaceRunes(current.Content); candidateLength != currentLength {
		return candidateLength > currentLength
	}
	return stableCandidateKey(candidate) < stableCandidateKey(current)
}

func nonWhitespaceRunes(value string) int {
	count := 0
	for _, character := range value {
		if !unicode.IsSpace(character) {
			count++
		}
	}
	return count
}

func stableCandidateKey(item collector.Candidate) string {
	connectorOrder := len(collector.ConnectorKeys())
	for index, connector := range collector.ConnectorKeys() {
		if item.Connector == connector {
			connectorOrder = index
			break
		}
	}
	return fmt.Sprintf("%03d\x00%09d\x00%s\x00%s\x00%s", connectorOrder, item.ResultPosition, item.SourceExternalID, item.Content, item.Title)
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
	windowEnd := request.CollectedAt.UTC()
	windowStart := windowEnd.Add(-time.Duration(request.TimeWindowHours) * time.Hour)
	payload := manifestPayload{
		Schema: "collector_artifact_manifest.v1", ExecutionID: request.RunID,
		ExecutionStatus: string(terminalStatus(*result)),
		AgentKey:        "collector", AgentVersion: "collector.v1",
		PromptSHA256: hex.EncodeToString(promptHash[:]), PromptBytes: len([]byte(request.Prompt)),
		StartedAt: request.CollectedAt.UTC().Format(time.RFC3339), CompletedAt: completedAt.UTC().Format(time.RFC3339Nano),
		TimeWindowHours: request.TimeWindowHours,
		WindowStart:     windowStart.Format(time.RFC3339), WindowEnd: windowEnd.Format(time.RFC3339),
		StopReason: result.StopReason, ConnectorsAttempted: len(runs), ConnectorCounts: result.Stats.ConnectorCounts,
		CandidateCounts: statsMap(result.Stats), ResultsPending: result.Stats.ResultsPending,
		ContentLevels: contentLevelMap(result.Stats.ContentLevels),
		Artifacts:     map[string]string{"documents": result.Documents, "candidates": result.Candidates, "summary": result.Summary, "index": result.Index, "manifest": result.Manifest},
	}
	connectorNames := orderedRunNames(runs)
	for _, connector := range connectorNames {
		run := runs[connector]
		outcome := connectorOutcome{Connector: connector, ResultCount: len(run.Results)}
		if run.ErrorCode == "" {
			outcome.Status = "completed"
			payload.ConnectorsCompleted++
		} else {
			outcome.Status = "failed"
			payload.ConnectorsFailed++
			outcome.ErrorCode = run.ErrorCode
			outcome.ErrorSummary = run.ErrorSummary
			payload.ConnectorFailures = append(payload.ConnectorFailures, connectorFailure{
				Connector: connector, Code: run.ErrorCode, Summary: run.ErrorSummary,
			})
		}
		payload.ConnectorOutcomes = append(payload.ConnectorOutcomes, outcome)
	}
	sort.Slice(payload.ConnectorFailures, func(left, right int) bool {
		return payload.ConnectorFailures[left].Connector < payload.ConnectorFailures[right].Connector
	})
	if len(payload.ConnectorFailures) == len(collector.ConnectorKeys()) {
		payload.ErrorCode = "all_connectors_failed"
		payload.ErrorSummary = "All Connector invocations failed"
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

func statsMap(stats collector.Stats) map[string]int {
	return map[string]int{
		"raw_results": stats.RawResults, "merged_results": stats.MergedResults,
		"results_terminal": stats.ResultsTerminal, "results_pending": stats.ResultsPending,
		"accepted":  stats.Accepted,
		"known_url": stats.KnownURL, "out_of_window": stats.OutOfWindow,
		"invalid_result": stats.InvalidResult, "exact_duplicate": stats.ExactDuplicate,
		"near_duplicate": stats.NearDuplicate,
	}
}

func contentLevelMap(levels map[collector.ContentLevel]int) map[string]int {
	return map[string]int{
		string(collector.LevelFullText): levels[collector.LevelFullText],
		string(collector.LevelSummary):  levels[collector.LevelSummary],
		string(collector.LevelSnippet):  levels[collector.LevelSnippet],
		string(collector.LevelTitle):    levels[collector.LevelTitle],
	}
}

func connectorPosition(name string) int {
	for position, key := range collector.ConnectorKeys() {
		if key == name {
			return position
		}
	}
	return len(collector.ConnectorKeys())
}

func orderedRunNames(runs map[string]collector.ConnectorRun) []string {
	names := make([]string, 0, len(runs))
	for name := range runs {
		names = append(names, name)
	}
	sort.Slice(names, func(left, right int) bool {
		leftPosition := connectorPosition(names[left])
		rightPosition := connectorPosition(names[right])
		if leftPosition != rightPosition {
			return leftPosition < rightPosition
		}
		return names[left] < names[right]
	})
	return names
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
		if record.PublishedAt != "" && parseTime(record.PublishedAt).IsZero() {
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

func canonicalURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	hostname := strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port == "" || parsed.Scheme == "https" && port == "443" || parsed.Scheme == "http" && port == "80" {
		if strings.Contains(hostname, ":") {
			parsed.Host = "[" + hostname + "]"
		} else {
			parsed.Host = hostname
		}
	} else {
		parsed.Host = net.JoinHostPort(hostname, port)
	}
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "ref" || lower == "mc_cid" || lower == "mc_eid" {
			query.Del(key)
			continue
		}
		sort.Strings(query[key])
	}
	parsed.RawQuery = query.Encode()
	if parsed.Path == "" {
		parsed.Path = "/"
	} else if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
		if parsed.Path == "" {
			parsed.Path = "/"
		}
	}
	return parsed.String()
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func simhash(value string) string {
	weights := make([]int, 64)
	counts := make(map[string]int)
	for _, token := range tokenPattern.FindAllString(strings.ToLower(norm.NFKC.String(value)), -1) {
		counts[token]++
	}
	if len(counts) == 0 {
		return "0000000000000000"
	}
	for token, count := range counts {
		digest, _ := blake2b.New(8, nil)
		_, _ = digest.Write([]byte(token))
		bits := binary.BigEndian.Uint64(digest.Sum(nil))
		for bit := range 64 {
			if bits&(uint64(1)<<bit) != 0 {
				weights[bit] += count
			} else {
				weights[bit] -= count
			}
		}
	}
	var result uint64
	for bit, weight := range weights {
		if weight >= 0 {
			result |= uint64(1) << bit
		}
	}
	return fmt.Sprintf("%016x", result)
}

func normalizeBody(value string) string {
	value = norm.NFC.String(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n"))
	lines := strings.Split(value, "\n")
	for index := range lines {
		lines[index] = strings.TrimRightFunc(lines[index], unicode.IsSpace)
	}
	return excessNewlinePattern.ReplaceAllString(strings.TrimSpace(strings.Join(lines, "\n")), "\n\n")
}

func hamming(left, right string) int {
	a, errA := strconv.ParseUint(left, 16, 64)
	b, errB := strconv.ParseUint(right, 16, 64)
	if errA != nil || errB != nil {
		return 65
	}
	value := a ^ b
	count := 0
	for value != 0 {
		value &= value - 1
		count++
	}
	return count
}

func parseTime(value string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	} {
		parsed, err := time.Parse(layout, strings.TrimSpace(value))
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func rank(level collector.ContentLevel) int {
	switch level {
	case collector.LevelFullText:
		return 3
	case collector.LevelSummary:
		return 2
	case collector.LevelSnippet:
		return 1
	case collector.LevelTitle:
		return 0
	default:
		return -1
	}
}

func validLevel(level collector.ContentLevel) bool { return rank(level) >= 0 }

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
