package eventsemantics

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
	"strings"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	ListEligibleEvents(context.Context, int, *EligibleEventCursor) ([]EligibleEvent, error)
	CreateContextLease(context.Context, ContextLeaseRequest) (ContextLease, error)
	Context(context.Context, string) (Context, error)
	Resolve(context.Context, string, []EntityMention) ([]EntityResolution, error)
	SearchDirectTargets(context.Context, string, string, []string) ([]DirectTarget, error)
	ReplaySubmission(context.Context, string, string) (SubmissionResult, bool, error)
	CreateSubmission(context.Context, Submission, PrecheckResult, []byte, string) (SubmissionResult, error)
	SubmitReview(context.Context, ReviewSubmission, []byte, string) (SubmissionResult, error)
	GetEventSemantics(context.Context, string) (EventSemanticsResult, error)
}

type NotFoundError struct{ Resource string }

func (e *NotFoundError) Error() string { return e.Resource + " not found" }

type ConflictError struct{ Reason string }

func (e *ConflictError) Error() string { return e.Reason }

type ValidationError struct{ Reason string }

func (e *ValidationError) Error() string { return e.Reason }

type NotRequiredError struct{ Reason string }

func (e *NotRequiredError) Error() string { return e.Reason }

type InputInvalidError struct{ Reason string }

func (e *InputInvalidError) Error() string { return e.Reason }

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListEligibleEvents(
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

func (s *Service) CreateContextLease(ctx context.Context, request ContextLeaseRequest) (ContextLease, error) {
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

func (s *Service) Context(ctx context.Context, contextLeaseID string) (Context, error) {
	if strings.TrimSpace(contextLeaseID) == "" {
		return Context{}, &ValidationError{Reason: "context_lease_id is required"}
	}
	return s.store.Context(ctx, contextLeaseID)
}

func (s *Service) Resolve(ctx context.Context, contextLeaseID string, mentions []EntityMention) ([]EntityResolution, error) {
	if strings.TrimSpace(contextLeaseID) == "" || len(mentions) == 0 || len(mentions) > 20 {
		return nil, &ValidationError{Reason: "context_lease_id and one to twenty mentions are required"}
	}
	for _, mention := range mentions {
		if strings.TrimSpace(mention.Mention) == "" || len(mention.AllowedEntityTypes) == 0 {
			return nil, &ValidationError{Reason: "mention and allowed_entity_types are required"}
		}
	}
	return s.store.Resolve(ctx, contextLeaseID, mentions)
}

func (s *Service) SearchDirectTargets(
	ctx context.Context,
	contextLeaseID string,
	subjectEntityID string,
	allowedTargetTypes []string,
) ([]DirectTarget, error) {
	if strings.TrimSpace(contextLeaseID) == "" || strings.TrimSpace(subjectEntityID) == "" ||
		len(allowedTargetTypes) == 0 || len(allowedTargetTypes) > 5 {
		return nil, &ValidationError{Reason: "context_lease_id, subject_entity_id and bounded target types are required"}
	}
	return s.store.SearchDirectTargets(ctx, contextLeaseID, subjectEntityID, allowedTargetTypes)
}

func (s *Service) CreateSubmission(ctx context.Context, submission Submission) (SubmissionResult, error) {
	if strings.TrimSpace(submission.ContextLeaseID) == "" || strings.TrimSpace(submission.EventID) == "" ||
		strings.TrimSpace(submission.AgentExecutionID) == "" ||
		submission.AgentKey != "event-semantic-enricher" ||
		submission.AgentVersion != "event-semantic-enricher.v1" {
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
	contextSnapshot, err := s.store.Context(ctx, submission.ContextLeaseID)
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

func (s *Service) SubmitReview(ctx context.Context, submission ReviewSubmission) (SubmissionResult, error) {
	if strings.TrimSpace(submission.SubmissionID) == "" || strings.TrimSpace(submission.ReviewerExecutionKey) == "" ||
		!validHash(submission.PromptHash) || strings.TrimSpace(submission.Model) == "" ||
		len(submission.Items) == 0 {
		return SubmissionResult{}, &ValidationError{Reason: "review identity and items are invalid"}
	}
	for _, item := range submission.Items {
		if !contains([]string{"entity_link", "variable_signal", "direct_impact"}, item.CandidateType) ||
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

func (s *Service) Get(ctx context.Context, eventID string) (EventSemanticsResult, error) {
	if strings.TrimSpace(eventID) == "" {
		return EventSemanticsResult{}, &ValidationError{Reason: "event_id is required"}
	}
	return s.store.GetEventSemantics(ctx, eventID)
}

func validateSubmissionMetadata(submission Submission) error {
	for _, hash := range []string{
		submission.GeneratorPromptHash,
		submission.ReviewerPromptHash,
	} {
		if !validHash(hash) {
			return &ValidationError{Reason: "prompt hashes must be lowercase SHA-256"}
		}
	}
	if submission.AdjudicatorPromptHash != "" && !validHash(submission.AdjudicatorPromptHash) {
		return &ValidationError{Reason: "adjudicator prompt hash must be lowercase SHA-256"}
	}
	if strings.TrimSpace(submission.GeneratorModel) == "" ||
		strings.TrimSpace(submission.ReviewerModel) == "" ||
		strings.TrimSpace(submission.OntologyVersion) == "" ||
		strings.TrimSpace(submission.AcceptancePolicyVersion) == "" {
		return &ValidationError{Reason: "submission version snapshots are required"}
	}
	for _, keys := range [][]string{
		entityLinkKeys(submission.EntityLinks),
		variableSignalKeys(submission.VariableSignals),
		directImpactKeys(submission.DirectImpacts),
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

func directImpactKeys(items []DirectImpactCandidate) []string {
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
