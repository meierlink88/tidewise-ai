package event

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type Store interface {
	PublicationStore
	CreateEvent(context.Context, Aggregate) error
	EventByID(context.Context, string) (Aggregate, error)
	ListEvents(context.Context, EventListFilter) (EventStorePage, error)
}

type UseCase struct{ store Store }

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Event store is required")
	}
	return &UseCase{store: store}, nil
}

type Modality string

const (
	ModalityFact Modality = "FACT"
	ModalityPlan Modality = "PLAN"
	ModalitySpec Modality = "SPEC"
)

type LifecycleStatus string

const (
	LifecycleStatusActive     LifecycleStatus = "ACTIVE"
	LifecycleStatusDeprecated LifecycleStatus = "DEPRECATED"
	LifecycleStatusArchived   LifecycleStatus = "ARCHIVED"
)

type EventStage string

const (
	EventStageOccurred    EventStage = "OCCURRED"
	EventStageAnnounced   EventStage = "ANNOUNCED"
	EventStageEffective   EventStage = "EFFECTIVE"
	EventStageImplemented EventStage = "IMPLEMENTED"
	EventStageUpdated     EventStage = "UPDATED"
	EventStageSuspended   EventStage = "SUSPENDED"
	EventStageTerminated  EventStage = "TERMINATED"
	EventStageExpected    EventStage = "EXPECTED"
)

type TimePrecision string

const (
	TimePrecisionInstant TimePrecision = "INSTANT"
	TimePrecisionDay     TimePrecision = "DAY"
	TimePrecisionRange   TimePrecision = "RANGE"
	TimePrecisionMonth   TimePrecision = "MONTH"
	TimePrecisionQuarter TimePrecision = "QUARTER"
	TimePrecisionYear    TimePrecision = "YEAR"
	TimePrecisionUnknown TimePrecision = "UNKNOWN"
)

// Semantic is the canonical business proposition for one real-world Event.
type Semantic struct {
	Actors        []string   `json:"actors"`
	Action        string     `json:"action"`
	Objects       []string   `json:"objects"`
	Stage         EventStage `json:"stage"`
	Modality      Modality   `json:"modality"`
	Time          EventTime  `json:"time"`
	Jurisdictions []string   `json:"jurisdictions"`
	Reason        *string    `json:"reason"`
	Method        *string    `json:"method"`
	Metrics       []Metric   `json:"metrics"`
}

type EventTime struct {
	OccurredAt  *time.Time    `json:"occurred_at"`
	AnnouncedAt *time.Time    `json:"announced_at"`
	EffectiveAt *time.Time    `json:"effective_at"`
	ObservedAt  *time.Time    `json:"observed_at,omitempty"`
	Precision   TimePrecision `json:"precision"`
}

type Metric struct {
	Name   string  `json:"name"`
	Value  *string `json:"value"`
	Unit   *string `json:"unit"`
	Change *string `json:"change"`
	Period *string `json:"period"`
}

type Event struct {
	ID       string
	Title    string
	Summary  string
	Semantic Semantic
	Status   LifecycleStatus
}

type EvidenceLink struct {
	ID                 string
	EventID            string
	EvidenceID         string
	ContributionWeight float64
}

type ActorType string

const (
	ActorTypeCountry      ActorType = "COUNTRY"
	ActorTypePerson       ActorType = "PERSON"
	ActorTypeOrganization ActorType = "ORGANIZATION"
	ActorTypeCompany      ActorType = "COMPANY"
)

type ActorRelationType string

const (
	ActorRelationMentions       ActorRelationType = "MENTIONS"
	ActorRelationAffects        ActorRelationType = "AFFECTS"
	ActorRelationOriginatesFrom ActorRelationType = "ORIGINATES_FROM"
	ActorRelationTargets        ActorRelationType = "TARGETS"
)

type ActorLink struct {
	ID               string
	EventID          string
	ActorID          string
	ActorType        ActorType
	ActorName        *string
	RelationType     ActorRelationType
	RelationStrength *float64
	Confidence       float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type AssetType string

const (
	AssetTypeSecurity   AssetType = "SECURITY"
	AssetTypeCommodity  AssetType = "COMMODITY"
	AssetTypeIndex      AssetType = "INDEX"
	AssetTypeRate       AssetType = "RATE"
	AssetTypeForex      AssetType = "FOREX"
	AssetTypeDerivative AssetType = "DERIVATIVE"
)

type ImpactDirection string

const (
	ImpactDirectionPositive ImpactDirection = "POSITIVE"
	ImpactDirectionNegative ImpactDirection = "NEGATIVE"
	ImpactDirectionNeutral  ImpactDirection = "NEUTRAL"
)

type AssetLink struct {
	ID              string
	EventID         string
	AssetID         string
	AssetType       AssetType
	AssetName       *string
	ImpactDirection ImpactDirection
	ImpactMagnitude *float64
}

type Aggregate struct {
	Event    Event
	Evidence []EvidenceLink
	Actors   []ActorLink
	Assets   []AssetLink
}

type EvidenceLinkInput struct {
	EvidenceID         string
	ContributionWeight float64
}

type ActorLinkInput struct {
	ActorID          string
	ActorType        ActorType
	ActorName        *string
	RelationType     ActorRelationType
	RelationStrength *float64
	Confidence       *float64
}

type AssetLinkInput struct {
	AssetID         string
	AssetType       AssetType
	AssetName       *string
	ImpactDirection ImpactDirection
	ImpactMagnitude *float64
}

type CreateInput struct {
	Title    string
	Summary  string
	Semantic Semantic
	Status   LifecycleStatus
	Evidence []EvidenceLinkInput
	Actors   []ActorLinkInput
	Assets   []AssetLinkInput
}

type PublicationReceipt struct {
	ID, PublisherSubject, PublicationKey, PayloadHash, EventID string
	PublishedAt                                                time.Time
}

type PublicationResult struct {
	Event       Event
	Evidence    []EvidenceLink
	ReceiptID   string
	PayloadHash string
	Replayed    bool
}

var ErrPublicationPayloadConflict = errors.New("Event publication key conflicts with another payload")

// Publish atomically persists an Event and its initial Evidence links. Replays
// return the original Event and never create another Event or Evidence link.
func (s *UseCase) Publish(ctx context.Context, publisher, publicationKey string, input CreateInput) (PublicationResult, error) {
	if s == nil || s.store == nil {
		return PublicationResult{}, errors.New("Event store is required")
	}
	publisher, publicationKey = strings.TrimSpace(publisher), strings.TrimSpace(publicationKey)
	if publisher == "" || utf8.RuneCountInString(publisher) > 200 {
		return PublicationResult{}, invalidEvent("publisher subject must contain 1..200 characters")
	}
	if publicationKey == "" || utf8.RuneCountInString(publicationKey) > 200 {
		return PublicationResult{}, invalidEvent("publication key must contain 1..200 characters")
	}
	if input.Status == "" {
		input.Status = LifecycleStatusActive
	}
	if err := validateCreateInput(input); err != nil {
		return PublicationResult{}, err
	}
	aggregate, err := buildAggregate(input)
	if err != nil {
		return PublicationResult{}, err
	}
	hashPayload := struct {
		Input CreateInput `json:"event"`
	}{Input: input}
	encoded, err := json.Marshal(hashPayload)
	if err != nil {
		return PublicationResult{}, fmt.Errorf("encode Event publication hash input: %w", err)
	}
	digest := sha256.Sum256(encoded)
	payloadHash := hex.EncodeToString(digest[:])
	receiptID, err := coreid.New(coreid.EventPublicationReceipt)
	if err != nil {
		return PublicationResult{}, fmt.Errorf("generate Event publication receipt ID: %w", err)
	}
	receipt := PublicationReceipt{ID: receiptID, PublisherSubject: publisher, PublicationKey: publicationKey,
		PayloadHash: payloadHash, EventID: aggregate.Event.ID, PublishedAt: time.Now().UTC().Truncate(time.Microsecond)}
	stored := receipt
	replayed := false
	err = s.store.InEventPublicationTransaction(ctx, func(tx PublicationTransaction) error {
		if err := tx.Lock(ctx, publisher+":"+publicationKey); err != nil {
			return err
		}
		existing, err := tx.Receipt(ctx, publisher, publicationKey)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.PayloadHash != payloadHash {
				return ErrPublicationPayloadConflict
			}
			stored, replayed = *existing, true
			return nil
		}
		evidenceIDs := make([]string, len(aggregate.Evidence))
		for index, link := range aggregate.Evidence {
			evidenceIDs[index] = link.EvidenceID
		}
		existingEvidenceIDs, err := tx.ExistingEvidenceIDs(ctx, evidenceIDs)
		if err != nil {
			return err
		}
		if len(existingEvidenceIDs) != len(evidenceIDs) {
			return &ReferenceError{
				Field:   "evidence_ids",
				Message: "contains an Evidence identity that is not published",
			}
		}
		if err := tx.InsertAggregate(ctx, aggregate); err != nil {
			return err
		}
		return tx.InsertReceipt(ctx, receipt)
	})
	if err != nil {
		return PublicationResult{}, err
	}
	if replayed {
		aggregate, err = s.store.EventByID(ctx, stored.EventID)
		if err != nil {
			return PublicationResult{}, fmt.Errorf("read replayed Event: %w", err)
		}
	}
	return PublicationResult{Event: aggregate.Event, Evidence: aggregate.Evidence, ReceiptID: stored.ID,
		PayloadHash: stored.PayloadHash, Replayed: replayed}, nil
}

var ErrEventNotFound = errors.New("Event was not found")

func (s *UseCase) Create(ctx context.Context, input CreateInput) (Aggregate, error) {
	if s == nil || s.store == nil {
		return Aggregate{}, errors.New("Event store is required")
	}
	if input.Status == "" {
		input.Status = LifecycleStatusActive
	}
	if err := validateCreateInput(input); err != nil {
		return Aggregate{}, err
	}
	aggregate, err := buildAggregate(input)
	if err != nil {
		return Aggregate{}, err
	}
	eventID := aggregate.Event.ID
	if err := s.store.CreateEvent(ctx, aggregate); err != nil {
		return Aggregate{}, fmt.Errorf("create Event: %w", err)
	}
	return s.Get(ctx, eventID)
}

func buildAggregate(input CreateInput) (Aggregate, error) {
	eventID, err := coreid.New(coreid.Event)
	if err != nil {
		return Aggregate{}, fmt.Errorf("generate Event ID: %w", err)
	}
	aggregate := Aggregate{Event: Event{ID: eventID, Title: input.Title, Summary: input.Summary,
		Semantic: cloneSemantic(input.Semantic), Status: input.Status}}
	for _, item := range input.Evidence {
		linkID, idErr := coreid.New(coreid.EventEvidenceLink)
		if idErr != nil {
			return Aggregate{}, idErr
		}
		aggregate.Evidence = append(aggregate.Evidence, EvidenceLink{ID: linkID, EventID: eventID, EvidenceID: item.EvidenceID, ContributionWeight: item.ContributionWeight})
	}
	for _, item := range input.Actors {
		linkID, idErr := coreid.New(coreid.EventActorLink)
		if idErr != nil {
			return Aggregate{}, idErr
		}
		confidence := .70
		if item.Confidence != nil {
			confidence = *item.Confidence
		}
		aggregate.Actors = append(aggregate.Actors, ActorLink{ID: linkID, EventID: eventID, ActorID: item.ActorID,
			ActorType: item.ActorType, ActorName: cloneString(item.ActorName), RelationType: item.RelationType,
			RelationStrength: cloneFloat(item.RelationStrength), Confidence: confidence})
	}
	for _, item := range input.Assets {
		linkID, idErr := coreid.New(coreid.EventAssetLink)
		if idErr != nil {
			return Aggregate{}, idErr
		}
		aggregate.Assets = append(aggregate.Assets, AssetLink{ID: linkID, EventID: eventID, AssetID: item.AssetID,
			AssetType: item.AssetType, AssetName: cloneString(item.AssetName), ImpactDirection: item.ImpactDirection,
			ImpactMagnitude: cloneFloat(item.ImpactMagnitude)})
	}
	return aggregate, nil
}

func (s *UseCase) Get(ctx context.Context, eventID string) (Aggregate, error) {
	if s == nil || s.store == nil {
		return Aggregate{}, errors.New("Event store is required")
	}
	if !coreid.Is(eventID, coreid.Event) {
		return Aggregate{}, errors.New("Event ID is invalid")
	}
	return s.store.EventByID(ctx, eventID)
}

type EventListRequest struct {
	Title                      string
	Modality                   Modality
	Status                     LifecycleStatus
	OccurredFrom, OccurredTo   *time.Time
	AnnouncedFrom, AnnouncedTo *time.Time
	Page, PageSize             int
}

type EventListFilter = EventListRequest

type EventPage struct {
	Items    []Event
	Total    int
	Page     int
	PageSize int
}

type EventStorePage = EventPage

func (s *UseCase) ListEvents(ctx context.Context, request EventListRequest) (EventPage, error) {
	if s == nil || s.store == nil {
		return EventPage{}, errors.New("Event store is required")
	}
	if request.Modality != "" && !validStatus(request.Modality, ModalityFact, ModalityPlan, ModalitySpec) {
		return EventPage{}, errors.New("Event modality is invalid")
	}
	if request.Status != "" && !validStatus(request.Status, LifecycleStatusActive, LifecycleStatusDeprecated, LifecycleStatusArchived) {
		return EventPage{}, errors.New("Event status is invalid")
	}
	return s.store.ListEvents(ctx, request)
}

func validateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.Title) == "" || utf8.RuneCountInString(input.Title) > 200 {
		return invalidEvent("Event title is required and must contain at most 200 characters")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return invalidEvent("Event summary is required")
	}
	if len(input.Semantic.Actors) == 0 || len(input.Semantic.Objects) == 0 || strings.TrimSpace(input.Semantic.Action) == "" ||
		input.Semantic.Jurisdictions == nil || input.Semantic.Metrics == nil ||
		!validStatus(input.Semantic.Stage, EventStageOccurred, EventStageAnnounced, EventStageEffective, EventStageImplemented,
			EventStageUpdated, EventStageSuspended, EventStageTerminated, EventStageExpected) ||
		!validStatus(input.Semantic.Modality, ModalityFact, ModalityPlan, ModalitySpec) ||
		!validStatus(input.Semantic.Time.Precision, TimePrecisionInstant, TimePrecisionDay, TimePrecisionRange,
			TimePrecisionMonth, TimePrecisionQuarter, TimePrecisionYear, TimePrecisionUnknown) {
		return invalidEvent("Event semantic identity is invalid")
	}
	for _, values := range [][]string{input.Semantic.Actors, input.Semantic.Objects, input.Semantic.Jurisdictions} {
		seen := make(map[string]struct{}, len(values))
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return invalidEvent("Event semantic identity is invalid")
			}
			if _, duplicate := seen[value]; duplicate {
				return invalidEvent("Event semantic identity is duplicated")
			}
			seen[value] = struct{}{}
		}
	}
	if input.Semantic.Time.OccurredAt == nil && input.Semantic.Time.AnnouncedAt == nil &&
		input.Semantic.Time.EffectiveAt == nil && input.Semantic.Time.ObservedAt == nil {
		return invalidEvent("Event semantic requires at least one time anchor")
	}
	for _, metric := range input.Semantic.Metrics {
		if strings.TrimSpace(metric.Name) == "" || metric.Value == nil && metric.Change == nil {
			return invalidEvent("Event semantic metric is invalid")
		}
	}
	if !validStatus(input.Status, LifecycleStatusActive, LifecycleStatusDeprecated, LifecycleStatusArchived) {
		return invalidEvent("Event status is invalid")
	}
	if len(input.Evidence) == 0 {
		return invalidEvent("Event requires at least one Evidence Link")
	}
	seenEvidence := make(map[string]struct{}, len(input.Evidence))
	for _, link := range input.Evidence {
		if !coreid.Is(link.EvidenceID, coreid.Evidence) || !validUnitInterval(link.ContributionWeight) {
			return invalidEvent("Event Evidence Link is invalid")
		}
		if _, duplicate := seenEvidence[link.EvidenceID]; duplicate {
			return invalidEvent("Event Evidence Link is duplicated")
		}
		seenEvidence[link.EvidenceID] = struct{}{}
	}
	seenActors := make(map[string]struct{}, len(input.Actors))
	for _, link := range input.Actors {
		if strings.TrimSpace(link.ActorID) == "" || utf8.RuneCountInString(link.ActorID) > 64 ||
			(link.ActorType != "" && !validStatus(link.ActorType, ActorTypeCountry, ActorTypePerson, ActorTypeOrganization, ActorTypeCompany)) ||
			!validStatus(link.RelationType, ActorRelationMentions, ActorRelationAffects, ActorRelationOriginatesFrom, ActorRelationTargets) ||
			(link.RelationStrength != nil && !validUnitInterval(*link.RelationStrength)) ||
			(link.Confidence != nil && (!validUnitInterval(*link.Confidence) || *link.Confidence > 0.99)) ||
			(link.ActorName != nil && utf8.RuneCountInString(*link.ActorName) > 200) {
			return invalidEvent("Event Actor Link is invalid")
		}
		key := link.ActorID + "\x00" + string(link.RelationType)
		if _, duplicate := seenActors[key]; duplicate {
			return invalidEvent("Event Actor Link is duplicated")
		}
		seenActors[key] = struct{}{}
	}
	seenAssets := make(map[string]struct{}, len(input.Assets))
	for _, link := range input.Assets {
		if strings.TrimSpace(link.AssetID) == "" || utf8.RuneCountInString(link.AssetID) > 64 ||
			(link.AssetType != "" && !validStatus(link.AssetType, AssetTypeSecurity, AssetTypeCommodity, AssetTypeIndex, AssetTypeRate, AssetTypeForex, AssetTypeDerivative)) ||
			!validStatus(link.ImpactDirection, ImpactDirectionPositive, ImpactDirectionNegative, ImpactDirectionNeutral) ||
			(link.ImpactMagnitude != nil && !validUnitInterval(*link.ImpactMagnitude)) ||
			(link.AssetName != nil && utf8.RuneCountInString(*link.AssetName) > 200) {
			return invalidEvent("Event Asset Link is invalid")
		}
		if _, duplicate := seenAssets[link.AssetID]; duplicate {
			return invalidEvent("Event Asset Link is duplicated")
		}
		seenAssets[link.AssetID] = struct{}{}
	}
	return nil
}

func validUnitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func validStatus[T comparable](value T, allowed ...T) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneSemantic(value Semantic) Semantic {
	return Semantic{Actors: cloneStrings(value.Actors), Action: value.Action,
		Objects: cloneStrings(value.Objects), Stage: value.Stage, Modality: value.Modality,
		Time: EventTime{OccurredAt: cloneTime(value.Time.OccurredAt), AnnouncedAt: cloneTime(value.Time.AnnouncedAt),
			EffectiveAt: cloneTime(value.Time.EffectiveAt), ObservedAt: cloneTime(value.Time.ObservedAt),
			Precision: value.Time.Precision},
		Jurisdictions: cloneStrings(value.Jurisdictions), Reason: cloneString(value.Reason),
		Method: cloneString(value.Method), Metrics: cloneMetrics(value.Metrics)}
}

func cloneMetrics(values []Metric) []Metric {
	if values == nil {
		return nil
	}
	result := make([]Metric, len(values))
	for index, value := range values {
		result[index] = Metric{Name: value.Name, Value: cloneString(value.Value), Unit: cloneString(value.Unit),
			Change: cloneString(value.Change), Period: cloneString(value.Period)}
	}
	return result
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
