package dataclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

type semanticContextLeaseWire eventsemantic.ContextLease
type semanticContextWire eventsemantic.Context
type semanticSubmissionWire eventsemantic.SubmissionResult
type eventSemanticsWire eventsemantic.EventSemantics

type semanticErrorEnvelope struct {
	RequestID string `json:"request_id"`
	Error     struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func (c *Client) ListEligibleEvents(ctx context.Context, limit int, cursor string) (eventsemantic.EligibleEventPage, error) {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(limit))
	query.Set("pagination", "cursor")
	if cursor != "" {
		query.Set("cursor", cursor)
	}
	var response struct {
		Events     []eventsemantic.EligibleEvent `json:"events"`
		NextCursor string                        `json:"next_cursor,omitempty"`
	}
	_, err := c.semanticDo(
		ctx, http.MethodGet, dataAPIPrefix+"/event-semantics/eligible-events?"+query.Encode(), nil,
		&response, "data.v1.listEligibleEventSemanticEvents", "/api/data/v1/event-semantics/eligible-events",
	)
	if err != nil {
		return eventsemantic.EligibleEventPage{}, err
	}
	for _, item := range response.Events {
		if strings.TrimSpace(item.EventID) == "" {
			return eventsemantic.EligibleEventPage{}, invalidSemanticResponse()
		}
	}
	return eventsemantic.EligibleEventPage{Events: response.Events, NextCursor: response.NextCursor}, nil
}

func (c *Client) CreateContextLease(ctx context.Context, request eventsemantic.ContextLeaseRequest) (eventsemantic.ContextLease, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return eventsemantic.ContextLease{}, err
	}
	var response semanticContextLeaseWire
	_, err = c.semanticReplaySafeDo(
		ctx, http.MethodPost, dataAPIPrefix+"/event-semantics/context-leases", payload, &response,
		"data.v1.createEventSemanticContextLease", "/api/data/v1/event-semantics/context-leases",
	)
	result := eventsemantic.ContextLease(response)
	if err == nil && (!validUUID(result.ContextLeaseID) || result.EventID != request.EventID ||
		result.Status != "active" || result.LeaseExpiresAt.IsZero()) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) Context(ctx context.Context, contextLeaseID string) (eventsemantic.Context, error) {
	var response semanticContextWire
	_, err := c.semanticDo(
		ctx, http.MethodGet,
		dataAPIPrefix+"/event-semantics/context-leases/"+url.PathEscape(contextLeaseID)+"/context", nil,
		&response, "data.v1.getEventSemanticContext",
		"/api/data/v1/event-semantics/context-leases/{context_lease_id}/context",
	)
	result := eventsemantic.Context(response)
	if err == nil && !validSemanticContext(result, contextLeaseID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) CreateSubmission(ctx context.Context, request eventsemantic.SubmissionRequest) (eventsemantic.SubmissionResult, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	var response semanticSubmissionWire
	_, err = c.semanticReplaySafeDo(
		ctx, http.MethodPost, dataAPIPrefix+"/event-semantics/submissions", payload, &response,
		"data.v1.createEventSemanticSubmission", "/api/data/v1/event-semantics/submissions",
	)
	result := eventsemantic.SubmissionResult(response)
	if err == nil && !validSemanticSubmission(result) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) SubmitReview(ctx context.Context, submissionID string, request eventsemantic.ReviewRequest) (eventsemantic.SubmissionResult, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	var response semanticSubmissionWire
	_, err = c.semanticReplaySafeDo(
		ctx, http.MethodPost,
		dataAPIPrefix+"/event-semantics/submissions/"+url.PathEscape(submissionID)+"/reviews",
		payload, &response, "data.v1.submitEventSemanticReview",
		"/api/data/v1/event-semantics/submissions/{submission_id}/reviews",
	)
	result := eventsemantic.SubmissionResult(response)
	if err == nil && (!validSemanticSubmission(result) || result.SubmissionID != submissionID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) GetEventSemantics(ctx context.Context, eventID string) (eventsemantic.EventSemantics, error) {
	var response eventSemanticsWire
	_, err := c.semanticDo(
		ctx, http.MethodGet, dataAPIPrefix+"/events/"+url.PathEscape(eventID)+"/semantics", nil,
		&response, "data.v1.getEventSemantics", "/api/data/v1/events/{event_id}/semantics",
	)
	result := eventsemantic.EventSemantics(response)
	if err == nil && !validEventSemantics(result, eventID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) semanticReplaySafeDo(
	ctx context.Context,
	method, path string,
	payload []byte,
	target any,
	operation, pathTemplate string,
) (int, error) {
	ctx, cancel := c.semanticTotalContext(ctx)
	defer cancel()
	var status int
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		status, err = c.semanticDo(ctx, method, path, payload, target, operation, pathTemplate)
		if err == nil {
			return status, nil
		}
		var remote *eventsemantic.RemoteError
		if !errors.As(err, &remote) || !remote.Retryable {
			return status, err
		}
	}
	return status, err
}

func (c *Client) semanticDo(
	ctx context.Context,
	method, path string,
	payload []byte,
	target any,
	operation, pathTemplate string,
) (int, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return 0, &eventsemantic.RemoteError{Code: "data_request_invalid", Summary: "Data Service request is invalid"}
	}
	requestID := uuid.NewString()
	request.Header.Set("Authorization", "Bearer "+c.serviceToken)
	request.Header.Set("X-Request-ID", requestID)
	request.Header.Set("X-Tidewise-Operation", operation)
	request.Header.Set("X-Tidewise-Path-Template", pathTemplate)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return 0, &eventsemantic.RemoteError{Code: "data_transport_unavailable", Summary: "Data Service is unavailable", Retryable: true}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil || int64(len(body)) > c.maxResponseBytes || response.Header.Get("X-Request-ID") != requestID {
		return response.StatusCode, invalidSemanticResponse()
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var public semanticErrorEnvelope
		if decodeSemanticWire(body, &public) != nil || public.RequestID != requestID ||
			strings.TrimSpace(public.Error.Code) == "" || strings.TrimSpace(public.Error.Message) == "" || public.Error.Details == nil {
			return response.StatusCode, invalidSemanticResponse()
		}
		return response.StatusCode, &eventsemantic.RemoteError{
			Status: response.StatusCode, Code: public.Error.Code, Summary: public.Error.Message,
			Retryable: public.Error.Code == "EVENT_SEMANTIC_CONTEXT_DRIFT" ||
				response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500,
		}
	}
	var envelope struct {
		RequestID string          `json:"request_id"`
		Result    json.RawMessage `json:"result"`
	}
	if decodeSemanticWire(body, &envelope) != nil || envelope.RequestID != requestID ||
		decodeSemanticWire(envelope.Result, target) != nil {
		return response.StatusCode, invalidSemanticResponse()
	}
	return response.StatusCode, nil
}

func decodeSemanticWire(payload []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values")
	}
	return nil
}

func (c *Client) semanticTotalContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= c.timeout {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func invalidSemanticResponse() *eventsemantic.RemoteError {
	return &eventsemantic.RemoteError{Code: "data_response_invalid", Summary: "Data Service response contract is invalid"}
}

func validSemanticContext(value eventsemantic.Context, contextLeaseID string) bool {
	if value.ContextLeaseID != contextLeaseID || value.ManifestContractVersion != "event-semantic-context-manifest.v3" ||
		!validUUID(value.ContextLeaseID) || strings.TrimSpace(value.AgentExecutionID) == "" ||
		strings.TrimSpace(value.WorkerID) == "" || !validRFC3339(value.LeaseExpiresAt) ||
		!validSemanticHash(value.ContextFingerprint) || !validSemanticHash(value.EventFingerprint) ||
		!validSemanticHash(value.EvidenceFingerprint) || strings.TrimSpace(value.OntologyVersion) == "" ||
		strings.TrimSpace(value.AcceptancePolicyVersion) == "" || !validUUID(value.Event.ID) ||
		strings.TrimSpace(value.Event.Title) == "" || value.Event.EventStatus != "confirmed" ||
		value.Event.FactStatus != "verified" || len(value.Evidence) == 0 ||
		len(value.EntityTypeDefinitions) == 0 || len(value.VariableDefinitions) == 0 ||
		len(value.AssertionModalities) == 0 || value.MeasurementContract.Representation != "evidence_grounded_narrative" ||
		value.MeasurementContract.MaxItemsPerSignal < 1 || value.MeasurementContract.MaxTextCharacters < 1 ||
		!value.MeasurementContract.RequiresEvidenceIDs || value.MeasurementContract.NumericValidation {
		return false
	}
	evidenceIDs := make(map[string]struct{}, len(value.Evidence))
	for _, evidence := range value.Evidence {
		if !validUUID(evidence.EvidenceID) || !validUUID(evidence.RawDocumentID) ||
			!validSemanticHash(evidence.EvidenceHash) || strings.TrimSpace(evidence.Excerpt) == "" ||
			strings.TrimSpace(evidence.SourceName) == "" || strings.TrimSpace(evidence.SourceType) == "" ||
			strings.TrimSpace(evidence.Title) == "" || !validRFC3339(evidence.FirstSeenAt) ||
			!validRFC3339(evidence.KnowledgeAvailableAt) || !validRFC3339(evidence.AcceptedAt) ||
			duplicateString(evidenceIDs, evidence.EvidenceID) {
			return false
		}
	}
	entityTypes := make(map[string]struct{}, len(value.EntityTypeDefinitions))
	for _, definition := range value.EntityTypeDefinitions {
		if strings.TrimSpace(definition.TypeKey) == "" || definition.Version < 1 ||
			strings.TrimSpace(definition.NameZH) == "" || strings.TrimSpace(definition.NameEN) == "" ||
			strings.TrimSpace(definition.BusinessDefinition) == "" ||
			!validNonblankSet(definition.InclusionCriteria) ||
			!validNonblankSet(definition.ExclusionCriteria) ||
			definition.Status != "active" || !validNonblankSet(definition.AllowedEventRoles) ||
			duplicateString(entityTypes, definition.TypeKey) {
			return false
		}
	}
	variables := make(map[string]struct{}, len(value.VariableDefinitions))
	for _, definition := range value.VariableDefinitions {
		identity := definition.Key + "\x00" + strconv.Itoa(definition.Version)
		if strings.TrimSpace(definition.Key) == "" || definition.Version < 1 ||
			strings.TrimSpace(definition.NameZH) == "" || strings.TrimSpace(definition.NameEN) == "" ||
			strings.TrimSpace(definition.Domain) == "" || strings.TrimSpace(definition.BusinessDefinition) == "" ||
			strings.TrimSpace(definition.ValueType) == "" || definition.Status != "active" ||
			!validNonblankSet(definition.AllowedDirections) ||
			!validNonblankSet(definition.ApplicableEntityTypes) ||
			!allMembers(definition.ApplicableEntityTypes, entityTypes) ||
			!validOptionalNonblankSet(definition.AllowedUnits) || duplicateString(variables, identity) {
			return false
		}
	}
	return validNonblankSet(value.AssertionModalities)
}

func validSemanticSubmission(value eventsemantic.SubmissionResult) bool {
	if !validUUID(value.SubmissionID) || !validUUID(value.EventID) ||
		!validSemanticHash(value.CanonicalPayloadHash) || !semanticStatus(value.Status) {
		return false
	}
	for _, group := range [][]eventsemantic.CandidateDecision{value.EntityLinks, value.VariableSignals} {
		for _, item := range group {
			if strings.TrimSpace(item.CandidateKey) == "" || !semanticStatus(item.Status) ||
				(item.RecordID != "" && !validUUID(item.RecordID)) {
				return false
			}
		}
	}
	if !validReviewerWorkPackage(value.ReviewerWorkPackage, true) ||
		!validReviewerWorkPackage(value.AuditWorkPackage, false) {
		return false
	}
	return true
}

func validReviewerWorkPackage(work *eventsemantic.ReviewerWorkPackage, requireResolvedEntities bool) bool {
	if work == nil {
		return true
	}
	entities := make(map[string]struct{}, len(work.ResolvedEntities))
	for _, entity := range work.ResolvedEntities {
		if !validUUID(entity.EntityID) || strings.TrimSpace(entity.EntityType) == "" ||
			strings.TrimSpace(entity.CanonicalName) == "" || entity.Status != "active" {
			return false
		}
		entities[entity.EntityID] = struct{}{}
	}
	links := make(map[string]struct{}, len(work.EntityLinks))
	for _, link := range work.EntityLinks {
		if strings.TrimSpace(link.CandidateKey) == "" || strings.TrimSpace(link.Mention) == "" ||
			!validUUID(link.EntityID) || !validUUIDSet(link.EvidenceIDs) {
			return false
		}
		if requireResolvedEntities {
			if _, ok := entities[link.EntityID]; !ok {
				return false
			}
		}
		if duplicateString(links, link.CandidateKey) {
			return false
		}
	}
	for _, signal := range work.VariableSignals {
		if strings.TrimSpace(signal.CandidateKey) == "" || strings.TrimSpace(signal.SubjectLinkKey) == "" ||
			strings.TrimSpace(signal.VariableKey) == "" || signal.VariableVersion < 1 ||
			strings.TrimSpace(signal.Direction) == "" || strings.TrimSpace(signal.AssertionModality) == "" ||
			!validUUIDSet(signal.EvidenceIDs) {
			return false
		}
		if _, ok := links[signal.SubjectLinkKey]; !ok {
			return false
		}
		for _, measurement := range signal.Measurements {
			if strings.TrimSpace(measurement.MeasurementText) == "" || !validUUIDSet(measurement.EvidenceIDs) {
				return false
			}
		}
	}
	return true
}

func validEventSemantics(value eventsemantic.EventSemantics, eventID string) bool {
	if !validUUID(eventID) || value.EventID != eventID {
		return false
	}
	for _, submission := range value.Submissions {
		if submission.EventID != eventID || !validSemanticSubmission(submission) {
			return false
		}
	}
	return true
}

func validSemanticHash(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func validRFC3339(value string) bool {
	_, err := time.Parse(time.RFC3339Nano, value)
	return err == nil
}

func validNonblankSet(values []string) bool {
	return len(values) > 0 && validOptionalNonblankSet(values)
}

func validOptionalNonblankSet(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || duplicateString(seen, value) {
			return false
		}
	}
	return true
}

func allMembers(values []string, allowed map[string]struct{}) bool {
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return false
		}
	}
	return true
}

func validUUIDSet(values []string) bool {
	if len(values) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validUUID(value) || duplicateString(seen, value) {
			return false
		}
	}
	return true
}

func duplicateString(values map[string]struct{}, value string) bool {
	if _, ok := values[value]; ok {
		return true
	}
	values[value] = struct{}{}
	return false
}

func semanticStatus(value string) bool {
	switch value {
	case "pending_review", "needs_reanalysis", "quarantined", "accepted", "rejected", "superseded":
		return true
	default:
		return false
	}
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
