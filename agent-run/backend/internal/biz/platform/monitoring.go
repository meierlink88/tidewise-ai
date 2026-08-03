package agentrun

import "time"

type MonitoringWindow string

const (
	MonitoringWindow1Hour   MonitoringWindow = "1h"
	MonitoringWindow6Hours  MonitoringWindow = "6h"
	MonitoringWindow12Hours MonitoringWindow = "12h"
	MonitoringWindow24Hours MonitoringWindow = "24h"
)

func (window MonitoringWindow) Duration() (time.Duration, bool) {
	switch window {
	case MonitoringWindow1Hour:
		return time.Hour, true
	case MonitoringWindow6Hours:
		return 6 * time.Hour, true
	case MonitoringWindow12Hours:
		return 12 * time.Hour, true
	case MonitoringWindow24Hours:
		return 24 * time.Hour, true
	default:
		return 0, false
	}
}

type MonitoringKind string

const (
	MonitoringCollector          MonitoringKind = "collector"
	MonitoringArtifactExtraction MonitoringKind = "artifact_extraction"
	MonitoringSemantic           MonitoringKind = "semantic"
)

type MonitoringState string

const (
	MonitoringStateAll     MonitoringState = "all"
	MonitoringStateSuccess MonitoringState = "success"
	MonitoringStateRunning MonitoringState = "running"
	MonitoringStateFailure MonitoringState = "failure"
)

type MonitoringStatusCount struct {
	Kind   MonitoringKind
	Status string
	Count  int
}

type MonitoringStateCounts struct {
	Success int
	Running int
	Failure int
}

type MonitoringBusinessTotals struct {
	CollectorRawResults        int
	CollectorMergedResults     int
	CollectorAcceptedArtifacts int
	ArtifactPublished          int
	ArtifactNoEvents           int
	ArtifactFormalEvents       int
	SemanticSubmissions        int
	SemanticAcceptedCandidates int
	SemanticRejectedCandidates int
}

type MonitoringStageSummary struct {
	Kind   MonitoringKind
	Counts MonitoringStateCounts
}

type MonitoringSummary struct {
	Window      MonitoringWindow
	GeneratedAt time.Time
	Collector   MonitoringStageSummary
	Artifact    MonitoringStageSummary
	Semantic    MonitoringStageSummary
	Business    MonitoringBusinessTotals
}

type MonitoringListQuery struct {
	Since    time.Time
	Statuses []string
	Page     int
	PageSize int
}

type CollectorMonitoringRecord struct {
	ExecutionID       string
	RawStatus         string
	TriggerSource     string
	StartedAt         *time.Time
	CompletedAt       *time.Time
	RawResults        int
	MergedResults     int
	AcceptedArtifacts int
	ErrorCode         string
}

type CollectorMonitoringPage struct {
	Items      []CollectorMonitoringRecord
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

type ArtifactExtractionMonitoringRecord struct {
	ExtractionKey        string
	ArtifactID           string
	CollectorExecutionID string
	RawStatus            string
	UpdatedAt            time.Time
	StartedAt            *time.Time
	CompletedAt          *time.Time
	EventCandidates      int
	AcknowledgedJournals int
	TotalJournals        int
	ErrorCode            string
}

type ArtifactExtractionMonitoringPage struct {
	Items      []ArtifactExtractionMonitoringRecord
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

type SemanticMonitoringRecord struct {
	WorkItemID         string
	EventID            string
	TriggerSource      string
	RawStatus          string
	UpdatedAt          time.Time
	StartedAt          *time.Time
	CompletedAt        *time.Time
	AttemptCount       int
	MaxAttempts        int
	AcceptedCandidates int
	RejectedCandidates int
	ErrorCode          string
}

type SemanticMonitoringPage struct {
	Items      []SemanticMonitoringRecord
	Page       int
	PageSize   int
	TotalItems int
	TotalPages int
}

func MonitoringStatuses(kind MonitoringKind, state MonitoringState) ([]string, bool) {
	groups, ok := monitoringStatusGroups[kind]
	if !ok {
		return nil, false
	}
	if state == MonitoringStateAll {
		statuses := make([]string, 0, len(groups.success)+len(groups.running)+len(groups.failure))
		statuses = append(statuses, groups.success...)
		statuses = append(statuses, groups.running...)
		statuses = append(statuses, groups.failure...)
		return statuses, true
	}
	switch state {
	case MonitoringStateSuccess:
		return append([]string(nil), groups.success...), true
	case MonitoringStateRunning:
		return append([]string(nil), groups.running...), true
	case MonitoringStateFailure:
		return append([]string(nil), groups.failure...), true
	default:
		return nil, false
	}
}

func MonitoringStateForStatus(kind MonitoringKind, status string) (MonitoringState, bool) {
	groups, ok := monitoringStatusGroups[kind]
	if !ok {
		return "", false
	}
	for _, candidate := range groups.success {
		if candidate == status {
			return MonitoringStateSuccess, true
		}
	}
	for _, candidate := range groups.running {
		if candidate == status {
			return MonitoringStateRunning, true
		}
	}
	for _, candidate := range groups.failure {
		if candidate == status {
			return MonitoringStateFailure, true
		}
	}
	return "", false
}

type monitoringGroups struct {
	success []string
	running []string
	failure []string
}

var monitoringStatusGroups = map[MonitoringKind]monitoringGroups{
	MonitoringCollector: {
		success: []string{"succeeded", "succeeded_no_change", "partially_succeeded"},
		running: []string{"queued", "planning", "collecting", "materializing", "running"},
		failure: []string{"failed"},
	},
	MonitoringArtifactExtraction: {
		success: []string{"published", "no_events"},
		running: []string{"pending", "running", "awaiting_tag_catalog", "ready_to_publish", "publishing", "retry_wait"},
		failure: []string{"blocked", "rejected"},
	},
	MonitoringSemantic: {
		success: []string{"succeeded"},
		running: []string{"pending", "running"},
		failure: []string{"failed"},
	},
}
