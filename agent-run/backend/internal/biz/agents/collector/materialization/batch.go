package materialization

import (
	"fmt"
	"sort"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

const DefaultNearDuplicateRadius = 3

type Batch struct {
	Merged []collector.Candidate
	Stats  collector.Stats
}

type Processor struct {
	Batch
	records             []ExistingRecord
	nearDuplicateRadius int
	window              CollectionWindow
}

type CollectionWindow struct {
	Start time.Time
	End   time.Time
	Hours int
}

type ConnectorOutcome struct {
	Connector    string
	Status       string
	ResultCount  int
	ErrorCode    string
	ErrorSummary string
}

type RunAudit struct {
	ExecutionStatus     agentrun.ExecutionStatus
	ErrorCode           string
	ErrorSummary        string
	ConnectorsAttempted int
	ConnectorsCompleted int
	ConnectorsFailed    int
	ConnectorOutcomes   []ConnectorOutcome
	CandidateCounts     map[string]int
	ContentLevels       map[string]int
}

func NewBatch(runs map[string]collector.ConnectorRun) Batch {
	stats := collector.Stats{
		ConnectorCounts: make(map[string]int),
		ConnectorErrors: make(map[string]string),
		ContentLevels: map[collector.ContentLevel]int{
			collector.LevelFullText: 0,
			collector.LevelSummary:  0,
			collector.LevelSnippet:  0,
			collector.LevelTitle:    0,
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
	merged := Merge(candidates)
	stats.MergedResults = len(merged)
	stats.ResultsPending = len(merged)
	return Batch{Merged: merged, Stats: stats}
}

func NewProcessor(
	runs map[string]collector.ConnectorRun,
	records []ExistingRecord,
	nearDuplicateRadius int,
	request collector.Request,
) *Processor {
	return &Processor{
		Batch:               NewBatch(runs),
		records:             append([]ExistingRecord(nil), records...),
		nearDuplicateRadius: nearDuplicateRadius,
		window:              WindowFor(request),
	}
}

func WindowFor(request collector.Request) CollectionWindow {
	end := request.CollectedAt.UTC()
	return CollectionWindow{
		Start: end.Add(-time.Duration(request.TimeWindowHours) * time.Hour),
		End:   end,
		Hours: request.TimeWindowHours,
	}
}

func (p *Processor) Decide(item collector.Candidate) (Decision, error) {
	decision := Evaluate(item, p.window.End, p.window.Start, p.records, p.nearDuplicateRadius)
	if err := p.Record(item.ContentLevel, decision.Disposition); err != nil {
		return Decision{}, err
	}
	if decision.Disposition == collector.DispositionAccepted {
		p.records = append(p.records, ExistingRecord{
			URLHash: decision.URLHash, ContentHash: decision.ContentHash, SimHash: decision.SimHash,
		})
	}
	return decision, nil
}

func (b *Batch) Record(level collector.ContentLevel, disposition collector.CandidateDisposition) error {
	if b.Stats.ResultsPending <= 0 {
		return fmt.Errorf("Candidate conservation failed")
	}
	b.Stats.ContentLevels[level]++
	switch disposition {
	case collector.DispositionAccepted:
		b.Stats.Accepted++
	case collector.DispositionKnownURL:
		b.Stats.KnownURL++
	case collector.DispositionOutOfWindow:
		b.Stats.OutOfWindow++
	case collector.DispositionInvalidResult:
		b.Stats.InvalidResult++
	case collector.DispositionExactDuplicate:
		b.Stats.ExactDuplicate++
	case collector.DispositionNearDuplicate:
		b.Stats.NearDuplicate++
	default:
		return fmt.Errorf("Candidate disposition is invalid")
	}
	b.Stats.ResultsTerminal++
	b.Stats.ResultsPending--
	return nil
}

func (b Batch) Finish() (string, error) {
	if b.Stats.MergedResults != b.Stats.ResultsTerminal+b.Stats.ResultsPending ||
		b.Stats.ResultsPending != 0 {
		return "", fmt.Errorf("Candidate conservation failed")
	}
	if len(b.Stats.ConnectorErrors) > 0 {
		return "completed_with_connector_failures", nil
	}
	return "connectors_completed", nil
}

func TerminalStatus(result collector.Result) agentrun.ExecutionStatus {
	if len(result.Stats.ConnectorErrors) == len(collector.ConnectorKeys()) {
		return agentrun.StatusFailed
	}
	if len(result.Stats.ConnectorErrors) > 0 {
		return agentrun.StatusPartiallySucceeded
	}
	if result.Stats.Accepted == 0 {
		return agentrun.StatusSucceededNoChange
	}
	return agentrun.StatusSucceeded
}

func BuildRunAudit(result collector.Result, runs map[string]collector.ConnectorRun) RunAudit {
	audit := RunAudit{
		ExecutionStatus:     TerminalStatus(result),
		ConnectorsAttempted: len(runs),
		CandidateCounts:     CandidateCounts(result.Stats),
		ContentLevels: map[string]int{
			string(collector.LevelFullText): result.Stats.ContentLevels[collector.LevelFullText],
			string(collector.LevelSummary):  result.Stats.ContentLevels[collector.LevelSummary],
			string(collector.LevelSnippet):  result.Stats.ContentLevels[collector.LevelSnippet],
			string(collector.LevelTitle):    result.Stats.ContentLevels[collector.LevelTitle],
		},
	}
	for _, connector := range orderedRunNames(runs) {
		run := runs[connector]
		outcome := ConnectorOutcome{Connector: connector, ResultCount: len(run.Results)}
		if run.ErrorCode == "" {
			outcome.Status = "completed"
			audit.ConnectorsCompleted++
		} else {
			outcome.Status = "failed"
			outcome.ErrorCode = run.ErrorCode
			outcome.ErrorSummary = run.ErrorSummary
			audit.ConnectorsFailed++
		}
		audit.ConnectorOutcomes = append(audit.ConnectorOutcomes, outcome)
	}
	if audit.ConnectorsFailed == len(collector.ConnectorKeys()) {
		audit.ErrorCode = "all_connectors_failed"
		audit.ErrorSummary = "All Connector invocations failed"
	}
	return audit
}

func CandidateCounts(stats collector.Stats) map[string]int {
	return map[string]int{
		"raw_results": stats.RawResults, "merged_results": stats.MergedResults,
		"results_terminal": stats.ResultsTerminal, "results_pending": stats.ResultsPending,
		"accepted": stats.Accepted, "known_url": stats.KnownURL,
		"out_of_window": stats.OutOfWindow, "invalid_result": stats.InvalidResult,
		"exact_duplicate": stats.ExactDuplicate, "near_duplicate": stats.NearDuplicate,
	}
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

func connectorPosition(name string) int {
	for position, connector := range collector.ConnectorKeys() {
		if name == connector {
			return position
		}
	}
	return len(collector.ConnectorKeys())
}
