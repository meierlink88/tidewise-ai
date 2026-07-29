package dataclient

import (
	"bytes"
	"context"
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
type semanticEligibleEventWire eventsemantic.EligibleEvent
type semanticContextWire eventsemantic.Context
type semanticResolutionWire eventsemantic.EntityResolution
type semanticTargetWire eventsemantic.DirectTarget
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

func (c *Client) ListEligibleEvents(
	ctx context.Context,
	limit int,
) ([]eventsemantic.EligibleEvent, error) {
	var response struct {
		Events []semanticEligibleEventWire `json:"events"`
	}
	_, err := c.semanticDo(
		ctx, http.MethodGet,
		dataAPIPrefix+"/event-semantics/eligible-events?limit="+strconv.Itoa(limit),
		nil, &response,
		"data.v1.listEligibleEventSemanticEvents",
		"/api/data/v1/event-semantics/eligible-events",
	)
	result := make([]eventsemantic.EligibleEvent, 0, len(response.Events))
	for _, item := range response.Events {
		result = append(result, eventsemantic.EligibleEvent(item))
	}
	if err == nil && !validEligibleEvents(result) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) CreateContextLease(
	ctx context.Context,
	request eventsemantic.ContextLeaseRequest,
) (eventsemantic.ContextLease, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return eventsemantic.ContextLease{}, err
	}
	var response semanticContextLeaseWire
	_, err = c.semanticDo(
		ctx, http.MethodPost, dataAPIPrefix+"/event-semantics/context-leases", payload, &response,
		"data.v1.createEventSemanticContextLease", "/api/data/v1/event-semantics/context-leases",
	)
	if err != nil {
		return eventsemantic.ContextLease{}, err
	}
	result := eventsemantic.ContextLease(response)
	if !validSemanticContextLease(result) || result.EventID != request.EventID ||
		result.SupersedesSubmissionID != request.SupersedesSubmissionID {
		return eventsemantic.ContextLease{}, invalidSemanticResponse()
	}
	return result, nil
}

func (c *Client) Context(ctx context.Context, contextLeaseID string) (eventsemantic.Context, error) {
	var response semanticContextWire
	_, err := c.semanticDo(
		ctx,
		http.MethodGet,
		dataAPIPrefix+"/event-semantics/context-leases/"+url.PathEscape(contextLeaseID)+"/context",
		nil,
		&response,
		"data.v1.getEventSemanticContext",
		"/api/data/v1/event-semantics/context-leases/{context_lease_id}/context",
	)
	result := eventsemantic.Context(response)
	if err == nil && !validSemanticContext(result, contextLeaseID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) Resolve(
	ctx context.Context,
	contextLeaseID string,
	mentions []eventsemantic.EntityMention,
) ([]eventsemantic.EntityResolution, error) {
	payload, err := json.Marshal(map[string]any{"context_lease_id": contextLeaseID, "mentions": mentions})
	if err != nil {
		return nil, err
	}
	var response struct {
		Resolutions []semanticResolutionWire `json:"resolutions"`
	}
	_, err = c.semanticDo(
		ctx, http.MethodPost, dataAPIPrefix+"/event-semantics/entity-resolutions", payload, &response,
		"data.v1.resolveEventSemanticEntities", "/api/data/v1/event-semantics/entity-resolutions",
	)
	result := make([]eventsemantic.EntityResolution, 0, len(response.Resolutions))
	for _, item := range response.Resolutions {
		result = append(result, eventsemantic.EntityResolution(item))
	}
	if err == nil && !validSemanticResolutions(result) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) SearchDirectTargets(
	ctx context.Context,
	contextLeaseID string,
	subjectEntityID string,
	allowedTargetTypes []string,
) ([]eventsemantic.DirectTarget, error) {
	payload, err := json.Marshal(map[string]any{
		"context_lease_id": contextLeaseID, "subject_entity_id": subjectEntityID,
		"allowed_target_types": allowedTargetTypes,
	})
	if err != nil {
		return nil, err
	}
	var response struct {
		Targets []semanticTargetWire `json:"targets"`
	}
	_, err = c.semanticDo(
		ctx, http.MethodPost, dataAPIPrefix+"/event-semantics/direct-targets:search", payload, &response,
		"data.v1.searchEventSemanticDirectTargets",
		"/api/data/v1/event-semantics/direct-targets:search",
	)
	result := make([]eventsemantic.DirectTarget, 0, len(response.Targets))
	for _, item := range response.Targets {
		result = append(result, eventsemantic.DirectTarget(item))
	}
	if err == nil && !validSemanticTargets(result, subjectEntityID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) CreateSubmission(
	ctx context.Context,
	request eventsemantic.SubmissionRequest,
) (eventsemantic.SubmissionResult, error) {
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
	if err == nil && !validSemanticRun(result) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) SubmitReview(
	ctx context.Context,
	runID string,
	request eventsemantic.ReviewRequest,
) (eventsemantic.SubmissionResult, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	var response semanticSubmissionWire
	_, err = c.semanticReplaySafeDo(
		ctx,
		http.MethodPost,
		dataAPIPrefix+"/event-semantics/submissions/"+url.PathEscape(runID)+"/reviews",
		payload,
		&response,
		"data.v1.submitEventSemanticReview",
		"/api/data/v1/event-semantics/submissions/{submission_id}/reviews",
	)
	result := eventsemantic.SubmissionResult(response)
	if err == nil && (!validSemanticRun(result) || result.SubmissionID != runID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) semanticReplaySafeDo(
	ctx context.Context,
	method string,
	path string,
	payload []byte,
	target any,
	operation string,
	pathTemplate string,
) (int, error) {
	ctx, cancel := c.semanticTotalContext(ctx)
	defer cancel()
	var status int
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		status, err = c.semanticDo(
			ctx, method, path, payload, target, operation, pathTemplate,
		)
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

func (c *Client) GetEventSemantics(
	ctx context.Context,
	eventID string,
) (eventsemantic.EventSemantics, error) {
	var response eventSemanticsWire
	_, err := c.semanticDo(
		ctx,
		http.MethodGet,
		dataAPIPrefix+"/events/"+url.PathEscape(eventID)+"/semantics",
		nil,
		&response,
		"data.v1.getEventSemantics",
		"/api/data/v1/events/{event_id}/semantics",
	)
	result := eventsemantic.EventSemantics(response)
	if err == nil && !validEventSemantics(result, eventID) {
		err = invalidSemanticResponse()
	}
	return result, err
}

func (c *Client) semanticDo(
	ctx context.Context,
	method string,
	path string,
	payload []byte,
	target any,
	operation string,
	pathTemplate string,
) (int, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return 0, &eventsemantic.RemoteError{Code: "data_request_invalid", Summary: "Data Service request is invalid"}
	}
	request.Header.Set("Authorization", "Bearer "+c.serviceToken)
	requestID := uuid.NewString()
	request.Header.Set("X-Request-ID", requestID)
	request.Header.Set("X-Tidewise-Operation", operation)
	request.Header.Set("X-Tidewise-Path-Template", pathTemplate)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return 0, &eventsemantic.RemoteError{
			Code: "data_transport_unavailable", Summary: "Data Service is unavailable", Retryable: true,
		}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil || int64(len(body)) > c.maxResponseBytes {
		return response.StatusCode, &eventsemantic.RemoteError{
			Status: response.StatusCode, Code: "data_response_invalid",
			Summary: "Data Service response is unavailable", Retryable: response.StatusCode >= 500,
		}
	}
	if response.Header.Get("X-Request-ID") != requestID {
		return response.StatusCode, invalidSemanticResponse()
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var public semanticErrorEnvelope
		if err := decodeSemanticWire(body, &public); err != nil ||
			public.RequestID != requestID ||
			strings.TrimSpace(public.Error.Code) == "" ||
			strings.TrimSpace(public.Error.Message) == "" ||
			public.Error.Details == nil {
			return response.StatusCode, invalidSemanticResponse()
		}
		return response.StatusCode, &eventsemantic.RemoteError{
			Status: response.StatusCode,
			Code:   public.Error.Code, Summary: public.Error.Message,
			Retryable: response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500,
		}
	}
	var envelope struct {
		RequestID string          `json:"request_id"`
		Result    json.RawMessage `json:"result"`
	}
	if err := decodeSemanticWire(body, &envelope); err != nil ||
		envelope.RequestID != requestID {
		return response.StatusCode, &eventsemantic.RemoteError{
			Status: response.StatusCode, Code: "data_response_invalid",
			Summary: "Data Service response contract is invalid",
		}
	}
	if err := decodeSemanticWire(envelope.Result, target); err != nil {
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
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
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
	return &eventsemantic.RemoteError{
		Code: "data_response_invalid", Summary: "Data Service response contract is invalid",
	}
}

func validSemanticContextLease(value eventsemantic.ContextLease) bool {
	return validUUID(value.ContextLeaseID) && validUUID(value.EventID) &&
		value.Status == "active" && !value.LeaseExpiresAt.IsZero()
}

func validEligibleEvents(values []eventsemantic.EligibleEvent) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validUUID(value.EventID) {
			return false
		}
		if _, exists := seen[value.EventID]; exists {
			return false
		}
		seen[value.EventID] = struct{}{}
	}
	return true
}

func validSemanticContext(value eventsemantic.Context, contextLeaseID string) bool {
	if value.ContextLeaseID != contextLeaseID || value.OntologyVersion == "" ||
		value.AcceptancePolicyVersion == "" || !validUUID(value.Event.ID) ||
		value.Event.EventStatus != "confirmed" || value.Event.FactStatus != "verified" ||
		len(value.Evidence) == 0 {
		return false
	}
	for _, evidence := range value.Evidence {
		if !validUUID(evidence.EvidenceID) || !validUUID(evidence.RawDocumentID) ||
			len(evidence.EvidenceHash) != 64 ||
			strings.TrimSpace(evidence.Excerpt) == "" ||
			strings.TrimSpace(evidence.SourceName) == "" ||
			strings.TrimSpace(evidence.SourceType) == "" ||
			strings.TrimSpace(evidence.Title) == "" ||
			strings.TrimSpace(evidence.FirstSeenAt) == "" ||
			strings.TrimSpace(evidence.KnowledgeAvailableAt) == "" ||
			strings.TrimSpace(evidence.AcceptedAt) == "" {
			return false
		}
	}
	return true
}

func validSemanticResolutions(values []eventsemantic.EntityResolution) bool {
	for _, value := range values {
		if strings.TrimSpace(value.Mention) == "" {
			return false
		}
		for _, candidate := range value.Candidates {
			if !validUUID(candidate.EntityID) || candidate.EntityType == "" ||
				candidate.CanonicalName == "" || candidate.Status != "active" {
				return false
			}
		}
	}
	return true
}

func validSemanticTargets(values []eventsemantic.DirectTarget, subjectEntityID string) bool {
	for _, value := range values {
		if !validUUID(value.Entity.EntityID) || value.Entity.Status != "active" ||
			!validUUID(value.Relation.EntityRelationID) ||
			value.Relation.FromEntityID != subjectEntityID ||
			value.Relation.ToEntityID != value.Entity.EntityID ||
			value.Relation.Status != "active" {
			return false
		}
	}
	return true
}

func validSemanticRun(value eventsemantic.SubmissionResult) bool {
	if !validUUID(value.SubmissionID) || !validUUID(value.EventID) ||
		len(value.CanonicalPayloadHash) != 64 ||
		!semanticStatus(value.Status) {
		return false
	}
	for _, group := range [][]eventsemantic.CandidateDecision{
		value.EntityLinks, value.VariableSignals, value.DirectImpacts,
	} {
		for _, decision := range group {
			if strings.TrimSpace(decision.CandidateKey) == "" ||
				!semanticStatus(decision.Status) ||
				(decision.RecordID != "" && !validUUID(decision.RecordID)) {
				return false
			}
		}
	}
	return true
}

func validEventSemantics(value eventsemantic.EventSemantics, eventID string) bool {
	if value.EventID != eventID {
		return false
	}
	for _, run := range value.Submissions {
		if run.EventID != eventID || !validSemanticRun(run) {
			return false
		}
	}
	return true
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func semanticStatus(value string) bool {
	switch value {
	case "pending_review", "needs_reanalysis", "quarantined", "accepted", "rejected", "superseded":
		return true
	default:
		return false
	}
}
