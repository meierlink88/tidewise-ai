package event

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type Store interface {
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

type Semantic struct {
	Who   *string `json:"who"`
	What  *string `json:"what"`
	When  *string `json:"when"`
	Where *string `json:"where"`
	Why   *string `json:"why"`
	How   *string `json:"how"`
}

type Event struct {
	ID          string
	Title       string
	Summary     string
	Semantic    Semantic
	Modality    Modality
	OccurredAt  *time.Time
	AnnouncedAt *time.Time
	Status      LifecycleStatus
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
	Title       string
	Summary     string
	Semantic    Semantic
	Modality    Modality
	OccurredAt  *time.Time
	AnnouncedAt *time.Time
	Status      LifecycleStatus
	Evidence    []EvidenceLinkInput
	Actors      []ActorLinkInput
	Assets      []AssetLinkInput
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
	eventID, err := coreid.New(coreid.Event)
	if err != nil {
		return Aggregate{}, fmt.Errorf("generate Event ID: %w", err)
	}
	aggregate := Aggregate{Event: Event{
		ID: eventID, Title: input.Title, Summary: input.Summary, Semantic: input.Semantic,
		Modality: input.Modality, OccurredAt: cloneTime(input.OccurredAt),
		AnnouncedAt: cloneTime(input.AnnouncedAt), Status: input.Status,
	}}
	for _, item := range input.Evidence {
		id, idErr := coreid.New(coreid.EventEvidenceLink)
		if idErr != nil {
			return Aggregate{}, fmt.Errorf("generate Event Evidence Link ID: %w", idErr)
		}
		aggregate.Evidence = append(aggregate.Evidence, EvidenceLink{
			ID: id, EventID: eventID, EvidenceID: item.EvidenceID, ContributionWeight: item.ContributionWeight,
		})
	}
	for _, item := range input.Actors {
		id, idErr := coreid.New(coreid.EventActorLink)
		if idErr != nil {
			return Aggregate{}, fmt.Errorf("generate Event Actor Link ID: %w", idErr)
		}
		confidence := 0.70
		if item.Confidence != nil {
			confidence = *item.Confidence
		}
		aggregate.Actors = append(aggregate.Actors, ActorLink{
			ID: id, EventID: eventID, ActorID: item.ActorID, ActorType: item.ActorType,
			ActorName: cloneString(item.ActorName), RelationType: item.RelationType,
			RelationStrength: cloneFloat(item.RelationStrength), Confidence: confidence,
		})
	}
	for _, item := range input.Assets {
		id, idErr := coreid.New(coreid.EventAssetLink)
		if idErr != nil {
			return Aggregate{}, fmt.Errorf("generate Event Asset Link ID: %w", idErr)
		}
		aggregate.Assets = append(aggregate.Assets, AssetLink{
			ID: id, EventID: eventID, AssetID: item.AssetID, AssetType: item.AssetType,
			AssetName: cloneString(item.AssetName), ImpactDirection: item.ImpactDirection,
			ImpactMagnitude: cloneFloat(item.ImpactMagnitude),
		})
	}
	if err := s.store.CreateEvent(ctx, aggregate); err != nil {
		return Aggregate{}, fmt.Errorf("create Event: %w", err)
	}
	return s.Get(ctx, eventID)
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
		return errors.New("Event title is required and must contain at most 200 characters")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return errors.New("Event summary is required")
	}
	if !validStatus(input.Modality, ModalityFact, ModalityPlan, ModalitySpec) {
		return errors.New("Event modality is invalid")
	}
	if !validStatus(input.Status, LifecycleStatusActive, LifecycleStatusDeprecated, LifecycleStatusArchived) {
		return errors.New("Event status is invalid")
	}
	if len(input.Evidence) == 0 {
		return errors.New("Event requires at least one Evidence Link")
	}
	seenEvidence := make(map[string]struct{}, len(input.Evidence))
	for _, link := range input.Evidence {
		if !coreid.Is(link.EvidenceID, coreid.Evidence) || !validUnitInterval(link.ContributionWeight) {
			return errors.New("Event Evidence Link is invalid")
		}
		if _, duplicate := seenEvidence[link.EvidenceID]; duplicate {
			return errors.New("Event Evidence Link is duplicated")
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
			return errors.New("Event Actor Link is invalid")
		}
		key := link.ActorID + "\x00" + string(link.RelationType)
		if _, duplicate := seenActors[key]; duplicate {
			return errors.New("Event Actor Link is duplicated")
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
			return errors.New("Event Asset Link is invalid")
		}
		if _, duplicate := seenAssets[link.AssetID]; duplicate {
			return errors.New("Event Asset Link is duplicated")
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
