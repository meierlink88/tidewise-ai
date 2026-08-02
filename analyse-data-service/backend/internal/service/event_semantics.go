package service

import (
	"context"
	"errors"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
)

func (s *DataService) ListEligibleEventSemanticEvents(
	ctx context.Context,
	request *v1.EligibleEventSemanticEventsRequest,
) (*v1.Response[v1.EligibleEventSemanticEvents], error) {
	if s == nil || s.dependencies.EventSemantics == nil {
		return nil, eventSemanticsNotReady()
	}
	page, err := s.dependencies.EventSemantics.ListEligibleEvents(
		ctx, request.Limit, request.Cursor,
	)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	result := v1.EligibleEventSemanticEvents{
		Events: make([]v1.EligibleEventSemanticEvent, 0, len(page.Events)),
	}
	if request.Pagination == "cursor" {
		result.NextCursor = page.NextCursor
	}
	for _, item := range page.Events {
		result.Events = append(result.Events, v1.EligibleEventSemanticEvent{EventID: item.EventID})
	}
	return &v1.Response[v1.EligibleEventSemanticEvents]{Status: v1.StatusOK, Result: result}, nil
}

func (s *DataService) CreateEventSemanticContextLease(
	ctx context.Context,
	request *v1.EventSemanticContextLeaseRequest,
) (*v1.Response[v1.EventSemanticContextLease], error) {
	if s == nil || s.dependencies.EventSemantics == nil {
		return nil, eventSemanticsNotReady()
	}
	result, err := s.dependencies.EventSemantics.CreateContextLease(ctx, eventsemantics.ContextLeaseRequest{
		EventID: request.EventID, SupersedesSubmissionID: request.SupersedesSubmissionID,
		AgentExecutionID: request.AgentExecutionID, WorkerID: request.WorkerID,
		Lease: time.Duration(request.LeaseSeconds) * time.Second,
	})
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	return &v1.Response[v1.EventSemanticContextLease]{
		Status: v1.StatusCreated,
		Result: v1.EventSemanticContextLease{
			ContextLeaseID: result.ID, EventID: result.EventID,
			SupersedesSubmissionID: result.SupersedesSubmissionID, Status: result.Status,
			LeaseExpiresAt: result.LeaseExpiresAt.UTC().Format(time.RFC3339Nano),
		},
	}, nil
}

func (s *DataService) GetEventSemanticContext(
	ctx context.Context,
	request *v1.EventSemanticContextRequest,
) (*v1.Response[v1.EventSemanticContext], error) {
	if s == nil || s.dependencies.EventSemantics == nil {
		return nil, eventSemanticsNotReady()
	}
	result, err := s.dependencies.EventSemantics.Context(ctx, request.ContextLeaseID)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	return &v1.Response[v1.EventSemanticContext]{
		Status: v1.StatusOK,
		Result: eventSemanticContextDTO(result),
	}, nil
}

func (s *DataService) CreateEventSemanticSubmission(
	ctx context.Context,
	request *v1.EventSemanticSubmissionRequest,
) (*v1.Response[v1.EventSemanticSubmissionResult], error) {
	if s == nil || s.dependencies.EventSemantics == nil {
		return nil, eventSemanticsNotReady()
	}
	input, err := eventSemanticSubmissionInput(request)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "Event Semantic Submission contains an invalid UTC timestamp")
	}
	result, err := s.dependencies.EventSemantics.CreateSubmission(ctx, input)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[v1.EventSemanticSubmissionResult]{
		Status: status, Result: eventSemanticSubmissionResultDTO(result),
	}, nil
}

func (s *DataService) SubmitEventSemanticReview(
	ctx context.Context,
	request *v1.EventSemanticReviewRequest,
) (*v1.Response[v1.EventSemanticSubmissionResult], error) {
	if s == nil || s.dependencies.EventSemantics == nil {
		return nil, eventSemanticsNotReady()
	}
	items := make([]eventsemantics.ReviewItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, eventsemantics.ReviewItem{
			CandidateType: item.CandidateType, CandidateKey: item.CandidateKey,
			Decision: item.Decision, ReasonCodes: item.ReasonCodes, EvidenceIDs: item.EvidenceIDs,
		})
	}
	result, err := s.dependencies.EventSemantics.SubmitReview(ctx, eventsemantics.ReviewSubmission{
		SubmissionID: request.SubmissionID, ReviewerExecutionKey: request.ReviewerExecutionKey,
		PromptHash: request.PromptHash, Model: request.Model, Items: items,
	})
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	return &v1.Response[v1.EventSemanticSubmissionResult]{
		Status: v1.StatusOK, Result: eventSemanticSubmissionResultDTO(result),
	}, nil
}

func (s *DataService) GetEventSemantics(
	ctx context.Context,
	request *v1.GetEventSemanticsRequest,
) (*v1.Response[v1.EventSemanticsResult], error) {
	if s == nil || s.dependencies.EventSemantics == nil {
		return nil, eventSemanticsNotReady()
	}
	result, err := s.dependencies.EventSemantics.Get(ctx, request.EventID)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	submissions := make([]v1.EventSemanticSubmissionResult, 0, len(result.Submissions))
	for _, submission := range result.Submissions {
		submissions = append(submissions, eventSemanticSubmissionResultDTO(submission))
	}
	return &v1.Response[v1.EventSemanticsResult]{
		Status: v1.StatusOK,
		Result: v1.EventSemanticsResult{EventID: result.EventID, Submissions: submissions},
	}, nil
}

func eventSemanticsNotReady() error {
	return publicError(v1.StatusServiceUnavailable, "EVENT_SEMANTICS_NOT_READY", "Event Semantics service is unavailable")
}

func eventSemanticsError(err error) error {
	var validation *eventsemantics.ValidationError
	var notFound *eventsemantics.NotFoundError
	var conflict *eventsemantics.ConflictError
	var notRequired *eventsemantics.NotRequiredError
	var inputInvalid *eventsemantics.InputInvalidError
	var contextDrift *eventsemantics.ContextDriftError
	switch {
	case errors.As(err, &contextDrift):
		return publicError(v1.StatusConflict, "EVENT_SEMANTIC_CONTEXT_DRIFT", contextDrift.Reason)
	case errors.As(err, &notRequired):
		return publicError(v1.StatusConflict, "EVENT_SEMANTICS_NOT_REQUIRED", notRequired.Reason)
	case errors.As(err, &inputInvalid):
		return publicError(v1.StatusUnprocessableEntity, "EVENT_SEMANTICS_INPUT_INVALID", inputInvalid.Reason)
	case errors.As(err, &validation):
		return publicError(v1.StatusUnprocessableEntity, "EVENT_SEMANTICS_INVALID", validation.Reason)
	case errors.As(err, &notFound):
		return publicError(v1.StatusNotFound, "EVENT_SEMANTICS_NOT_FOUND", "Event Semantics resource was not found")
	case errors.As(err, &conflict):
		return publicError(v1.StatusConflict, "EVENT_SEMANTICS_CONFLICT", conflict.Reason)
	default:
		return publicError(v1.StatusInternalServerError, "EVENT_SEMANTICS_FAILED", "Event Semantics operation failed")
	}
}

func eventSemanticSubmissionInput(request *v1.EventSemanticSubmissionRequest) (eventsemantics.Submission, error) {
	result := eventsemantics.Submission{
		ContextLeaseID: request.ContextLeaseID, EventID: request.EventID, AgentExecutionID: request.AgentExecutionID,
		AgentKey: request.AgentKey, AgentVersion: request.AgentVersion, SupersedesSubmissionID: request.SupersedesSubmissionID,
		GeneratorPromptHash: request.GeneratorPromptHash, GeneratorModel: request.GeneratorModel,
		ReviewerPromptHash: request.ReviewerPromptHash, ReviewerModel: request.ReviewerModel,
		AdjudicatorPromptHash: request.AdjudicatorPromptHash, AdjudicatorModel: request.AdjudicatorModel,
		OntologyVersion: request.OntologyVersion, AcceptancePolicyVersion: request.AcceptancePolicyVersion,
	}
	for _, link := range request.EntityLinks {
		candidate := eventsemantics.EntityLinkCandidate{
			Key: link.CandidateKey, Mention: link.Mention, EntityID: link.EntityID,
			ProjectedEntityType: link.ProjectedEntityType, EntityRole: link.EntityRole, EvidenceIDs: link.EvidenceIDs,
			ResolutionMethod: link.ResolutionMethod, ResolutionConfidence: link.ResolutionConfidence,
		}
		result.EntityLinks = append(result.EntityLinks, candidate)
	}
	for _, signal := range request.VariableSignals {
		statementAt, err := eventSemanticOptionalUTC(signal.StatementAt)
		if err != nil {
			return eventsemantics.Submission{}, err
		}
		validFrom, err := eventSemanticOptionalUTC(signal.ValidFrom)
		if err != nil {
			return eventsemantics.Submission{}, err
		}
		validUntil, err := eventSemanticOptionalUTC(signal.ValidUntil)
		if err != nil {
			return eventsemantics.Submission{}, err
		}
		forecastStart, err := eventSemanticOptionalUTC(signal.ForecastPeriodStart)
		if err != nil {
			return eventsemantics.Submission{}, err
		}
		forecastEnd, err := eventSemanticOptionalUTC(signal.ForecastPeriodEnd)
		if err != nil {
			return eventsemantics.Submission{}, err
		}
		measurements := make([]eventsemantics.MeasurementValue, 0, len(signal.Measurements))
		for _, measurement := range signal.Measurements {
			measurements = append(measurements, eventsemantics.MeasurementValue{
				Text: measurement.MeasurementText, EvidenceIDs: measurement.EvidenceIDs,
			})
		}
		result.VariableSignals = append(result.VariableSignals, eventsemantics.VariableSignalCandidate{
			Key: signal.CandidateKey, SubjectLinkKey: signal.SubjectLinkKey,
			VariableKey: signal.VariableKey, VariableVersion: signal.VariableVersion,
			Direction: signal.Direction, AssertionModality: signal.AssertionModality,
			EvidenceIDs: signal.EvidenceIDs, Measurements: measurements,
			StatementAt: statementAt, ValidFrom: validFrom, ValidUntil: validUntil,
			ForecastPeriodStart: forecastStart, ForecastPeriodEnd: forecastEnd,
			ExtractionConfidence: signal.ExtractionConfidence,
		})
	}
	return result, nil
}

func eventSemanticOptionalUTC(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	return optionalUTC(*raw)
}

func eventSemanticContextDTO(value eventsemantics.Context) v1.EventSemanticContext {
	result := v1.EventSemanticContext{
		ContextLeaseID: value.ContextLeaseID, AgentExecutionID: value.AgentExecutionID,
		WorkerID: value.WorkerID, LeaseExpiresAt: value.LeaseExpiresAt.UTC().Format(time.RFC3339Nano),
		ManifestContractVersion: value.ManifestContractVersion,
		ContextFingerprint:      value.ContextFingerprint, EventFingerprint: value.EventFingerprint,
		EvidenceFingerprint: value.EvidenceFingerprint, OntologyVersion: value.OntologyVersion,
		AcceptancePolicyVersion: value.PolicyVersion,
		Event:                   eventSemanticEventDTO(value.Event),
		Evidence:                make([]v1.EventSemanticEvidence, 0, len(value.Evidence)),
		EntityTypeDefinitions:   make([]v1.EventSemanticEntityTypeDefinition, 0, len(value.EntityTypes)),
		VariableDefinitions:     make([]v1.EventSemanticVariableDefinition, 0, len(value.Variables)),
		AssertionModalities:     value.AssertionModalities,
		MeasurementContract: v1.EventSemanticMeasurementContract{
			Representation:      value.MeasurementContract.Representation,
			MaxItemsPerSignal:   value.MeasurementContract.MaxItemsPerSignal,
			MaxTextCharacters:   value.MeasurementContract.MaxTextCharacters,
			RequiresEvidenceIDs: value.MeasurementContract.RequiresEvidenceIDs,
			NumericValidation:   value.MeasurementContract.NumericValidation,
		},
	}
	for _, evidence := range value.Evidence {
		result.Evidence = append(result.Evidence, eventSemanticEvidenceDTO(evidence))
	}
	for _, definition := range value.EntityTypes {
		result.EntityTypeDefinitions = append(result.EntityTypeDefinitions, v1.EventSemanticEntityTypeDefinition{
			TypeKey: definition.TypeKey, Version: definition.Version,
			NameZH: definition.NameZH, NameEN: definition.NameEN,
			BusinessDefinition:   definition.BusinessDefinition,
			InclusionCriteria:    definition.InclusionCriteria,
			ExclusionCriteria:    definition.ExclusionCriteria,
			EventLinkAllowed:     definition.EventLinkAllowed,
			SignalSubjectAllowed: definition.SignalSubjectAllowed,
			AllowedEventRoles:    definition.AllowedEventRoles, Status: definition.Status,
		})
	}
	for _, variable := range value.Variables {
		result.VariableDefinitions = append(result.VariableDefinitions, v1.EventSemanticVariableDefinition{
			Key: variable.Key, Version: variable.Version, NameZH: variable.NameZH, NameEN: variable.NameEN,
			Domain: variable.Domain, BusinessDefinition: variable.BusinessDefinition,
			ValueType: variable.ValueType, Status: variable.Status,
			AllowedDirections:     variable.AllowedDirections,
			AllowedUnits:          variable.AllowedUnits,
			ApplicableEntityTypes: variable.ApplicableEntityTypes,
		})
	}
	return result
}

func eventSemanticSubmissionResultDTO(value eventsemantics.SubmissionResult) v1.EventSemanticSubmissionResult {
	result := v1.EventSemanticSubmissionResult{
		SubmissionID: value.SubmissionID, EventID: value.EventID, Status: string(value.Status),
		CanonicalPayloadHash: value.CanonicalPayloadHash, Replayed: value.Replayed,
		EntityLinks:     eventSemanticDecisionsDTO(value.Precheck.EntityLinks),
		VariableSignals: eventSemanticDecisionsDTO(value.Precheck.VariableSignals),
		ContextLeaseID:  value.ContextLeaseID, AgentExecutionID: value.AgentExecutionID,
		AgentKey: value.AgentKey, AgentVersion: value.AgentVersion,
		SupersedesSubmissionID: value.SupersedesSubmissionID,
		GeneratorPromptHash:    value.GeneratorPromptHash, GeneratorModel: value.GeneratorModel,
		ReviewerPromptHash: value.ReviewerPromptHash, ReviewerModel: value.ReviewerModel,
		AdjudicatorPromptHash: value.AdjudicatorPromptHash, AdjudicatorModel: value.AdjudicatorModel,
		OntologyVersion: value.OntologyVersion, AcceptancePolicyVersion: value.AcceptancePolicyVersion,
		CandidateSnapshot: value.CandidateSnapshot, FinalizedAt: formatOptionalTime(value.FinalizedAt),
	}
	if !value.CreatedAt.IsZero() {
		result.CreatedAt = value.CreatedAt.UTC().Format(time.RFC3339)
	}
	for _, review := range value.ReviewSnapshots {
		result.ReviewSnapshots = append(result.ReviewSnapshots, v1.EventSemanticReviewSnapshot{
			ReviewerExecutionKey: review.ReviewerExecutionKey,
			CanonicalPayloadHash: review.CanonicalPayloadHash,
			Payload:              review.Payload, CreatedAt: review.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	if !value.CreatedAt.IsZero() {
		result.AuditWorkPackage = eventSemanticReviewerWorkPackageDTO(value.Precheck.ReviewerWorkPackage)
	}
	work := reviewableEventSemanticWorkPackage(value.Precheck)
	if len(work.EntityLinks)+len(work.VariableSignals) > 0 {
		result.ReviewerWorkPackage = eventSemanticReviewerWorkPackageDTO(work)
	}
	return result
}

func eventSemanticReviewerWorkPackageDTO(
	work eventsemantics.ReviewerWorkPackage,
) *v1.EventSemanticReviewerWorkPackage {
	result := &v1.EventSemanticReviewerWorkPackage{
		Event:           eventSemanticEventDTO(work.Event),
		EntityLinks:     eventSemanticLinkCandidatesDTO(work.EntityLinks),
		VariableSignals: eventSemanticSignalCandidatesDTO(work.VariableSignals),
	}
	for _, entity := range work.ResolvedEntities {
		result.ResolvedEntities = append(result.ResolvedEntities, v1.EventSemanticEntity{
			EntityID: entity.ID, EntityType: entity.Type, Name: entity.Name,
			CanonicalName: entity.CanonicalName, Aliases: entity.Aliases, Status: entity.Status,
		})
	}
	for _, evidence := range work.Evidence {
		result.Evidence = append(result.Evidence, eventSemanticEvidenceDTO(evidence))
	}
	return result
}

func eventSemanticEvidenceDTO(value eventsemantics.Evidence) v1.EventSemanticEvidence {
	result := v1.EventSemanticEvidence{
		EvidenceID: value.ID, EvidenceHash: value.Hash, Statement: value.Statement,
		SourceLevel: value.SourceLevel, Relation: value.Relation,
		SupportsFields: value.SupportsFields,
		RawDocumentID:  value.RawDocumentID, SourceName: value.SourceName,
		SourceType: value.SourceType, SourceURL: value.SourceURL, Title: value.Title,
		FirstSeenAt:          value.FirstSeenAt.UTC().Format(time.RFC3339Nano),
		KnowledgeAvailableAt: value.KnowledgeAvailableAt.UTC().Format(time.RFC3339Nano),
		AcceptedAt:           value.AcceptedAt.UTC().Format(time.RFC3339Nano),
		StatementSource:      value.StatementSource,
	}
	if value.PublishedAt != nil {
		formatted := value.PublishedAt.UTC().Format(time.RFC3339Nano)
		result.PublishedAt = &formatted
	}
	return result
}

func reviewableEventSemanticWorkPackage(
	precheck eventsemantics.PrecheckResult,
) eventsemantics.ReviewerWorkPackage {
	work := precheck.ReviewerWorkPackage
	reviewable := func(status eventsemantics.ReviewStatus) bool {
		return status == eventsemantics.StatusPendingReview ||
			status == eventsemantics.StatusNeedsReanalysis
	}
	linkStatus := make(map[string]eventsemantics.ReviewStatus, len(precheck.EntityLinks))
	for _, item := range precheck.EntityLinks {
		linkStatus[item.CandidateKey] = item.Status
	}
	signalStatus := make(map[string]eventsemantics.ReviewStatus, len(precheck.VariableSignals))
	for _, item := range precheck.VariableSignals {
		signalStatus[item.CandidateKey] = item.Status
	}
	work.EntityLinks = filterEventSemanticCandidates(
		work.EntityLinks, func(item eventsemantics.EntityLinkCandidate) bool {
			return reviewable(linkStatus[item.Key])
		},
	)
	work.VariableSignals = filterEventSemanticCandidates(
		work.VariableSignals, func(item eventsemantics.VariableSignalCandidate) bool {
			return reviewable(signalStatus[item.Key])
		},
	)
	work.DirectImpacts = nil
	return work
}

func filterEventSemanticCandidates[T any](items []T, keep func(T) bool) []T {
	result := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

func eventSemanticEventDTO(value eventsemantics.Event) v1.EventSemanticEvent {
	return v1.EventSemanticEvent{
		ID: value.ID, Title: value.Title, Summary: value.Summary,
		OccurredAt: formatOptionalTime(value.OccurredAt), EventStatus: value.Status, FactStatus: value.FactStatus,
	}
}

func eventSemanticEntityDTO(value eventsemantics.Entity) v1.EventSemanticEntity {
	return v1.EventSemanticEntity{
		EntityID: value.ID, EntityType: value.Type, Name: value.Name,
		CanonicalName: value.CanonicalName, Aliases: value.Aliases, Status: value.Status,
	}
}

func eventSemanticDecisionsDTO(values []eventsemantics.CandidateDecision) []v1.EventSemanticCandidateDecision {
	result := make([]v1.EventSemanticCandidateDecision, 0, len(values))
	for _, value := range values {
		result = append(result, v1.EventSemanticCandidateDecision{
			CandidateKey: value.CandidateKey, Status: string(value.Status),
			ReasonCode: value.ReasonCode, RecordID: value.RecordID,
		})
	}
	return result
}

func eventSemanticLinkCandidatesDTO(values []eventsemantics.EntityLinkCandidate) []v1.EventSemanticEntityLinkCandidate {
	result := make([]v1.EventSemanticEntityLinkCandidate, 0, len(values))
	for _, value := range values {
		item := v1.EventSemanticEntityLinkCandidate{
			CandidateKey: value.Key, Mention: value.Mention, EntityID: value.EntityID,
			ProjectedEntityType: value.ProjectedEntityType, EntityRole: value.EntityRole, EvidenceIDs: value.EvidenceIDs,
			ResolutionMethod: value.ResolutionMethod, ResolutionConfidence: value.ResolutionConfidence,
		}
		result = append(result, item)
	}
	return result
}

func eventSemanticSignalCandidatesDTO(values []eventsemantics.VariableSignalCandidate) []v1.EventSemanticVariableSignalCandidate {
	result := make([]v1.EventSemanticVariableSignalCandidate, 0, len(values))
	for _, value := range values {
		item := v1.EventSemanticVariableSignalCandidate{
			CandidateKey: value.Key, SubjectLinkKey: value.SubjectLinkKey,
			VariableKey: value.VariableKey, VariableVersion: value.VariableVersion,
			Direction: value.Direction, AssertionModality: value.AssertionModality,
			EvidenceIDs: value.EvidenceIDs, StatementAt: formatOptionalTime(value.StatementAt),
			ValidFrom: formatOptionalTime(value.ValidFrom), ValidUntil: formatOptionalTime(value.ValidUntil),
			ForecastPeriodStart:  formatOptionalTime(value.ForecastPeriodStart),
			ForecastPeriodEnd:    formatOptionalTime(value.ForecastPeriodEnd),
			ExtractionConfidence: value.ExtractionConfidence,
		}
		for _, measurement := range value.Measurements {
			item.Measurements = append(item.Measurements, v1.EventSemanticMeasurement{
				MeasurementText: measurement.Text,
				EvidenceIDs:     measurement.EvidenceIDs,
			})
		}
		result = append(result, item)
	}
	return result
}
