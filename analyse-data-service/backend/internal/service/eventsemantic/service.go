package eventsemantic

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventsemanticapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/eventsemantic"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
)

type UseCase interface {
	ListEligibleEvents(context.Context, int, string) (eventbiz.EligibleEventPage, error)
	CreateContextLease(context.Context, eventbiz.ContextLeaseRequest) (eventbiz.ContextLease, error)
	Context(context.Context, string) (eventbiz.Context, error)
	CreateSubmission(context.Context, eventbiz.Submission) (eventbiz.SubmissionResult, error)
	SubmitReview(context.Context, eventbiz.ReviewSubmission) (eventbiz.SubmissionResult, error)
	Get(context.Context, string) (eventbiz.EventSemanticsResult, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Event Semantic use case is required")
	}
	return &Service{useCase: useCase}, nil
}

var _ eventsemanticapi.Service = (*Service)(nil)

func (s *Service) ListEligibleEventSemanticEvents(
	ctx context.Context,
	request *eventsemanticapi.EligibleEventSemanticEventsRequest,
) (*v1.Response[eventsemanticapi.EligibleEventSemanticEvents], error) {
	if s == nil || s.useCase == nil {
		return nil, eventSemanticsNotReady()
	}
	page, err := s.useCase.ListEligibleEvents(
		ctx, request.Limit, request.Cursor,
	)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	result := eventsemanticapi.EligibleEventSemanticEvents{
		Events: make([]eventsemanticapi.EligibleEventSemanticEvent, 0, len(page.Events)),
	}
	if request.Pagination == "cursor" {
		result.NextCursor = page.NextCursor
	}
	for _, item := range page.Events {
		result.Events = append(result.Events, eventsemanticapi.EligibleEventSemanticEvent{EventID: item.EventID})
	}
	return &v1.Response[eventsemanticapi.EligibleEventSemanticEvents]{Status: v1.StatusOK, Result: result}, nil
}

func (s *Service) CreateEventSemanticContextLease(
	ctx context.Context,
	request *eventsemanticapi.EventSemanticContextLeaseRequest,
) (*v1.Response[eventsemanticapi.EventSemanticContextLease], error) {
	if s == nil || s.useCase == nil {
		return nil, eventSemanticsNotReady()
	}
	result, err := s.useCase.CreateContextLease(ctx, eventbiz.ContextLeaseRequest{
		EventID: request.EventID, SupersedesSubmissionID: request.SupersedesSubmissionID,
		AgentExecutionID: request.AgentExecutionID, WorkerID: request.WorkerID,
		Lease: time.Duration(request.LeaseSeconds) * time.Second,
	})
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	return &v1.Response[eventsemanticapi.EventSemanticContextLease]{
		Status: v1.StatusCreated,
		Result: eventsemanticapi.EventSemanticContextLease{
			ContextLeaseID: result.ID, EventID: result.EventID,
			SupersedesSubmissionID: result.SupersedesSubmissionID, Status: result.Status,
			LeaseExpiresAt: result.LeaseExpiresAt.UTC().Format(time.RFC3339Nano),
		},
	}, nil
}

func (s *Service) GetEventSemanticContext(
	ctx context.Context,
	request *eventsemanticapi.EventSemanticContextRequest,
) (*v1.Response[eventsemanticapi.EventSemanticContext], error) {
	if s == nil || s.useCase == nil {
		return nil, eventSemanticsNotReady()
	}
	result, err := s.useCase.Context(ctx, request.ContextLeaseID)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	return &v1.Response[eventsemanticapi.EventSemanticContext]{
		Status: v1.StatusOK,
		Result: eventSemanticContextDTO(result),
	}, nil
}

func (s *Service) CreateEventSemanticSubmission(
	ctx context.Context,
	request *eventsemanticapi.EventSemanticSubmissionRequest,
) (*v1.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	if s == nil || s.useCase == nil {
		return nil, eventSemanticsNotReady()
	}
	input, err := eventSemanticSubmissionInput(request)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, eventsemanticapi.ErrorInvalidRequest, "Event Semantic Submission contains an invalid UTC timestamp")
	}
	result, err := s.useCase.CreateSubmission(ctx, input)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[eventsemanticapi.EventSemanticSubmissionResult]{
		Status: status, Result: eventSemanticSubmissionResultDTO(result),
	}, nil
}

func (s *Service) SubmitEventSemanticReview(
	ctx context.Context,
	request *eventsemanticapi.EventSemanticReviewRequest,
) (*v1.Response[eventsemanticapi.EventSemanticSubmissionResult], error) {
	if s == nil || s.useCase == nil {
		return nil, eventSemanticsNotReady()
	}
	items := make([]eventbiz.ReviewItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, eventbiz.ReviewItem{
			CandidateType: item.CandidateType, CandidateKey: item.CandidateKey,
			Decision: item.Decision, ReasonCodes: item.ReasonCodes, EvidenceIDs: item.EvidenceIDs,
		})
	}
	result, err := s.useCase.SubmitReview(ctx, eventbiz.ReviewSubmission{
		SubmissionID: request.SubmissionID, ReviewerExecutionKey: request.ReviewerExecutionKey,
		PromptHash: request.PromptHash, Model: request.Model, Items: items,
	})
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	return &v1.Response[eventsemanticapi.EventSemanticSubmissionResult]{
		Status: v1.StatusOK, Result: eventSemanticSubmissionResultDTO(result),
	}, nil
}

func (s *Service) GetEventSemantics(
	ctx context.Context,
	request *eventsemanticapi.GetEventSemanticsRequest,
) (*v1.Response[eventsemanticapi.EventSemanticsResult], error) {
	if s == nil || s.useCase == nil {
		return nil, eventSemanticsNotReady()
	}
	result, err := s.useCase.Get(ctx, request.EventID)
	if err != nil {
		return nil, eventSemanticsError(err)
	}
	submissions := make([]eventsemanticapi.EventSemanticSubmissionResult, 0, len(result.Submissions))
	for _, submission := range result.Submissions {
		submissions = append(submissions, eventSemanticSubmissionResultDTO(submission))
	}
	return &v1.Response[eventsemanticapi.EventSemanticsResult]{
		Status: v1.StatusOK,
		Result: eventsemanticapi.EventSemanticsResult{EventID: result.EventID, Submissions: submissions},
	}, nil
}

func eventSemanticsNotReady() error {
	return publicError(v1.StatusServiceUnavailable, eventsemanticapi.ErrorNotReady, "Event Semantics service is unavailable")
}

func eventSemanticsError(err error) error {
	var validation *eventbiz.ValidationError
	var notFound *eventbiz.NotFoundError
	var conflict *eventbiz.ConflictError
	var notRequired *eventbiz.NotRequiredError
	var inputInvalid *eventbiz.InputInvalidError
	var contextDrift *eventbiz.ContextDriftError
	switch {
	case errors.As(err, &contextDrift):
		return publicError(v1.StatusConflict, eventsemanticapi.ErrorContextDrift, contextDrift.Reason)
	case errors.As(err, &notRequired):
		return publicError(v1.StatusConflict, eventsemanticapi.ErrorNotRequired, notRequired.Reason)
	case errors.As(err, &inputInvalid):
		return publicError(v1.StatusUnprocessableEntity, eventsemanticapi.ErrorInputInvalid, inputInvalid.Reason)
	case errors.As(err, &validation):
		return publicError(v1.StatusUnprocessableEntity, eventsemanticapi.ErrorInvalid, validation.Reason)
	case errors.As(err, &notFound):
		return publicError(v1.StatusNotFound, eventsemanticapi.ErrorNotFound, "Event Semantics resource was not found")
	case errors.As(err, &conflict):
		return publicError(v1.StatusConflict, eventsemanticapi.ErrorConflict, conflict.Reason)
	default:
		return publicError(v1.StatusInternalServerError, eventsemanticapi.ErrorFailed, "Event Semantics operation failed")
	}
}

func eventSemanticSubmissionInput(request *eventsemanticapi.EventSemanticSubmissionRequest) (eventbiz.Submission, error) {
	result := eventbiz.Submission{
		ContextLeaseID: request.ContextLeaseID, EventID: request.EventID, AgentExecutionID: request.AgentExecutionID,
		AgentKey: request.AgentKey, AgentVersion: request.AgentVersion, SupersedesSubmissionID: request.SupersedesSubmissionID,
		GeneratorPromptHash: request.GeneratorPromptHash, GeneratorModel: request.GeneratorModel,
		ReviewerPromptHash: request.ReviewerPromptHash, ReviewerModel: request.ReviewerModel,
		AdjudicatorPromptHash: request.AdjudicatorPromptHash, AdjudicatorModel: request.AdjudicatorModel,
		OntologyVersion: request.OntologyVersion, AcceptancePolicyVersion: request.AcceptancePolicyVersion,
	}
	for _, link := range request.EntityLinks {
		candidate := eventbiz.EntityLinkCandidate{
			Key: link.CandidateKey, Mention: link.Mention, EntityID: link.EntityID,
			ProjectedEntityType: link.ProjectedEntityType, EntityRole: link.EntityRole, EvidenceIDs: link.EvidenceIDs,
			ResolutionMethod: link.ResolutionMethod, ResolutionConfidence: link.ResolutionConfidence,
		}
		result.EntityLinks = append(result.EntityLinks, candidate)
	}
	for _, signal := range request.VariableSignals {
		statementAt, err := eventSemanticOptionalUTC(signal.StatementAt)
		if err != nil {
			return eventbiz.Submission{}, err
		}
		validFrom, err := eventSemanticOptionalUTC(signal.ValidFrom)
		if err != nil {
			return eventbiz.Submission{}, err
		}
		validUntil, err := eventSemanticOptionalUTC(signal.ValidUntil)
		if err != nil {
			return eventbiz.Submission{}, err
		}
		forecastStart, err := eventSemanticOptionalUTC(signal.ForecastPeriodStart)
		if err != nil {
			return eventbiz.Submission{}, err
		}
		forecastEnd, err := eventSemanticOptionalUTC(signal.ForecastPeriodEnd)
		if err != nil {
			return eventbiz.Submission{}, err
		}
		measurements := make([]eventbiz.MeasurementValue, 0, len(signal.Measurements))
		for _, measurement := range signal.Measurements {
			measurements = append(measurements, eventbiz.MeasurementValue{
				Text: measurement.MeasurementText, EvidenceIDs: measurement.EvidenceIDs,
			})
		}
		result.VariableSignals = append(result.VariableSignals, eventbiz.VariableSignalCandidate{
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

func eventSemanticContextDTO(value eventbiz.Context) eventsemanticapi.EventSemanticContext {
	result := eventsemanticapi.EventSemanticContext{
		ContextLeaseID: value.ContextLeaseID, AgentExecutionID: value.AgentExecutionID,
		WorkerID: value.WorkerID, LeaseExpiresAt: value.LeaseExpiresAt.UTC().Format(time.RFC3339Nano),
		ManifestContractVersion: value.ManifestContractVersion,
		ContextFingerprint:      value.ContextFingerprint, EventFingerprint: value.EventFingerprint,
		EvidenceFingerprint: value.EvidenceFingerprint, OntologyVersion: value.OntologyVersion,
		AcceptancePolicyVersion: value.PolicyVersion,
		Event:                   eventSemanticEventDTO(value.Event),
		Evidence:                make([]eventsemanticapi.EventSemanticEvidence, 0, len(value.Evidence)),
		EntityTypeDefinitions:   make([]eventsemanticapi.EventSemanticEntityTypeDefinition, 0, len(value.EntityTypes)),
		VariableDefinitions:     make([]eventsemanticapi.EventSemanticVariableDefinition, 0, len(value.Variables)),
		AssertionModalities:     value.AssertionModalities,
		MeasurementContract: eventsemanticapi.EventSemanticMeasurementContract{
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
		result.EntityTypeDefinitions = append(result.EntityTypeDefinitions, eventsemanticapi.EventSemanticEntityTypeDefinition{
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
		result.VariableDefinitions = append(result.VariableDefinitions, eventsemanticapi.EventSemanticVariableDefinition{
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

func eventSemanticSubmissionResultDTO(value eventbiz.SubmissionResult) eventsemanticapi.EventSemanticSubmissionResult {
	result := eventsemanticapi.EventSemanticSubmissionResult{
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
		result.ReviewSnapshots = append(result.ReviewSnapshots, eventsemanticapi.EventSemanticReviewSnapshot{
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
	work eventbiz.ReviewerWorkPackage,
) *eventsemanticapi.EventSemanticReviewerWorkPackage {
	result := &eventsemanticapi.EventSemanticReviewerWorkPackage{
		Event:           eventSemanticEventDTO(work.Event),
		EntityLinks:     eventSemanticLinkCandidatesDTO(work.EntityLinks),
		VariableSignals: eventSemanticSignalCandidatesDTO(work.VariableSignals),
	}
	for _, entity := range work.ResolvedEntities {
		result.ResolvedEntities = append(result.ResolvedEntities, eventsemanticapi.EventSemanticEntity{
			EntityID: entity.ID, EntityType: entity.Type, Name: entity.Name,
			CanonicalName: entity.CanonicalName, Aliases: entity.Aliases, Status: entity.Status,
		})
	}
	for _, evidence := range work.Evidence {
		result.Evidence = append(result.Evidence, eventSemanticEvidenceDTO(evidence))
	}
	return result
}

func eventSemanticEvidenceDTO(value eventbiz.Evidence) eventsemanticapi.EventSemanticEvidence {
	result := eventsemanticapi.EventSemanticEvidence{
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
	precheck eventbiz.PrecheckResult,
) eventbiz.ReviewerWorkPackage {
	work := precheck.ReviewerWorkPackage
	reviewable := func(status eventbiz.ReviewStatus) bool {
		return status == eventbiz.StatusPendingReview ||
			status == eventbiz.StatusNeedsReanalysis
	}
	linkStatus := make(map[string]eventbiz.ReviewStatus, len(precheck.EntityLinks))
	for _, item := range precheck.EntityLinks {
		linkStatus[item.CandidateKey] = item.Status
	}
	signalStatus := make(map[string]eventbiz.ReviewStatus, len(precheck.VariableSignals))
	for _, item := range precheck.VariableSignals {
		signalStatus[item.CandidateKey] = item.Status
	}
	work.EntityLinks = filterEventSemanticCandidates(
		work.EntityLinks, func(item eventbiz.EntityLinkCandidate) bool {
			return reviewable(linkStatus[item.Key])
		},
	)
	work.VariableSignals = filterEventSemanticCandidates(
		work.VariableSignals, func(item eventbiz.VariableSignalCandidate) bool {
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

func eventSemanticEventDTO(value eventbiz.Event) eventsemanticapi.EventSemanticEvent {
	return eventsemanticapi.EventSemanticEvent{
		ID: value.ID, Title: value.Title, Summary: value.Summary,
		OccurredAt: formatOptionalTime(value.OccurredAt), EventStatus: value.Status, FactStatus: value.FactStatus,
	}
}

func eventSemanticEntityDTO(value eventbiz.Entity) eventsemanticapi.EventSemanticEntity {
	return eventsemanticapi.EventSemanticEntity{
		EntityID: value.ID, EntityType: value.Type, Name: value.Name,
		CanonicalName: value.CanonicalName, Aliases: value.Aliases, Status: value.Status,
	}
}

func eventSemanticDecisionsDTO(values []eventbiz.CandidateDecision) []eventsemanticapi.EventSemanticCandidateDecision {
	result := make([]eventsemanticapi.EventSemanticCandidateDecision, 0, len(values))
	for _, value := range values {
		result = append(result, eventsemanticapi.EventSemanticCandidateDecision{
			CandidateKey: value.CandidateKey, Status: string(value.Status),
			ReasonCode: value.ReasonCode, RecordID: value.RecordID,
		})
	}
	return result
}

func eventSemanticLinkCandidatesDTO(values []eventbiz.EntityLinkCandidate) []eventsemanticapi.EventSemanticEntityLinkCandidate {
	result := make([]eventsemanticapi.EventSemanticEntityLinkCandidate, 0, len(values))
	for _, value := range values {
		item := eventsemanticapi.EventSemanticEntityLinkCandidate{
			CandidateKey: value.Key, Mention: value.Mention, EntityID: value.EntityID,
			ProjectedEntityType: value.ProjectedEntityType, EntityRole: value.EntityRole, EvidenceIDs: value.EvidenceIDs,
			ResolutionMethod: value.ResolutionMethod, ResolutionConfidence: value.ResolutionConfidence,
		}
		result = append(result, item)
	}
	return result
}

func eventSemanticSignalCandidatesDTO(values []eventbiz.VariableSignalCandidate) []eventsemanticapi.EventSemanticVariableSignalCandidate {
	result := make([]eventsemanticapi.EventSemanticVariableSignalCandidate, 0, len(values))
	for _, value := range values {
		item := eventsemanticapi.EventSemanticVariableSignalCandidate{
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
			item.Measurements = append(item.Measurements, eventsemanticapi.EventSemanticMeasurement{
				MeasurementText: measurement.Text,
				EvidenceIDs:     measurement.EvidenceIDs,
			})
		}
		result = append(result, item)
	}
	return result
}

func optionalUTC(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil || value.Location() != time.UTC {
		return nil, errors.New("timestamp must use UTC")
	}
	return &value, nil
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}
