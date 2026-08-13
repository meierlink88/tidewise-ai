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
	GetEventSemantics(context.Context, string) (EventSemanticsResult, error)
	ListResearchSemantics(context.Context, ResearchSemanticQuery) ([]ResearchSemanticRecord, error)
	ResearchSemanticClosure(context.Context, ResearchSemanticClosureQuery) (ResearchSemanticDictionaries, error)
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
	store   Store
	now     func() time.Time
	newUUID func() string
}

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Event Semantic store is required")
	}
	return &UseCase{
		store:   store,
		now:     func() time.Time { return time.Now().UTC() },
		newUUID: uuid.NewString,
	}, nil
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

func (s *UseCase) ListResearchSemantics(ctx context.Context, query ResearchSemanticQuery) ([]ResearchSemanticRecord, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("Event Semantic store is required")
	}
	return s.store.ListResearchSemantics(ctx, query)
}

func (s *UseCase) ResearchSemanticClosure(ctx context.Context, query ResearchSemanticClosureQuery) (ResearchSemanticDictionaries, error) {
	if s == nil || s.store == nil {
		return ResearchSemanticDictionaries{}, errors.New("Event Semantic store is required")
	}
	return s.store.ResearchSemanticClosure(ctx, query)
}

type ResearchSemanticQuery struct {
	EventIDs                                 []string
	DiscoveryWindowStart, DiscoveryWindowEnd time.Time
	AnalysisAsOf                             time.Time
}

type ResearchSemanticRecord struct {
	EventID         string                   `json:"-"`
	EntityLinks     []ResearchEntityLink     `json:"entity_links"`
	VariableSignals []ResearchVariableSignal `json:"variable_signals"`
}

type ResearchVersionedReference struct {
	Key     string
	Version int
}

type ResearchSemanticClosureQuery struct {
	AnalysisAsOf            time.Time
	VariableDefinitions     []ResearchVersionedReference
	DirectTransmissionRules []ResearchVersionedReference
	SemanticSubmissionIDs   []string
}

type ResearchSemanticDictionaries struct {
	VariableDefinitions     []ResearchVariableDefinition     `json:"variable_definitions"`
	DirectTransmissionRules []ResearchDirectTransmissionRule `json:"direct_transmission_rules"`
	AcceptancePolicies      []ResearchAcceptancePolicy       `json:"acceptance_policies"`
}

type ResearchEntityLink struct {
	EventEntityLinkID    string   `json:"event_entity_link_id"`
	SemanticSubmissionID string   `json:"semantic_submission_id"`
	EntityID             string   `json:"entity_id"`
	EntityRole           string   `json:"entity_role"`
	ResolvedMention      *string  `json:"resolved_mention"`
	ResolutionMethod     *string  `json:"resolution_method"`
	ResolutionConfidence *float64 `json:"resolution_confidence"`
	EvidenceIDs          []string `json:"evidence_ids"`
	ReviewStatus         string   `json:"review_status"`
}

type ResearchVariableSignal struct {
	VariableSignalID         string                 `json:"variable_signal_id"`
	SemanticSubmissionID     string                 `json:"semantic_submission_id"`
	SourceEventID            string                 `json:"source_event_id"`
	SubjectEventEntityLinkID string                 `json:"subject_event_entity_link_id"`
	SubjectEntityID          string                 `json:"subject_entity_id"`
	VariableKey              string                 `json:"variable_key"`
	VariableVersion          int                    `json:"variable_version"`
	Direction                string                 `json:"direction"`
	AssertionModality        string                 `json:"assertion_modality"`
	EvidenceIDs              []string               `json:"evidence_ids"`
	StatementAt              *time.Time             `json:"statement_at"`
	ValidFrom                *time.Time             `json:"valid_from"`
	ValidUntil               *time.Time             `json:"valid_until"`
	ForecastPeriodStart      *time.Time             `json:"forecast_period_start"`
	ForecastPeriodEnd        *time.Time             `json:"forecast_period_end"`
	ExtractionConfidence     *float64               `json:"extraction_confidence"`
	ReviewStatus             string                 `json:"review_status"`
	Measurements             []ResearchMeasurement  `json:"measurements"`
	DirectImpacts            []ResearchDirectImpact `json:"direct_impacts"`
}

type ResearchMeasurement struct {
	MeasurementID    string  `json:"measurement_id"`
	MeasurementRole  string  `json:"measurement_role"`
	ValueShape       string  `json:"value_shape"`
	RawValue         *string `json:"raw_value"`
	RawLower         *string `json:"raw_lower"`
	RawUpper         *string `json:"raw_upper"`
	RawUnit          *string `json:"raw_unit"`
	CanonicalValue   *string `json:"canonical_value"`
	CanonicalLower   *string `json:"canonical_lower"`
	CanonicalUpper   *string `json:"canonical_upper"`
	CanonicalUnit    *string `json:"canonical_unit"`
	Currency         *string `json:"currency"`
	Scale            *string `json:"scale"`
	ComparisonBasis  *string `json:"comparison_basis"`
	ComparisonPeriod *string `json:"comparison_period"`
	RawText          string  `json:"raw_text"`
	IsApproximate    bool    `json:"is_approximate"`
	EvidenceID       string  `json:"evidence_id"`
}

type ResearchDirectImpact struct {
	DirectImpactAssertionID string     `json:"direct_impact_assertion_id"`
	SemanticSubmissionID    string     `json:"semantic_submission_id"`
	SourceVariableSignalID  string     `json:"source_variable_signal_id"`
	TargetEntityID          string     `json:"target_entity_id"`
	AffectedVariableKey     string     `json:"affected_variable_key"`
	AffectedVariableVersion int        `json:"affected_variable_version"`
	AffectedDirection       string     `json:"affected_direction"`
	DerivationType          string     `json:"derivation_type"`
	MechanismSummary        string     `json:"mechanism_summary"`
	EvidenceIDs             []string   `json:"evidence_ids"`
	EntityRelationID        *string    `json:"entity_relation_id"`
	RuleKey                 *string    `json:"rule_key"`
	RuleVersion             *int       `json:"rule_version"`
	AssertionConfidence     *float64   `json:"assertion_confidence"`
	EffectiveFrom           *time.Time `json:"effective_from"`
	EffectiveTo             *time.Time `json:"effective_to"`
	ReviewStatus            string     `json:"review_status"`
}

type ResearchVariableDefinition struct {
	Key                   string   `json:"key"`
	Version               int      `json:"version"`
	NameZH                string   `json:"name_zh"`
	NameEN                string   `json:"name_en"`
	Domain                string   `json:"domain"`
	BusinessDefinition    string   `json:"business_definition"`
	ValueType             string   `json:"value_type"`
	AllowedDirections     []string `json:"allowed_directions"`
	CanonicalUnit         *string  `json:"canonical_unit"`
	Status                string   `json:"status"`
	ApplicableEntityTypes []string `json:"applicable_entity_types"`
}
type ResearchDirectTransmissionRule struct {
	RuleKey                 string `json:"rule_key"`
	Version                 int    `json:"version"`
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
	Status                  string `json:"status"`
}
type ResearchAcceptancePolicy struct {
	PolicyKey   string          `json:"policy_key"`
	Version     int             `json:"version"`
	RetryBudget int             `json:"retry_budget"`
	Status      string          `json:"status"`
	Policy      json.RawMessage `json:"policy"`
}

var ErrResearchHistoricalSemanticsUnavailable = errors.New("strict historical Event semantics are unavailable because a selected Event was superseded after analysis_as_of")
var ErrResearchReferenceClosureInconsistent = errors.New("Research Analysis Context reference closure is inconsistent; restart from the first page")

const (
	ResearchMaxBundleBytes     = 512 * 1024
	ResearchMaxDictionaryBytes = 4 * 1024 * 1024
	ResearchMaxBundleRows      = 1_000
	ResearchMaxDictionaryRows  = 50_000
)

type ResearchResourceLimitError struct {
	Reason        string
	Component     string
	ActualRows    *int64
	MaxRows       *int64
	ActualBytes   *int64
	MaxBytes      *int64
	RetryGuidance string
}

func (e *ResearchResourceLimitError) Error() string { return e.Reason }

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
	var result ContextLease
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		now := s.now().UTC()
		state, err := tx.LoadContextLeaseState(ctx, request, now)
		if err != nil {
			return err
		}
		if state.Existing != nil {
			existing := state.Existing
			if existing.EventID != request.EventID ||
				existing.AgentExecutionID != request.AgentExecutionID ||
				existing.WorkerID != request.WorkerID ||
				existing.SupersedesSubmissionID != request.SupersedesSubmissionID {
				return &ConflictError{Reason: "agent_execution_id is bound to a different Context Lease identity"}
			}
			if existing.SubmissionStatus != "" &&
				existing.SubmissionStatus != StatusPendingReview &&
				existing.SubmissionStatus != StatusNeedsReanalysis {
				return &ConflictError{Reason: "agent_execution_id already reached a terminal Semantic Submission"}
			}
			result = existing.ContextLease
			result.Status = "active"
			result.LeaseExpiresAt = now.Add(request.Lease)
			return tx.SaveContextLease(ctx, ContextLeaseWrite{
				Lease: result, AgentExecutionID: request.AgentExecutionID,
				WorkerID: request.WorkerID, ExpireLeaseIDs: state.ExpiredLeaseIDs,
				Refresh: true, TransitionedAt: now,
			})
		}
		if !state.Event.Found {
			return &NotFoundError{Resource: "Event"}
		}
		if state.Event.EventStatus != "confirmed" || state.Event.FactStatus != "verified" {
			return &NotRequiredError{Reason: "Event no longer requires initial Semantic processing"}
		}
		if !state.Event.InputValid {
			return &InputInvalidError{Reason: "Event does not satisfy the Event Semantic input contract"}
		}
		if state.ActiveLeaseID != "" {
			canReplacePriorLease := request.SupersedesSubmissionID != "" &&
				state.SupersededSubmission != nil &&
				state.ActiveLeaseID == state.SupersededSubmission.ContextLeaseID
			if !canReplacePriorLease {
				return &ConflictError{Reason: "Event already has an active Context Lease"}
			}
		}
		if request.SupersedesSubmissionID == "" {
			if state.HasActiveSubmission {
				return &ConflictError{Reason: "Event already has an active Semantic Submission"}
			}
		} else {
			prior := state.SupersededSubmission
			if prior == nil {
				return &NotFoundError{Resource: "superseded Event Semantic Submission"}
			}
			if prior.EventID != request.EventID ||
				(prior.Status != StatusNeedsReanalysis && prior.Status != StatusAccepted &&
					prior.Status != StatusRejected && prior.Status != StatusQuarantined) {
				return &ConflictError{Reason: "supersedes_submission_id is not an active terminal Submission for this Event"}
			}
		}
		result = ContextLease{
			ID: s.newUUID(), EventID: request.EventID,
			SupersedesSubmissionID: request.SupersedesSubmissionID,
			Status:                 "active", LeaseExpiresAt: now.Add(request.Lease),
		}
		return tx.SaveContextLease(ctx, ContextLeaseWrite{
			Lease: result, AgentExecutionID: request.AgentExecutionID,
			WorkerID: request.WorkerID, ExpireLeaseIDs: state.ExpiredLeaseIDs,
			ConsumeSupersededLease: request.SupersedesSubmissionID != "", TransitionedAt: now,
		})
	})
	if err != nil {
		return ContextLease{}, err
	}
	return result, nil
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
	var result SubmissionResult
	err = s.store.InTransaction(ctx, func(tx Transaction) error {
		observedAt := s.now().UTC()
		state, err := tx.LoadSubmissionState(ctx, submission)
		if err != nil {
			return err
		}
		if state.Existing != nil {
			if state.Existing.CanonicalPayloadHash != hash {
				return &ConflictError{Reason: "agent_execution_id is bound to a different canonical payload"}
			}
			result = *state.Existing
			result.Replayed = true
			return nil
		}
		if !state.Lease.Found || !state.Lease.LeaseExpiresAt.After(observedAt) {
			return &NotFoundError{Resource: "Event Semantic Context Lease"}
		}
		if state.Lease.EventID != submission.EventID || state.Lease.Status != "active" {
			return &ConflictError{Reason: "context lease is not active for this Event"}
		}
		if state.Lease.AgentExecutionID != submission.AgentExecutionID {
			return &ConflictError{Reason: "Submission agent_execution_id differs from its Context Lease"}
		}
		if state.Lease.SupersedesSubmissionID != submission.SupersedesSubmissionID {
			return &ConflictError{Reason: "Submission supersedes identity differs from its Context Lease"}
		}
		if state.Context.Event.ID != submission.EventID {
			return &ConflictError{Reason: "context lease is bound to a different Event"}
		}
		if submission.OntologyVersion != state.Context.OntologyVersion ||
			submission.AcceptancePolicyVersion != state.Context.PolicyVersion {
			return &ConflictError{Reason: "ontology or acceptance policy snapshot changed"}
		}
		if submission.SupersedesSubmissionID != "" {
			prior := state.SupersededSubmission
			if prior == nil {
				return &NotFoundError{Resource: "superseded Event Semantic Submission"}
			}
			if prior.EventID != submission.EventID || prior.Status == StatusSuperseded {
				return &ConflictError{Reason: "supersedes_submission_id must reference the current Event's active prior Submission"}
			}
		}
		precheck := Precheck(state.Context, submission)
		status := SummarizeSubmission(precheck)
		transitionedAt := observedAt
		result = SubmissionResult{
			SubmissionID: s.newUUID(), EventID: submission.EventID, Status: status,
			CanonicalPayloadHash: hash, Precheck: precheck,
		}
		return tx.SaveSubmission(ctx, SubmissionWrite{
			SubmissionID: result.SubmissionID, SnapshotID: s.newUUID(), Submission: submission,
			Payload: append(json.RawMessage(nil), payload...), PayloadHash: hash,
			Precheck: precheck, Status: status,
			ConsumeLease:   status != StatusPendingReview && status != StatusNeedsReanalysis,
			TransitionedAt: transitionedAt,
		})
	})
	if err != nil {
		return SubmissionResult{}, err
	}
	return result, nil
}

func (s *UseCase) SubmitReview(ctx context.Context, submission ReviewSubmission) (SubmissionResult, error) {
	if strings.TrimSpace(submission.SubmissionID) == "" || strings.TrimSpace(submission.ReviewerExecutionKey) == "" ||
		!validHash(submission.PromptHash) || strings.TrimSpace(submission.Model) == "" ||
		len(submission.Items) == 0 {
		return SubmissionResult{}, &ValidationError{Reason: "review identity and items are invalid"}
	}
	for _, item := range submission.Items {
		if !containsCandidateType([]CandidateType{CandidateTypeEntityLink, CandidateTypeVariableSignal}, item.CandidateType) ||
			strings.TrimSpace(item.CandidateKey) == "" ||
			!containsReviewDecision([]ReviewDecision{ReviewDecisionPass, ReviewDecisionFail, ReviewDecisionIndeterminate}, item.Decision) ||
			len(item.EvidenceIDs) == 0 {
			return SubmissionResult{}, &ValidationError{Reason: "review item is invalid"}
		}
	}
	payload, hash, err := canonicalHash(submission)
	if err != nil {
		return SubmissionResult{}, err
	}
	var result SubmissionResult
	err = s.store.InTransaction(ctx, func(tx Transaction) error {
		state, err := tx.LoadReviewState(ctx, submission)
		if err != nil {
			return err
		}
		if !state.Found {
			return &NotFoundError{Resource: "Event Semantic Submission"}
		}
		if !state.Identity.Matches(submission) {
			return &ConflictError{Reason: "review prompt or model does not match the frozen Submission identity"}
		}
		if state.Submission == nil {
			return &NotFoundError{Resource: "Event Semantic Submission"}
		}
		if state.ExistingSnapshot != nil {
			if state.ExistingSnapshot.CanonicalPayloadHash != hash {
				return &ConflictError{Reason: "reviewer_execution_key is bound to a different payload"}
			}
			result = *state.Submission
			result.Replayed = true
			return nil
		}
		result = *state.Submission
		if result.Status != StatusPendingReview && result.Status != StatusNeedsReanalysis {
			return &ConflictError{Reason: "Event Semantic Submission is not reviewable"}
		}
		if err := ApplyReview(&result.Precheck, submission, state.ReviewCount+1 > state.RetryBudget); err != nil {
			return err
		}
		status := SummarizeSubmission(result.Precheck)
		result.Status = status
		terminal := status == StatusAccepted || status == StatusRejected || status == StatusQuarantined
		transitionedAt := s.now().UTC()
		var finalizedAt *time.Time
		if terminal {
			finalizedAt = &transitionedAt
		}
		return tx.SaveReview(ctx, ReviewWrite{
			SnapshotID: s.newUUID(), Submission: submission,
			Payload: append(json.RawMessage(nil), payload...), PayloadHash: hash,
			Precheck: result.Precheck, Status: status,
			SupersedePrior: status == StatusAccepted, ConsumeLease: terminal,
			FinalizedAt: finalizedAt, TransitionedAt: transitionedAt,
		})
	})
	if err != nil {
		return SubmissionResult{}, err
	}
	return result, nil
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

type CandidateType string

const (
	CandidateTypeEntityLink     CandidateType = "entity_link"
	CandidateTypeVariableSignal CandidateType = "variable_signal"
	CandidateTypeDirectImpact   CandidateType = "direct_impact"
)

type ReviewDecision string

const (
	ReviewDecisionPass          ReviewDecision = "pass"
	ReviewDecisionFail          ReviewDecision = "fail"
	ReviewDecisionIndeterminate ReviewDecision = "indeterminate"
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
// The Context API hydrates the bounded Event, Evidence and Variable payload from these references.
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
	CandidateType CandidateType
	CandidateKey  string
	Decision      ReviewDecision
	ReasonCodes   []string
	EvidenceIDs   []string
}

func containsCandidateType(values []CandidateType, target CandidateType) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsReviewDecision(values []ReviewDecision, target ReviewDecision) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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

func ApplyReview(precheck *PrecheckResult, submission ReviewSubmission, quarantineIndeterminate bool) error {
	pending := map[string]*CandidateDecision{}
	candidateEvidence := make(map[string][]string)
	register := func(kind CandidateType, items []CandidateDecision) {
		for index := range items {
			if items[index].Status == StatusPendingReview || items[index].Status == StatusNeedsReanalysis {
				pending[string(kind)+":"+items[index].CandidateKey] = &items[index]
			}
		}
	}
	register(CandidateTypeEntityLink, precheck.EntityLinks)
	register(CandidateTypeVariableSignal, precheck.VariableSignals)
	register(CandidateTypeDirectImpact, precheck.DirectImpacts)
	for _, candidate := range precheck.ReviewerWorkPackage.EntityLinks {
		candidateEvidence[string(CandidateTypeEntityLink)+":"+candidate.Key] = candidate.EvidenceIDs
	}
	for _, candidate := range precheck.ReviewerWorkPackage.VariableSignals {
		candidateEvidence[string(CandidateTypeVariableSignal)+":"+candidate.Key] = candidate.EvidenceIDs
	}
	for _, candidate := range precheck.ReviewerWorkPackage.DirectImpacts {
		candidateEvidence[string(CandidateTypeDirectImpact)+":"+candidate.Key] = candidate.EvidenceIDs
	}
	if len(submission.Items) != len(pending) {
		return &ValidationError{Reason: "review must decide every reviewable candidate exactly once"}
	}
	seen := make(map[string]struct{}, len(submission.Items))
	for _, item := range submission.Items {
		identity := string(item.CandidateType) + ":" + item.CandidateKey
		decision, exists := pending[identity]
		if !exists {
			return &ConflictError{Reason: "review references a non-reviewable candidate"}
		}
		if _, duplicate := seen[identity]; duplicate {
			return &ValidationError{Reason: "review candidate identities must be unique"}
		}
		if !reviewEvidenceMatchesCandidate(item.EvidenceIDs, candidateEvidence[identity]) {
			return &ValidationError{Reason: "review Evidence must cite the candidate Event Evidence"}
		}
		seen[identity] = struct{}{}
		switch item.Decision {
		case ReviewDecisionPass:
			decision.Status, decision.ReasonCode = StatusAccepted, ""
		case ReviewDecisionFail:
			decision.Status, decision.ReasonCode = StatusRejected, firstReviewReason(item.ReasonCodes, "reviewer_failed")
		case ReviewDecisionIndeterminate:
			if quarantineIndeterminate {
				decision.Status, decision.ReasonCode = StatusQuarantined, "unresolved_after_retry_budget"
			} else {
				decision.Status, decision.ReasonCode = StatusNeedsReanalysis, firstReviewReason(item.ReasonCodes, "reviewer_indeterminate")
			}
		}
	}
	propagateReview(precheck)
	return nil
}

func reviewEvidenceMatchesCandidate(reviewed, candidate []string) bool {
	if len(reviewed) == 0 || len(candidate) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(candidate))
	for _, evidenceID := range candidate {
		allowed[evidenceID] = struct{}{}
	}
	for _, evidenceID := range reviewed {
		if _, exists := allowed[evidenceID]; !exists {
			return false
		}
	}
	return true
}

func firstReviewReason(reasons []string, fallback string) string {
	for _, reason := range reasons {
		if strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason)
		}
	}
	return fallback
}

func propagateReview(precheck *PrecheckResult) {
	linkStatus := make(map[string]CandidateDecision, len(precheck.EntityLinks))
	for _, item := range precheck.EntityLinks {
		linkStatus[item.CandidateKey] = item
	}
	signalStatus := make(map[string]CandidateDecision, len(precheck.VariableSignals))
	signalByKey := make(map[string]VariableSignalCandidate, len(precheck.ReviewerWorkPackage.VariableSignals))
	for _, item := range precheck.ReviewerWorkPackage.VariableSignals {
		signalByKey[item.Key] = item
	}
	for index := range precheck.VariableSignals {
		candidate := signalByKey[precheck.VariableSignals[index].CandidateKey]
		upstream := linkStatus[candidate.SubjectLinkKey]
		switch upstream.Status {
		case StatusRejected:
			precheck.VariableSignals[index].Status = StatusRejected
			precheck.VariableSignals[index].ReasonCode = "upstream_rejected"
		case StatusQuarantined:
			precheck.VariableSignals[index].Status = StatusQuarantined
			precheck.VariableSignals[index].ReasonCode = "upstream_quarantined"
		case StatusNeedsReanalysis:
			precheck.VariableSignals[index].Status = StatusNeedsReanalysis
			precheck.VariableSignals[index].ReasonCode = "upstream_pending"
		}
		signalStatus[precheck.VariableSignals[index].CandidateKey] = precheck.VariableSignals[index]
	}
	impactByKey := make(map[string]DirectImpactCandidate, len(precheck.ReviewerWorkPackage.DirectImpacts))
	for _, item := range precheck.ReviewerWorkPackage.DirectImpacts {
		impactByKey[item.Key] = item
	}
	for index := range precheck.DirectImpacts {
		candidate := impactByKey[precheck.DirectImpacts[index].CandidateKey]
		upstream := signalStatus[candidate.SourceSignalKey]
		switch upstream.Status {
		case StatusRejected:
			precheck.DirectImpacts[index].Status = StatusRejected
			precheck.DirectImpacts[index].ReasonCode = "upstream_rejected"
		case StatusQuarantined:
			precheck.DirectImpacts[index].Status = StatusQuarantined
			precheck.DirectImpacts[index].ReasonCode = "upstream_quarantined"
		case StatusNeedsReanalysis:
			precheck.DirectImpacts[index].Status = StatusNeedsReanalysis
			precheck.DirectImpacts[index].ReasonCode = "upstream_pending"
		}
	}
}

func SummarizeSubmission(precheck PrecheckResult) ReviewStatus {
	hasAccepted, hasPending, hasNeeds, hasQuarantined := false, false, false, false
	for _, group := range [][]CandidateDecision{precheck.EntityLinks, precheck.VariableSignals, precheck.DirectImpacts} {
		for _, item := range group {
			hasAccepted = hasAccepted || item.Status == StatusAccepted
			hasPending = hasPending || item.Status == StatusPendingReview
			hasNeeds = hasNeeds || item.Status == StatusNeedsReanalysis
			hasQuarantined = hasQuarantined || item.Status == StatusQuarantined
		}
	}
	switch {
	case hasPending:
		return StatusPendingReview
	case hasNeeds:
		return StatusNeedsReanalysis
	case hasAccepted:
		return StatusAccepted
	case hasQuarantined:
		return StatusQuarantined
	default:
		return StatusRejected
	}
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
	if !contains(allowedEntityRoles(), candidate.EntityRole) {
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

func allowedEntityRoles() []string {
	return []string{"event_subject", "actor", "affected_entity", "statement_source", "event_object", "context"}
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
