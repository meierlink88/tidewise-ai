package eventfact

import (
	"encoding/json"
	"time"
)

const (
	AgentKey     = "event-fact-extractor"
	AgentVersion = "event-fact-extractor.v1"
)

type WorkStatus string

const (
	WorkPending            WorkStatus = "pending"
	WorkRunning            WorkStatus = "running"
	WorkAwaitingTagCatalog WorkStatus = "awaiting_tag_catalog"
	WorkAwaitingReview     WorkStatus = "awaiting_review"
	WorkReadyToPublish     WorkStatus = "ready_to_publish"
	WorkPublishing         WorkStatus = "publishing"
	WorkPublished          WorkStatus = "published"
	WorkPartiallyPublished WorkStatus = "partially_published"
	WorkRetryWait          WorkStatus = "retry_wait"
	WorkBlocked            WorkStatus = "blocked"
	WorkRejected           WorkStatus = "rejected"
	WorkNoEvents           WorkStatus = "no_events"
)

type ReviewState string

const (
	ReviewAutoApproved ReviewState = "auto_approved"
	ReviewManual       ReviewState = "manual_review"
	ReviewRejected     ReviewState = "rejected"
)

type WorkItem struct {
	Key                   string
	CollectorExecutionIDs []string
	ExtractorAgentVersion string
	Status                WorkStatus
	CurrentExecutionID    string
	ExtractionResult      json.RawMessage
	TagCatalogRevision    string
	TagCatalogHash        string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ExtractionSnapshot struct {
	PromptSHA256 string
	SchemaSHA256 string
	ProviderKey  string
	Model        string
}

type ArtifactUnit struct {
	Key                  string
	WorkItemKey          string
	ArtifactOrdinal      int
	ArtifactID           string
	CollectorExecutionID string
	ContentSHA256        string
	Status               WorkStatus
	CurrentExecutionID   string
	ExtractionResult     json.RawMessage
	TagCatalogRevision   string
	TagCatalogHash       string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ExecutionAttempt struct {
	ID       string
	WorkItem WorkItem
	Unit     ArtifactUnit
	Snapshot ExtractionSnapshot
}

type Tag struct {
	ID       string `json:"id"`
	Kind     string `json:"tag_kind"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type TagCatalog struct {
	Revision string `json:"catalog_revision"`
	Hash     string `json:"catalog_hash"`
	Tags     []Tag  `json:"tags"`
}

type Artifact struct {
	ArtifactID           string     `json:"artifact_id"`
	CollectorExecutionID string     `json:"collector_execution_id"`
	DocumentID           string     `json:"document_id"`
	Title                string     `json:"title"`
	SourceName           string     `json:"source_name"`
	SourceType           string     `json:"source_type"`
	SourceURL            string     `json:"source_url"`
	ContentLevel         string     `json:"content_level"`
	PublishedAt          *time.Time `json:"published_at"`
	CollectedAt          time.Time  `json:"collected_at"`
	Language             string     `json:"language"`
	ContentSHA256        string     `json:"content_sha256"`
	Body                 string     `json:"body"`
}

type Candidate struct {
	CandidateID      string         `json:"candidate_id"`
	ArtifactID       string         `json:"artifact_id"`
	Title            string         `json:"title"`
	FactualSummary   string         `json:"factual_summary"`
	OccurredAt       *time.Time     `json:"occurred_at,omitempty"`
	FactPayload      map[string]any `json:"fact_payload"`
	EvidenceExcerpt  string         `json:"evidence_excerpt"`
	SupportsFields   []string       `json:"supports_fields"`
	SourceLevel      string         `json:"source_level"`
	ActorMentions    []string       `json:"actor_mentions"`
	Action           string         `json:"action"`
	ObjectMentions   []string       `json:"object_mentions"`
	Change           map[string]any `json:"change"`
	LifecycleStatus  string         `json:"lifecycle_status"`
	TimePrecision    string         `json:"time_precision"`
	LocationMentions []string       `json:"location_mentions"`
	ReferencePeriod  string         `json:"reference_period"`
	Quantities       []string       `json:"quantities"`
	TagCodes         []string       `json:"tag_codes"`
	DedupeKey        string         `json:"dedupe_key"`
	IdentityHash     string         `json:"identity_hash"`
	Tags             []AssignedTag  `json:"tags"`
	Review           Review         `json:"review"`
	ReviewState      ReviewState    `json:"review_state"`
}

type AssignedTag struct {
	ID               string  `json:"id"`
	Kind             string  `json:"kind"`
	Code             string  `json:"code"`
	Confidence       float64 `json:"confidence"`
	AssignmentReason string  `json:"assignment_reason"`
}

type Review struct {
	SemanticPass bool     `json:"semantic_pass"`
	Conflict     bool     `json:"conflict"`
	Reasons      []string `json:"reasons"`
	Confidence   float64  `json:"confidence"`
}

type Result struct {
	ExecutionID          string            `json:"execution_id"`
	Artifacts            []ArtifactSummary `json:"artifacts"`
	Candidates           []Candidate       `json:"candidates"`
	NoEventReason        map[string]string `json:"no_event_reasons"`
	ExtractionModelCalls int               `json:"extraction_model_calls"`
	ReviewModelCalls     int               `json:"review_model_calls"`
	PublicationArtifacts []Artifact        `json:"-"`
}

type ArtifactSummary struct {
	ArtifactID           string `json:"artifact_id"`
	CollectorExecutionID string `json:"collector_execution_id"`
	ContentSHA256        string `json:"content_sha256"`
}

type JournalEntry struct {
	WorkItemKey  string
	UnitKey      string
	BatchOrdinal int
	PackageID    string
	Payload      []byte
	PayloadHash  string
	Status       string
	ReceiptID    string
	AttemptCount int
}

type CanonicalEvent struct {
	DedupeKey    string
	IdentityHash string
	CoreFacts    json.RawMessage
}
