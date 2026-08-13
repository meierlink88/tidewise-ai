package eventsemantic

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	AgentKey     = "event-semantic-enricher"
	AgentVersion = "event-semantic-enricher.v3"
)

type ContextLease struct {
	ContextLeaseID         string    `json:"context_lease_id"`
	EventID                string    `json:"event_id"`
	SupersedesSubmissionID string    `json:"supersedes_submission_id,omitempty"`
	Status                 string    `json:"status"`
	LeaseExpiresAt         time.Time `json:"lease_expires_at"`
}

type ContextLeaseRequest struct {
	EventID                string `json:"event_id"`
	SupersedesSubmissionID string `json:"supersedes_submission_id,omitempty"`
	AgentExecutionID       string `json:"agent_execution_id"`
	WorkerID               string `json:"worker_id"`
	LeaseSeconds           int    `json:"lease_seconds"`
}

type EligibleEvent struct {
	EventID string `json:"event_id"`
}

type EligibleEventPage struct {
	Events     []EligibleEvent
	NextCursor string
}

type Event struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Summary     string  `json:"summary"`
	OccurredAt  *string `json:"occurred_at"`
	EventStatus string  `json:"event_status"`
	FactStatus  string  `json:"fact_status"`
}

type Evidence struct {
	EvidenceID           string   `json:"evidence_id"`
	EvidenceHash         string   `json:"evidence_hash"`
	Statement            string   `json:"evidence_statement"`
	SourceLevel          string   `json:"source_level"`
	Relation             string   `json:"relation"`
	SupportsFields       []string `json:"supports_fields"`
	RawDocumentID        string   `json:"raw_document_id"`
	SourceName           string   `json:"source_name"`
	SourceType           string   `json:"source_type"`
	SourceURL            string   `json:"source_url,omitempty"`
	Title                string   `json:"title"`
	PublishedAt          *string  `json:"published_at,omitempty"`
	FirstSeenAt          string   `json:"first_seen_at"`
	KnowledgeAvailableAt string   `json:"knowledge_available_at"`
	AcceptedAt           string   `json:"accepted_at"`
	StatementSource      string   `json:"statement_source"`
}

type Entity struct {
	EntityID      string   `json:"entity_id"`
	EntityType    string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Description   string   `json:"description,omitempty"`
	Status        string   `json:"status"`
}

type VariableDefinition struct {
	Key                   string   `json:"key"`
	Version               int      `json:"version"`
	NameZH                string   `json:"name_zh"`
	NameEN                string   `json:"name_en"`
	Domain                string   `json:"domain"`
	BusinessDefinition    string   `json:"business_definition"`
	ValueType             string   `json:"value_type"`
	Status                string   `json:"status"`
	AllowedDirections     []string `json:"allowed_directions"`
	AllowedUnits          []string `json:"allowed_units"`
	ApplicableEntityTypes []string `json:"applicable_entity_types"`
}

type MeasurementContract struct {
	Representation      string `json:"representation"`
	MaxItemsPerSignal   int    `json:"max_items_per_signal"`
	MaxTextCharacters   int    `json:"max_text_characters"`
	RequiresEvidenceIDs bool   `json:"requires_evidence_ids"`
	NumericValidation   bool   `json:"numeric_validation"`
}

type Context struct {
	ContextLeaseID          string               `json:"context_lease_id"`
	AgentExecutionID        string               `json:"agent_execution_id"`
	WorkerID                string               `json:"worker_id"`
	LeaseExpiresAt          string               `json:"lease_expires_at"`
	ManifestContractVersion string               `json:"manifest_contract_version"`
	ContextFingerprint      string               `json:"context_fingerprint"`
	EventFingerprint        string               `json:"event_fingerprint"`
	EvidenceFingerprint     string               `json:"evidence_fingerprint"`
	OntologyVersion         string               `json:"ontology_version"`
	AcceptancePolicyVersion string               `json:"acceptance_policy_version"`
	Event                   Event                `json:"event"`
	Evidence                []Evidence           `json:"evidence"`
	VariableDefinitions     []VariableDefinition `json:"variable_definitions"`
	AssertionModalities     []string             `json:"assertion_modalities"`
	MeasurementContract     MeasurementContract  `json:"measurement_contract"`
}

type Measurement struct {
	MeasurementText string   `json:"measurement_text"`
	EvidenceIDs     []string `json:"evidence_ids"`
}

type EntityLinkCandidate struct {
	CandidateKey         string   `json:"candidate_key"`
	Mention              string   `json:"mention"`
	EntityID             string   `json:"entity_id"`
	ProjectedEntityType  string   `json:"projected_entity_type"`
	EntityRole           string   `json:"entity_role"`
	EvidenceIDs          []string `json:"evidence_ids"`
	ResolutionMethod     string   `json:"resolution_method"`
	ResolutionConfidence string   `json:"resolution_confidence,omitempty"`
}

type VariableSignalCandidate struct {
	CandidateKey         string        `json:"candidate_key"`
	SubjectLinkKey       string        `json:"subject_link_key"`
	VariableKey          string        `json:"variable_key"`
	VariableVersion      int           `json:"variable_version"`
	Direction            string        `json:"direction"`
	AssertionModality    string        `json:"assertion_modality"`
	EvidenceIDs          []string      `json:"evidence_ids"`
	Measurements         []Measurement `json:"measurements"`
	StatementAt          *string       `json:"statement_at,omitempty"`
	ValidFrom            *string       `json:"valid_from,omitempty"`
	ValidUntil           *string       `json:"valid_until,omitempty"`
	ForecastPeriodStart  *string       `json:"forecast_period_start,omitempty"`
	ForecastPeriodEnd    *string       `json:"forecast_period_end,omitempty"`
	ExtractionConfidence string        `json:"extraction_confidence,omitempty"`
}

type CandidateSet struct {
	EntityLinks     []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals []VariableSignalCandidate `json:"variable_signals"`
}

type SubmissionRequest struct {
	ContextLeaseID          string                    `json:"context_lease_id"`
	EventID                 string                    `json:"event_id"`
	AgentExecutionID        string                    `json:"agent_execution_id"`
	AgentKey                string                    `json:"agent_key"`
	AgentVersion            string                    `json:"agent_version"`
	SupersedesSubmissionID  string                    `json:"supersedes_submission_id,omitempty"`
	GeneratorPromptHash     string                    `json:"generator_prompt_hash"`
	GeneratorModel          string                    `json:"generator_model"`
	ReviewerPromptHash      string                    `json:"reviewer_prompt_hash"`
	ReviewerModel           string                    `json:"reviewer_model"`
	AdjudicatorPromptHash   string                    `json:"adjudicator_prompt_hash"`
	AdjudicatorModel        string                    `json:"adjudicator_model"`
	OntologyVersion         string                    `json:"ontology_version"`
	AcceptancePolicyVersion string                    `json:"acceptance_policy_version"`
	EntityLinks             []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals         []VariableSignalCandidate `json:"variable_signals"`
}

type CandidateDecision struct {
	CandidateKey string `json:"candidate_key"`
	Status       string `json:"status"`
	ReasonCode   string `json:"reason_code,omitempty"`
	RecordID     string `json:"record_id,omitempty"`
}

type ReviewSnapshot struct {
	ReviewerExecutionKey string          `json:"reviewer_execution_key"`
	CanonicalPayloadHash string          `json:"canonical_payload_hash"`
	Payload              json.RawMessage `json:"payload"`
	CreatedAt            string          `json:"created_at"`
}

type ReviewerWorkPackage struct {
	Event            Event                     `json:"event"`
	Evidence         []Evidence                `json:"evidence"`
	ResolvedEntities []Entity                  `json:"resolved_entities"`
	EntityLinks      []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals  []VariableSignalCandidate `json:"variable_signals"`
}

type SubmissionResult struct {
	SubmissionID            string               `json:"submission_id"`
	EventID                 string               `json:"event_id"`
	Status                  string               `json:"status"`
	CanonicalPayloadHash    string               `json:"canonical_payload_hash"`
	Replayed                bool                 `json:"replayed"`
	EntityLinks             []CandidateDecision  `json:"entity_links"`
	VariableSignals         []CandidateDecision  `json:"variable_signals"`
	ReviewerWorkPackage     *ReviewerWorkPackage `json:"reviewer_work_package,omitempty"`
	AuditWorkPackage        *ReviewerWorkPackage `json:"audit_work_package,omitempty"`
	ContextLeaseID          string               `json:"context_lease_id,omitempty"`
	AgentExecutionID        string               `json:"agent_execution_id,omitempty"`
	AgentKey                string               `json:"agent_key,omitempty"`
	AgentVersion            string               `json:"agent_version,omitempty"`
	SupersedesSubmissionID  string               `json:"supersedes_submission_id,omitempty"`
	GeneratorPromptHash     string               `json:"generator_prompt_hash,omitempty"`
	GeneratorModel          string               `json:"generator_model,omitempty"`
	ReviewerPromptHash      string               `json:"reviewer_prompt_hash,omitempty"`
	ReviewerModel           string               `json:"reviewer_model,omitempty"`
	AdjudicatorPromptHash   string               `json:"adjudicator_prompt_hash,omitempty"`
	AdjudicatorModel        string               `json:"adjudicator_model,omitempty"`
	OntologyVersion         string               `json:"ontology_version,omitempty"`
	AcceptancePolicyVersion string               `json:"acceptance_policy_version,omitempty"`
	CandidateSnapshot       json.RawMessage      `json:"candidate_snapshot,omitempty"`
	ReviewSnapshots         []ReviewSnapshot     `json:"review_snapshots,omitempty"`
	CreatedAt               string               `json:"created_at,omitempty"`
	FinalizedAt             *string              `json:"finalized_at,omitempty"`
}

func (s SubmissionResult) CandidateOutcomeCounts() (accepted, rejected int) {
	for _, decisions := range [][]CandidateDecision{s.EntityLinks, s.VariableSignals} {
		for _, decision := range decisions {
			switch decision.Status {
			case "accepted":
				accepted++
			case "rejected":
				rejected++
			}
		}
	}
	return accepted, rejected
}

type ReviewItem struct {
	CandidateType string   `json:"candidate_type"`
	CandidateKey  string   `json:"candidate_key"`
	Decision      string   `json:"decision"`
	ReasonCodes   []string `json:"reason_codes"`
	EvidenceIDs   []string `json:"evidence_ids"`
}

type ReviewRequest struct {
	ReviewerExecutionKey string       `json:"reviewer_execution_key"`
	PromptHash           string       `json:"prompt_hash"`
	Model                string       `json:"model"`
	Items                []ReviewItem `json:"items"`
}

type EntityLookup struct {
	CandidateKey string `json:"candidate_key"`
	Mention      string `json:"mention"`
}

type EntityCandidate struct {
	Entity Entity  `json:"entity"`
	Score  float64 `json:"score"`
}

type EntityCandidateSet struct {
	CandidateKey string            `json:"candidate_key"`
	Candidates   []EntityCandidate `json:"candidates"`
}

// SemanticRetriever is an AgentRun-owned Qdrant consumer port. Implementations
// must execute each method in one Event-batched request, never mention-by-mention.
type SemanticRetriever interface {
	ExactEntities(context.Context, []EntityLookup) ([]EntityCandidateSet, error)
	SearchEntities(context.Context, []EntityLookup, int) ([]EntityCandidateSet, error)
}

type DataClient interface {
	ListEligibleEvents(context.Context, int, string) (EligibleEventPage, error)
	CreateContextLease(context.Context, ContextLeaseRequest) (ContextLease, error)
	Context(context.Context, string) (Context, error)
	CreateSubmission(context.Context, SubmissionRequest) (SubmissionResult, error)
	SubmitReview(context.Context, string, ReviewRequest) (SubmissionResult, error)
	GetEventSemantics(context.Context, string) (EventSemantics, error)
}

type Repository interface {
	EnsureInitialWorkItems(context.Context, []EligibleEvent, time.Time) (int, error)
	EnqueueReanalysis(context.Context, ReanalysisRequest, time.Time) (WorkItem, bool, error)
	StartNextExecution(context.Context, string, string, time.Time) (ExecutionAttempt, bool, error)
	SaveStageAudit(context.Context, string, StageAudit) error
	CompleteExecution(context.Context, ExecutionCompletion) error
}

type ProcessingPermit interface {
	WithEventSemanticProcessingPermit(context.Context, func() error) error
}

type ExecutionAttempt struct {
	ID           string
	WorkItem     WorkItem
	ContextLease ContextLease
	Context      Context
}

type WorkItem struct {
	ID                     string
	EventID                string
	SupersedesSubmissionID string
	TriggerSource          string
	Reason                 string
	IdempotencyKey         string
	Status                 string
	AttemptCount           int
	MaxAttempts            int
	LeaseExpiresAt         *time.Time
	CurrentExecutionID     string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ReanalysisRequest struct {
	EventID                string
	SupersedesSubmissionID string
	Reason                 string
	IdempotencyKey         string
}

type ExecutionCompletion struct {
	ExecutionID     string
	Status          string
	Retryable       bool
	ErrorCode       string
	ErrorSummary    string
	CandidateCounts map[string]int
	CompletedAt     time.Time
}

type Result struct {
	SubmissionID       string
	Status             string
	AcceptedCandidates int
	RejectedCandidates int
	Audit              StageAudit
}

type StageAudit struct {
	ContractVersion     string                    `json:"contract_version"`
	EventID             string                    `json:"event_id"`
	Mentions            []MentionAudit            `json:"mentions"`
	CandidateSets       []CandidateSetAudit       `json:"candidate_sets"`
	Selections          []SelectionAudit          `json:"selections"`
	ApplicableVariables []ApplicableVariableAudit `json:"applicable_variables"`
	Violations          []StageViolationAudit     `json:"violations"`
	Isolations          []CandidateIsolationAudit `json:"isolations"`
	ExecutionFailure    *ExecutionFailureAudit    `json:"execution_failure,omitempty"`
}

type MentionAudit struct {
	CandidateKey string   `json:"candidate_key"`
	Mention      string   `json:"mention"`
	EvidenceIDs  []string `json:"evidence_ids"`
}

type CandidateAudit struct {
	EntityID      string  `json:"entity_id"`
	EntityType    string  `json:"entity_type"`
	CanonicalName string  `json:"canonical_name"`
	Score         float64 `json:"score"`
}

type CandidateSetAudit struct {
	CandidateKey string           `json:"candidate_key"`
	Method       string           `json:"method"`
	Candidates   []CandidateAudit `json:"candidates"`
}

type SelectionAudit struct {
	CandidateKey    string `json:"candidate_key"`
	EntityID        string `json:"entity_id,omitempty"`
	EntityType      string `json:"entity_type,omitempty"`
	EntityRole      string `json:"entity_role,omitempty"`
	NoMatch         bool   `json:"no_match"`
	ResolutionRoute string `json:"resolution_route"`
	ReasonCode      string `json:"reason_code,omitempty"`
	Owner           string `json:"owner,omitempty"`
}

type ApplicableVariableAudit struct {
	SubjectLinkKey string   `json:"subject_link_key"`
	Definitions    []string `json:"definitions"`
}

type StageViolationAudit struct {
	Stage   string   `json:"stage"`
	Attempt string   `json:"attempt"`
	Codes   []string `json:"codes"`
}

type CandidateIsolationAudit struct {
	Stage        string `json:"stage"`
	CandidateKey string `json:"candidate_key,omitempty"`
	ReasonCode   string `json:"reason_code"`
	Owner        string `json:"owner"`
}

type ExecutionFailureAudit struct {
	ReasonCode string `json:"reason_code"`
	Owner      string `json:"owner"`
}

type EventSemantics struct {
	EventID     string             `json:"event_id"`
	Submissions []SubmissionResult `json:"submissions"`
}

type RemoteError struct {
	Status    int
	Code      string
	Summary   string
	Retryable bool
}

func (e *RemoteError) Error() string { return e.Code + ": " + e.Summary }

var ErrModelUnavailable = errors.New("Event Semantic model is unavailable")
var ErrReanalysisIdempotencyConflict = errors.New("Event Semantic reanalysis idempotency key conflicts with an existing Work Item")
