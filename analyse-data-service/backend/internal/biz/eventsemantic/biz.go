package eventsemantic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	ReadStore
	TransactionStore
}

type ReadStore interface {
	ListEligibleEvents(context.Context, int, *EligibleEventCursor) ([]EligibleEvent, error)
	Context(context.Context, string) (Context, error)
	SubmissionContext(context.Context, string, Submission) (Context, error)
	ReplaySubmission(context.Context, string, string) (SubmissionResult, bool, error)
	GetEventSemantics(context.Context, string) (EventSemanticsResult, error)
}

type NotFoundError struct{ Resource string }

func (e *NotFoundError) Error() string { return e.Resource + " not found" }

type ConflictError struct{ Reason string }

func (e *ConflictError) Error() string { return e.Reason }

type ContextDriftError struct{ Reason string }

func (e *ContextDriftError) Error() string { return e.Reason }

type ValidationError struct{ Reason string }

func (e *ValidationError) Error() string { return e.Reason }

type NotRequiredError struct{ Reason string }

func (e *NotRequiredError) Error() string { return e.Reason }

type InputInvalidError struct{ Reason string }

func (e *InputInvalidError) Error() string { return e.Reason }

type UseCase struct {
	store Store
}

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Event Semantic store is required")
	}
	return &UseCase{store: store}, nil
}

func (s *UseCase) ListEligibleEvents(
	ctx context.Context,
	limit int,
	cursor string,
) (EligibleEventPage, error) {
	if s == nil || s.store == nil {
		return EligibleEventPage{}, errors.New("Event Semantics store is required")
	}
	if limit < 1 || limit > 100 {
		return EligibleEventPage{}, &ValidationError{Reason: "limit must be between one and one hundred"}
	}
	after, err := decodeEligibleEventCursor(cursor)
	if err != nil {
		return EligibleEventPage{}, &ValidationError{Reason: "cursor is invalid"}
	}
	items, err := s.store.ListEligibleEvents(ctx, limit+1, after)
	if err != nil {
		return EligibleEventPage{}, err
	}
	page := EligibleEventPage{Events: items}
	if len(items) > limit {
		page.Events = items[:limit]
		page.NextCursor, err = encodeEligibleEventCursor(page.Events[len(page.Events)-1])
		if err != nil {
			return EligibleEventPage{}, err
		}
	}
	return page, nil
}

type eligibleEventCursorPayload struct {
	Version     int       `json:"v"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	EventID     string    `json:"event_id"`
}

func encodeEligibleEventCursor(item EligibleEvent) (string, error) {
	if item.FirstSeenAt.IsZero() {
		return "", errors.New("eligible Event cursor time is required")
	}
	if _, err := uuid.Parse(item.EventID); err != nil {
		return "", errors.New("eligible Event cursor ID is invalid")
	}
	payload, err := json.Marshal(eligibleEventCursorPayload{
		Version: 1, FirstSeenAt: item.FirstSeenAt.UTC(), EventID: item.EventID,
	})
	if err != nil {
		return "", fmt.Errorf("encode eligible Event cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeEligibleEventCursor(value string) (*EligibleEventCursor, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > 1024 {
		return nil, errors.New("eligible Event cursor encoding is invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(payload) > 1024 {
		return nil, errors.New("eligible Event cursor encoding is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var decoded eligibleEventCursorPayload
	if err := decoder.Decode(&decoded); err != nil {
		return nil, errors.New("eligible Event cursor payload is invalid")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("eligible Event cursor payload is invalid")
	}
	if decoded.Version != 1 || decoded.FirstSeenAt.IsZero() {
		return nil, errors.New("eligible Event cursor version is invalid")
	}
	if _, err := uuid.Parse(decoded.EventID); err != nil {
		return nil, errors.New("eligible Event cursor ID is invalid")
	}
	return &EligibleEventCursor{
		FirstSeenAt: decoded.FirstSeenAt.UTC(),
		EventID:     decoded.EventID,
	}, nil
}

func (s *UseCase) CreateContextLease(ctx context.Context, request ContextLeaseRequest) (ContextLease, error) {
	if s == nil || s.store == nil {
		return ContextLease{}, errors.New("Event Semantics store is required")
	}
	if strings.TrimSpace(request.EventID) == "" ||
		strings.TrimSpace(request.AgentExecutionID) == "" ||
		strings.TrimSpace(request.WorkerID) == "" ||
		request.Lease < time.Minute || request.Lease > 15*time.Minute {
		return ContextLease{}, &ValidationError{Reason: "event_id, agent_execution_id, worker_id and lease_seconds are required"}
	}
	request.EventID = strings.TrimSpace(request.EventID)
	request.SupersedesSubmissionID = strings.TrimSpace(request.SupersedesSubmissionID)
	request.AgentExecutionID = strings.TrimSpace(request.AgentExecutionID)
	request.WorkerID = strings.TrimSpace(request.WorkerID)
	return s.store.CreateContextLease(ctx, request)
}

func (s *UseCase) Context(ctx context.Context, contextLeaseID string) (Context, error) {
	if strings.TrimSpace(contextLeaseID) == "" {
		return Context{}, &ValidationError{Reason: "context_lease_id is required"}
	}
	return s.store.Context(ctx, contextLeaseID)
}

func (s *UseCase) CreateSubmission(ctx context.Context, submission Submission) (SubmissionResult, error) {
	if strings.TrimSpace(submission.ContextLeaseID) == "" || strings.TrimSpace(submission.EventID) == "" ||
		strings.TrimSpace(submission.AgentExecutionID) == "" ||
		submission.AgentKey != "event-semantic-enricher" ||
		submission.AgentVersion != "event-semantic-enricher.v3" {
		return SubmissionResult{}, &ValidationError{Reason: "submission identity is invalid"}
	}
	if err := validateSubmissionMetadata(submission); err != nil {
		return SubmissionResult{}, err
	}
	payload, hash, err := canonicalHash(submission)
	if err != nil {
		return SubmissionResult{}, err
	}
	if existing, found, err := s.store.ReplaySubmission(ctx, submission.AgentExecutionID, hash); err != nil {
		return SubmissionResult{}, err
	} else if found {
		return existing, nil
	}
	contextSnapshot, err := s.store.SubmissionContext(ctx, submission.ContextLeaseID, submission)
	if err != nil {
		return SubmissionResult{}, err
	}
	if contextSnapshot.Event.ID != submission.EventID {
		return SubmissionResult{}, &ConflictError{Reason: "context lease is bound to a different Event"}
	}
	if submission.OntologyVersion != contextSnapshot.OntologyVersion ||
		submission.AcceptancePolicyVersion != contextSnapshot.PolicyVersion {
		return SubmissionResult{}, &ConflictError{Reason: "ontology or acceptance policy snapshot changed"}
	}
	precheck := Precheck(contextSnapshot, submission)
	return s.store.CreateSubmission(ctx, submission, precheck, payload, hash)
}

func (s *UseCase) SubmitReview(ctx context.Context, submission ReviewSubmission) (SubmissionResult, error) {
	if strings.TrimSpace(submission.SubmissionID) == "" || strings.TrimSpace(submission.ReviewerExecutionKey) == "" ||
		!validHash(submission.PromptHash) || strings.TrimSpace(submission.Model) == "" ||
		len(submission.Items) == 0 {
		return SubmissionResult{}, &ValidationError{Reason: "review identity and items are invalid"}
	}
	for _, item := range submission.Items {
		if !contains([]string{"entity_link", "variable_signal"}, item.CandidateType) ||
			strings.TrimSpace(item.CandidateKey) == "" ||
			!contains([]string{"pass", "fail", "indeterminate"}, item.Decision) ||
			len(item.EvidenceIDs) == 0 {
			return SubmissionResult{}, &ValidationError{Reason: "review item is invalid"}
		}
	}
	payload, hash, err := canonicalHash(submission)
	if err != nil {
		return SubmissionResult{}, err
	}
	return s.store.SubmitReview(ctx, submission, payload, hash)
}

func (s *UseCase) Get(ctx context.Context, eventID string) (EventSemanticsResult, error) {
	if strings.TrimSpace(eventID) == "" {
		return EventSemanticsResult{}, &ValidationError{Reason: "event_id is required"}
	}
	return s.store.GetEventSemantics(ctx, eventID)
}

func validateSubmissionMetadata(submission Submission) error {
	for _, hash := range []string{
		submission.GeneratorPromptHash,
		submission.ReviewerPromptHash,
		submission.AdjudicatorPromptHash,
	} {
		if !validHash(hash) {
			return &ValidationError{Reason: "prompt hashes must be lowercase SHA-256"}
		}
	}
	if strings.TrimSpace(submission.GeneratorModel) == "" ||
		strings.TrimSpace(submission.ReviewerModel) == "" ||
		strings.TrimSpace(submission.AdjudicatorModel) == "" ||
		strings.TrimSpace(submission.OntologyVersion) == "" ||
		strings.TrimSpace(submission.AcceptancePolicyVersion) == "" {
		return &ValidationError{Reason: "submission version snapshots are required"}
	}
	for _, keys := range [][]string{
		entityLinkKeys(submission.EntityLinks),
		variableSignalKeys(submission.VariableSignals),
	} {
		seen := make(map[string]struct{}, len(keys))
		for _, key := range keys {
			if _, exists := seen[key]; exists {
				return &ValidationError{Reason: "candidate keys must be unique within each candidate type"}
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func entityLinkKeys(items []EntityLinkCandidate) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Key)
	}
	return result
}

func variableSignalKeys(items []VariableSignalCandidate) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Key)
	}
	return result
}

func validHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func canonicalHash(value any) ([]byte, string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, "", fmt.Errorf("encode canonical Event Semantics payload: %w", err)
	}
	sum := sha256.Sum256(payload)
	return payload, hex.EncodeToString(sum[:]), nil
}

type ReviewStatus string

const (
	StatusPendingReview   ReviewStatus = "pending_review"
	StatusNeedsReanalysis ReviewStatus = "needs_reanalysis"
	StatusQuarantined     ReviewStatus = "quarantined"
	StatusAccepted        ReviewStatus = "accepted"
	StatusRejected        ReviewStatus = "rejected"
	StatusSuperseded      ReviewStatus = "superseded"
)

type Event struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Summary    string     `json:"summary"`
	OccurredAt *time.Time `json:"occurred_at"`
	Status     string     `json:"event_status"`
	FactStatus string     `json:"fact_status"`
}

type Evidence struct {
	ID                   string     `json:"evidence_id"`
	Hash                 string     `json:"evidence_hash"`
	Statement            string     `json:"evidence_statement"`
	SourceLevel          string     `json:"source_level"`
	Relation             string     `json:"relation"`
	SupportsFields       []string   `json:"supports_fields"`
	RawDocumentID        string     `json:"raw_document_id"`
	SourceName           string     `json:"source_name"`
	SourceType           string     `json:"source_type"`
	SourceURL            string     `json:"source_url,omitempty"`
	Title                string     `json:"title"`
	PublishedAt          *time.Time `json:"published_at,omitempty"`
	FirstSeenAt          time.Time  `json:"first_seen_at"`
	KnowledgeAvailableAt time.Time  `json:"knowledge_available_at"`
	AcceptedAt           time.Time  `json:"accepted_at"`
	StatementSource      string     `json:"statement_source"`
}

type Entity struct {
	ID            string   `json:"entity_id"`
	Type          string   `json:"entity_type"`
	Name          string   `json:"name"`
	CanonicalName string   `json:"canonical_name"`
	Aliases       []string `json:"aliases"`
	Status        string   `json:"status"`
}

type EntityRelation struct {
	ID           string `json:"entity_relation_id"`
	FromEntityID string `json:"from_entity_id"`
	ToEntityID   string `json:"to_entity_id"`
	Type         string `json:"relation_type"`
	Status       string `json:"status"`
}

type EntityTypeDefinition struct {
	TypeKey              string   `json:"type_key"`
	Version              int      `json:"version"`
	NameZH               string   `json:"name_zh"`
	NameEN               string   `json:"name_en"`
	BusinessDefinition   string   `json:"business_definition"`
	InclusionCriteria    []string `json:"inclusion_criteria"`
	ExclusionCriteria    []string `json:"exclusion_criteria"`
	EventLinkAllowed     bool     `json:"event_link_allowed"`
	SignalSubjectAllowed bool     `json:"signal_subject_allowed"`
	AllowedEventRoles    []string `json:"allowed_event_roles"`
	Status               string   `json:"status"`
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

type DirectTransmissionRule struct {
	Key                     string `json:"rule_key"`
	Version                 int    `json:"version"`
	Status                  string `json:"status"`
	SourceEntityType        string `json:"source_entity_type"`
	SourceVariableKey       string `json:"source_variable_key"`
	SourceVariableVersion   int    `json:"source_variable_version"`
	SourceDirection         string `json:"source_direction"`
	RelationType            string `json:"relation_type"`
	TargetEntityType        string `json:"target_entity_type"`
	AffectedVariableKey     string `json:"affected_variable_key"`
	AffectedVariableVersion int    `json:"affected_variable_version"`
	AffectedDirection       string `json:"affected_direction"`
	ConditionSummary        string `json:"condition_summary"`
	MechanismTemplate       string `json:"mechanism_template"`
}

type Context struct {
	ContextLeaseID          string                   `json:"context_lease_id"`
	AgentExecutionID        string                   `json:"agent_execution_id"`
	WorkerID                string                   `json:"worker_id"`
	LeaseExpiresAt          time.Time                `json:"lease_expires_at"`
	ManifestContractVersion string                   `json:"manifest_contract_version"`
	ContextFingerprint      string                   `json:"context_fingerprint"`
	EventFingerprint        string                   `json:"event_fingerprint"`
	EvidenceFingerprint     string                   `json:"evidence_fingerprint"`
	OntologyVersion         string                   `json:"ontology_version"`
	PolicyVersion           string                   `json:"acceptance_policy_version"`
	RouteContractVersion    string                   `json:"-"`
	Event                   Event                    `json:"event"`
	Evidence                []Evidence               `json:"evidence"`
	EntityTypes             []EntityTypeDefinition   `json:"entity_type_definitions"`
	Variables               []VariableDefinition     `json:"variable_definitions"`
	AssertionModalities     []string                 `json:"assertion_modalities"`
	MeasurementContract     MeasurementContract      `json:"measurement_contract"`
	Rules                   []DirectTransmissionRule `json:"-"`
	// Legacy in-memory compatibility only. Compact manifests never persist these ABox collections.
	Entities  []Entity         `json:"-"`
	Relations []EntityRelation `json:"-"`
}

type MeasurementContract struct {
	Representation      string `json:"representation"`
	MaxItemsPerSignal   int    `json:"max_items_per_signal"`
	MaxTextCharacters   int    `json:"max_text_characters"`
	RequiresEvidenceIDs bool   `json:"requires_evidence_ids"`
	NumericValidation   bool   `json:"numeric_validation"`
}

type EvidenceReference struct {
	EvidenceID  string `json:"evidence_id"`
	Fingerprint string `json:"evidence_fingerprint"`
}

type VersionReference struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

// ContextManifest persists only immutable identities, version references and fingerprints.
// The Context API hydrates the bounded Event/Evidence/TBox payload from these references.
type ContextManifest struct {
	ContextLeaseID          string              `json:"context_lease_id"`
	AgentExecutionID        string              `json:"agent_execution_id"`
	WorkerID                string              `json:"worker_id"`
	LeaseStatus             string              `json:"lease_status"`
	LeaseExpiresAt          time.Time           `json:"lease_expires_at"`
	ManifestContractVersion string              `json:"manifest_contract_version"`
	ManifestFingerprint     string              `json:"manifest_fingerprint"`
	ContextFingerprint      string              `json:"context_fingerprint"`
	EventID                 string              `json:"event_id"`
	EventFingerprint        string              `json:"event_fingerprint"`
	Evidence                []EvidenceReference `json:"evidence_references"`
	EvidenceFingerprint     string              `json:"evidence_fingerprint"`
	OntologyVersion         string              `json:"ontology_version"`
	PolicyVersion           string              `json:"acceptance_policy_version"`
	RouteContractVersion    string              `json:"-"`
	EntityTypes             []VersionReference  `json:"entity_type_references"`
	Variables               []VersionReference  `json:"variable_definition_references"`
	Rules                   []VersionReference  `json:"-"`
}

type MeasurementValue struct {
	Text        string   `json:"measurement_text"`
	EvidenceIDs []string `json:"evidence_ids"`
	// Legacy fields remain readable for historical snapshots only and are never part of V2 writes.
	Role             string  `json:"-"`
	Shape            string  `json:"-"`
	RawValue         *string `json:"-"`
	RawLower         *string `json:"-"`
	RawUpper         *string `json:"-"`
	RawUnit          string  `json:"-"`
	CanonicalValue   *string `json:"-"`
	CanonicalLower   *string `json:"-"`
	CanonicalUpper   *string `json:"-"`
	CanonicalUnit    string  `json:"-"`
	Currency         string  `json:"-"`
	Scale            string  `json:"-"`
	ComparisonBasis  string  `json:"-"`
	ComparisonPeriod string  `json:"-"`
	RawText          string  `json:"-"`
	IsApproximate    bool    `json:"-"`
	EvidenceID       string  `json:"-"`
}

type EntityLinkCandidate struct {
	Key                  string             `json:"candidate_key"`
	Mention              string             `json:"mention"`
	EntityID             string             `json:"entity_id"`
	ProjectedEntityType  string             `json:"projected_entity_type"`
	EntityRole           string             `json:"entity_role"`
	EvidenceIDs          []string           `json:"evidence_ids"`
	ResolutionMethod     string             `json:"resolution_method"`
	ResolutionConfidence string             `json:"resolution_confidence,omitempty"`
	ResolutionReceipt    *ResolutionReceipt `json:"-"`
}

type VariableSignalCandidate struct {
	Key                  string             `json:"candidate_key"`
	SubjectLinkKey       string             `json:"subject_link_key"`
	VariableKey          string             `json:"variable_key"`
	VariableVersion      int                `json:"variable_version"`
	Direction            string             `json:"direction"`
	AssertionModality    string             `json:"assertion_modality"`
	EvidenceIDs          []string           `json:"evidence_ids"`
	Measurements         []MeasurementValue `json:"measurements"`
	StatementAt          *time.Time         `json:"statement_at,omitempty"`
	ValidFrom            *time.Time         `json:"valid_from,omitempty"`
	ValidUntil           *time.Time         `json:"valid_until,omitempty"`
	ForecastPeriodStart  *time.Time         `json:"forecast_period_start,omitempty"`
	ForecastPeriodEnd    *time.Time         `json:"forecast_period_end,omitempty"`
	ExtractionConfidence string             `json:"extraction_confidence,omitempty"`
}

type DirectImpactCandidate struct {
	Key                     string   `json:"candidate_key"`
	SourceSignalKey         string   `json:"source_signal_key"`
	TargetEntityID          string   `json:"target_entity_id"`
	AffectedVariableKey     string   `json:"affected_variable_key"`
	AffectedVariableVersion int      `json:"affected_variable_version"`
	AffectedDirection       string   `json:"affected_direction"`
	DerivationType          string   `json:"derivation_type"`
	MechanismSummary        string   `json:"mechanism_summary"`
	EntityRelationID        string   `json:"entity_relation_id,omitempty"`
	RuleKey                 string   `json:"rule_key,omitempty"`
	RuleVersion             int      `json:"rule_version,omitempty"`
	EvidenceIDs             []string `json:"evidence_ids"`
	AssertionConfidence     string   `json:"assertion_confidence,omitempty"`
}

type Submission struct {
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
	AdjudicatorPromptHash   string                    `json:"adjudicator_prompt_hash,omitempty"`
	AdjudicatorModel        string                    `json:"adjudicator_model,omitempty"`
	OntologyVersion         string                    `json:"ontology_version"`
	AcceptancePolicyVersion string                    `json:"acceptance_policy_version"`
	EntityLinks             []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals         []VariableSignalCandidate `json:"variable_signals"`
	// Historical persistence compatibility only. V2 never accepts or emits DirectImpact candidates.
	DirectImpacts []DirectImpactCandidate `json:"-"`
}

type CandidateDecision struct {
	CandidateKey string       `json:"candidate_key"`
	Status       ReviewStatus `json:"status"`
	ReasonCode   string       `json:"reason_code,omitempty"`
	RecordID     string       `json:"record_id,omitempty"`
}

type ReviewerWorkPackage struct {
	Event            Event                     `json:"event"`
	Evidence         []Evidence                `json:"evidence"`
	ResolvedEntities []Entity                  `json:"resolved_entities"`
	EntityLinks      []EntityLinkCandidate     `json:"entity_links"`
	VariableSignals  []VariableSignalCandidate `json:"variable_signals"`
	DirectImpacts    []DirectImpactCandidate   `json:"-"`
}

type PrecheckResult struct {
	EntityLinks         []CandidateDecision `json:"entity_links"`
	VariableSignals     []CandidateDecision `json:"variable_signals"`
	DirectImpacts       []CandidateDecision `json:"-"`
	ReviewerWorkPackage ReviewerWorkPackage `json:"reviewer_work_package"`
}

type ContextLease struct {
	ID                     string
	EventID                string
	SupersedesSubmissionID string
	Status                 string
	LeaseExpiresAt         time.Time
}

type ContextLeaseRequest struct {
	EventID                string
	SupersedesSubmissionID string
	AgentExecutionID       string
	WorkerID               string
	Lease                  time.Duration
}

type EligibleEvent struct {
	EventID     string
	FirstSeenAt time.Time
}

type EligibleEventCursor struct {
	FirstSeenAt time.Time
	EventID     string
}

type EligibleEventPage struct {
	Events     []EligibleEvent
	NextCursor string
}

type EntityMention struct {
	Mention            string
	AllowedEntityTypes []string
}

type EntityResolution struct {
	Mention    string
	Candidates []Entity
	Ambiguous  bool
}

type DirectTarget struct {
	Entity   Entity
	Relation EntityRelation
}

type ResolutionRoute struct {
	ID                  string            `json:"route_id"`
	ContractVersion     string            `json:"route_contract_version"`
	TargetEntityType    string            `json:"target_entity_type"`
	AnchorEntityType    string            `json:"anchor_entity_type"`
	MappingRelationType string            `json:"mapping_relation_type"`
	Partitions          []string          `json:"partitions"`
	PartitionLabels     map[string]string `json:"partition_labels"`
	Direction           string            `json:"direction"`
	Purpose             string            `json:"purpose"`
	NextOperation       string            `json:"next_operation"`
	OrderingContract    string            `json:"ordering_contract"`
}

type ResolutionAnchor struct {
	Entity            Entity `json:"entity"`
	Partition         string `json:"partition"`
	Description       string `json:"description"`
	HierarchyIdentity string `json:"hierarchy_identity"`
}

type ResolutionReceipt struct {
	RouteID               string `json:"route_id"`
	RouteContractVersion  string `json:"route_contract_version"`
	AnchorEntityID        string `json:"anchor_entity_id"`
	IndustryChainEntityID string `json:"industry_chain_entity_id"`
	MappingRelationID     string `json:"mapping_relation_id"`
	TargetEntityID        string `json:"target_entity_id"`
	MembershipPosition    int    `json:"membership_position"`
	MembershipUpdatedAt   string `json:"membership_updated_at"`
	PathFingerprint       string `json:"path_fingerprint"`
}

type ResolutionCandidate struct {
	Entity                  Entity            `json:"entity"`
	Description             string            `json:"description"`
	MatchedAnchorEntityIDs  []string          `json:"matched_anchor_entity_ids"`
	IndustryChainEntityName string            `json:"industry_chain_entity_name"`
	Receipt                 ResolutionReceipt `json:"resolution_receipt"`
}

type ResolutionAnchorPage struct {
	Anchors    []ResolutionAnchor
	NextCursor string
}

type ResolutionCandidatePage struct {
	Candidates []ResolutionCandidate
	NextCursor string
}

// ResolutionKeyset is the stable database ordering boundary carried by an opaque API cursor.
type ResolutionKeyset struct {
	CanonicalName string
	EntityID      string
}

type ReviewItem struct {
	CandidateType string
	CandidateKey  string
	Decision      string
	ReasonCodes   []string
	EvidenceIDs   []string
}

type ReviewSubmission struct {
	SubmissionID         string
	ReviewerExecutionKey string
	PromptHash           string
	Model                string
	Items                []ReviewItem
}

type SubmissionResult struct {
	SubmissionID            string
	EventID                 string
	Status                  ReviewStatus
	CanonicalPayloadHash    string
	Replayed                bool
	Precheck                PrecheckResult
	ContextLeaseID          string
	AgentExecutionID        string
	AgentKey                string
	AgentVersion            string
	SupersedesSubmissionID  string
	GeneratorPromptHash     string
	GeneratorModel          string
	ReviewerPromptHash      string
	ReviewerModel           string
	AdjudicatorPromptHash   string
	AdjudicatorModel        string
	OntologyVersion         string
	AcceptancePolicyVersion string
	CandidateSnapshot       json.RawMessage
	ReviewSnapshots         []ReviewSnapshot
	CreatedAt               time.Time
	FinalizedAt             *time.Time
}

type ReviewSnapshot struct {
	ReviewerExecutionKey string
	CanonicalPayloadHash string
	Payload              json.RawMessage
	CreatedAt            time.Time
}

type EventSemanticsResult struct {
	EventID     string
	Submissions []SubmissionResult
}

var confidencePattern = regexp.MustCompile(`^(0(\.\d{1,5})?|1(\.0{1,5})?)$`)

func Precheck(context Context, submission Submission) PrecheckResult {
	result := PrecheckResult{
		ReviewerWorkPackage: ReviewerWorkPackage{
			Event: context.Event, Evidence: append([]Evidence(nil), context.Evidence...),
		},
	}
	evidence := indexEvidence(context.Evidence)
	entities := indexEntities(context.Entities)
	variables := indexVariables(context.Variables)

	linkByKey := make(map[string]EntityLinkCandidate, len(submission.EntityLinks))
	linkStatus := make(map[string]ReviewStatus, len(submission.EntityLinks))
	seenEntityIDs := make(map[string]struct{}, len(submission.EntityLinks))
	reviewerEntityIDs := make(map[string]struct{}, len(submission.EntityLinks))
	for _, candidate := range submission.EntityLinks {
		reason := validateLink(context, candidate, evidence, entities)
		if reason == "" {
			if _, duplicate := seenEntityIDs[candidate.EntityID]; duplicate {
				reason = "duplicate_entity_link"
			} else {
				seenEntityIDs[candidate.EntityID] = struct{}{}
			}
		}
		decision := decision(candidate.Key, reason)
		result.EntityLinks = append(result.EntityLinks, decision)
		linkByKey[candidate.Key], linkStatus[candidate.Key] = candidate, decision.Status
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.EntityLinks = append(result.ReviewerWorkPackage.EntityLinks, candidate)
			if _, exists := reviewerEntityIDs[candidate.EntityID]; !exists {
				result.ReviewerWorkPackage.ResolvedEntities = append(
					result.ReviewerWorkPackage.ResolvedEntities, entities[candidate.EntityID],
				)
				reviewerEntityIDs[candidate.EntityID] = struct{}{}
			}
		}
	}

	for _, candidate := range submission.VariableSignals {
		reason := validateSignal(context, candidate, evidence, entities, variables, linkByKey, linkStatus)
		decision := decision(candidate.Key, reason)
		result.VariableSignals = append(result.VariableSignals, decision)
		if decision.Status == StatusPendingReview {
			result.ReviewerWorkPackage.VariableSignals = append(result.ReviewerWorkPackage.VariableSignals, candidate)
		}
	}
	return result
}

func decision(key, reason string) CandidateDecision {
	if reason != "" {
		return CandidateDecision{CandidateKey: key, Status: StatusRejected, ReasonCode: reason}
	}
	return CandidateDecision{CandidateKey: key, Status: StatusPendingReview}
}

func validateLink(context Context, candidate EntityLinkCandidate, evidence map[string]Evidence, entities map[string]Entity) string {
	if context.Event.Status != "confirmed" || context.Event.FactStatus != "verified" {
		return "event_not_eligible"
	}
	if strings.TrimSpace(candidate.Key) == "" || strings.TrimSpace(candidate.Mention) == "" ||
		strings.TrimSpace(candidate.EntityRole) == "" || strings.TrimSpace(candidate.ResolutionMethod) == "" {
		return "link_invalid"
	}
	entity, exists := entities[candidate.EntityID]
	if !exists || entity.Status != "active" {
		return "entity_not_found"
	}
	if strings.TrimSpace(candidate.ProjectedEntityType) == "" || candidate.ProjectedEntityType != entity.Type {
		return "entity_projection_type_mismatch"
	}
	entityType, exists := activeEntityType(context.EntityTypes, entity.Type)
	if !exists || !entityType.EventLinkAllowed || !contains(entityType.AllowedEventRoles, candidate.EntityRole) {
		return "entity_role_invalid"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !contains([]string{"qdrant_exact", "qdrant_vector"}, candidate.ResolutionMethod) {
		return "entity_resolution_method_invalid"
	}
	if !mentionGrounded(context, candidate) {
		return "entity_evidence_lineage_invalid"
	}
	if !validConfidence(candidate.ResolutionConfidence) {
		return "confidence_invalid"
	}
	return ""
}

func validateSignal(
	context Context,
	candidate VariableSignalCandidate,
	evidence map[string]Evidence,
	entities map[string]Entity,
	variables map[string]VariableDefinition,
	links map[string]EntityLinkCandidate,
	linkStatus map[string]ReviewStatus,
) string {
	link, exists := links[candidate.SubjectLinkKey]
	if !exists {
		return "subject_link_not_found"
	}
	if linkStatus[candidate.SubjectLinkKey] == StatusRejected {
		return "upstream_rejected"
	}
	entity := entities[link.EntityID]
	entityType, exists := activeEntityType(context.EntityTypes, entity.Type)
	if !exists || !entityType.SignalSubjectAllowed {
		return "signal_subject_not_allowed"
	}
	variable, exists := variables[definitionIdentity(candidate.VariableKey, candidate.VariableVersion)]
	if !exists || variable.Status != "active" {
		return "variable_not_found"
	}
	if !contains(variable.ApplicableEntityTypes, entity.Type) {
		return "variable_not_applicable"
	}
	if !contains(variable.AllowedDirections, candidate.Direction) {
		return "direction_not_allowed"
	}
	if !contains(context.AssertionModalities, candidate.AssertionModality) {
		return "assertion_modality_invalid"
	}
	if !allEvidenceExists(candidate.EvidenceIDs, evidence) {
		return "evidence_not_in_event"
	}
	if !validConfidence(candidate.ExtractionConfidence) {
		return "confidence_invalid"
	}
	if invalidTimeRange(candidate.ValidFrom, candidate.ValidUntil) ||
		invalidTimeRange(candidate.ForecastPeriodStart, candidate.ForecastPeriodEnd) {
		return "signal_time_invalid"
	}
	if len(candidate.Measurements) > context.MeasurementContract.MaxItemsPerSignal {
		return "measurement_count_invalid"
	}
	for _, measurement := range candidate.Measurements {
		if text := strings.TrimSpace(measurement.Text); text == "" ||
			len([]rune(text)) > context.MeasurementContract.MaxTextCharacters {
			return "measurement_text_invalid"
		}
		if !allEvidenceExists(measurement.EvidenceIDs, evidence) {
			return "evidence_not_in_event"
		}
	}
	return ""
}

func activeEntityType(items []EntityTypeDefinition, typeKey string) (EntityTypeDefinition, bool) {
	for _, item := range items {
		if item.TypeKey == typeKey && item.Status == "active" {
			return item, true
		}
	}
	return EntityTypeDefinition{}, false
}

func mentionGrounded(context Context, candidate EntityLinkCandidate) bool {
	if strings.TrimSpace(candidate.Mention) == "" || len(candidate.EvidenceIDs) == 0 {
		return false
	}
	evidenceByID := indexEvidence(context.Evidence)
	for _, evidenceID := range candidate.EvidenceIDs {
		if _, ok := evidenceByID[evidenceID]; !ok {
			return false
		}
	}
	return true
}

func indexEvidence(items []Evidence) map[string]Evidence {
	result := make(map[string]Evidence, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func indexEntities(items []Entity) map[string]Entity {
	result := make(map[string]Entity, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func indexVariables(items []VariableDefinition) map[string]VariableDefinition {
	result := make(map[string]VariableDefinition, len(items))
	for _, item := range items {
		result[definitionIdentity(item.Key, item.Version)] = item
	}
	return result
}

func allEvidenceExists(ids []string, evidence map[string]Evidence) bool {
	if len(ids) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := evidence[id]; !exists {
			return false
		}
		if _, duplicate := seen[id]; duplicate {
			return false
		}
		seen[id] = struct{}{}
	}
	return true
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func definitionIdentity(key string, version int) string {
	return key + "@" + strconv.Itoa(version)
}

func validConfidence(value string) bool {
	if value == "" {
		return true
	}
	return confidencePattern.MatchString(value)
}

func invalidTimeRange(start, end *time.Time) bool {
	return start != nil && end != nil && end.Before(*start)
}
