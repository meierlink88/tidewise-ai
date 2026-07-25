package domain

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type EventStatus string

const (
	EventStatusCandidate EventStatus = "candidate"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusRejected  EventStatus = "rejected"
)

type FactStatus string

const (
	FactStatusUnverified FactStatus = "unverified"
	FactStatusVerified   FactStatus = "verified"
	FactStatusDisputed   FactStatus = "disputed"
)

type FactPayload map[string]any

var forbiddenFactPayloadKeys = map[string]struct{}{
	"buy":                       {},
	"sell":                      {},
	"buy_recommendation":        {},
	"sell_recommendation":       {},
	"investment_advice":         {},
	"direct_investment_advice":  {},
	"investment_recommendation": {},
	"recommendation":            {},
	"price_prediction":          {},
	"price_forecast":            {},
	"return_prediction":         {},
	"prediction":                {},
	"forecast":                  {},
	"event_score":               {},
	"score":                     {},
	"scoring":                   {},
	"transmission_strength":     {},
	"favorable":                 {},
	"unfavorable":               {},
	"bullish":                   {},
	"bearish":                   {},
}

func ValidateFactPayload(payload any) error {
	if payload == nil {
		return fmt.Errorf("fact payload must be a JSON object")
	}

	var object map[string]any
	switch value := payload.(type) {
	case FactPayload:
		object = map[string]any(value)
	case map[string]any:
		object = value
	default:
		return fmt.Errorf("fact payload must be a JSON object")
	}
	if object == nil {
		return fmt.Errorf("fact payload must be a JSON object")
	}

	if _, err := json.Marshal(object); err != nil {
		return fmt.Errorf("fact payload must be JSON encodable: %w", err)
	}
	for key := range object {
		normalizedKey := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToLower(strings.TrimSpace(key)))
		if _, forbidden := forbiddenFactPayloadKeys[normalizedKey]; forbidden {
			return fmt.Errorf("fact payload key %q is not allowed", key)
		}
	}
	return nil
}

type Event struct {
	ID              string
	Title           string
	Summary         string
	EventTime       *time.Time
	FirstSeenAt     time.Time
	KnowableAt      *time.Time
	EventStatus     EventStatus
	FactStatus      FactStatus
	DedupeKey       string
	PrimarySourceID string
	FactPayload     FactPayload
}

func (e Event) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("event id is required")
	}
	if e.Title == "" {
		return fmt.Errorf("title is required")
	}
	if e.FirstSeenAt.IsZero() {
		return fmt.Errorf("first seen at is required")
	}
	if e.DedupeKey == "" {
		return fmt.Errorf("dedupe key is required")
	}
	if !validStatus(e.EventStatus, EventStatusCandidate, EventStatusConfirmed, EventStatusRejected) {
		return fmt.Errorf("unsupported event status %q", e.EventStatus)
	}
	if !validStatus(e.FactStatus, FactStatusUnverified, FactStatusVerified, FactStatusDisputed) {
		return fmt.Errorf("unsupported fact status %q", e.FactStatus)
	}
	if err := ValidateFactPayload(e.FactPayload); err != nil {
		return err
	}
	return nil
}

type EvidenceRelation string

const (
	EvidenceRelationSupports    EvidenceRelation = "supports"
	EvidenceRelationContradicts EvidenceRelation = "contradicts"
	EvidenceRelationContext     EvidenceRelation = "context"
)

type EventSource struct {
	ID               string
	EventID          string
	RawDocumentID    string
	SourceLevel      string
	EvidenceExcerpt  string
	EvidenceHash     string
	EvidenceRelation EvidenceRelation
	SupportsFields   []string
}

func (s EventSource) Validate() error {
	if s.EvidenceRelation == "" {
		return nil
	}
	if !validStatus(s.EvidenceRelation, EvidenceRelationSupports, EvidenceRelationContradicts, EvidenceRelationContext) {
		return fmt.Errorf("unsupported evidence relation %q", s.EvidenceRelation)
	}
	if s.EvidenceRelation != EvidenceRelationSupports && s.EvidenceRelation != EvidenceRelationContradicts {
		return nil
	}
	if len(s.SupportsFields) == 0 {
		return fmt.Errorf("supports fields are required for evidence relation %q", s.EvidenceRelation)
	}
	for _, field := range s.SupportsFields {
		if strings.TrimSpace(field) == "" {
			return fmt.Errorf("supports fields must not contain blank values")
		}
	}
	return nil
}

type EventTagDef struct {
	ID       string
	TagKind  string
	Code     string
	Name     string
	IsActive bool
}

type ReviewStatus string

const (
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusReviewed  ReviewStatus = "reviewed"
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusApproved  ReviewStatus = "approved"
	ReviewStatusRejected  ReviewStatus = "rejected"
)

type EventTagMap struct {
	ID               string
	EventID          string
	TagID            string
	AssignSource     string
	ReviewStatus     ReviewStatus
	Confidence       *float64
	AssignmentReason string
}

const (
	TagAssignSourceAI   = "ai"
	TagAssignSourceRule = "rule"
)

func (m EventTagMap) Validate() error {
	if m.Confidence != nil {
		if math.IsNaN(*m.Confidence) || math.IsInf(*m.Confidence, 0) || *m.Confidence < 0 || *m.Confidence > 1 {
			return fmt.Errorf("tag confidence must be between 0 and 1")
		}
	}
	assignSource := strings.ToLower(strings.TrimSpace(m.AssignSource))
	if (assignSource == TagAssignSourceAI || assignSource == TagAssignSourceRule) && strings.TrimSpace(m.AssignmentReason) == "" {
		return fmt.Errorf("assignment reason is required for %s tag assignment", assignSource)
	}
	return nil
}

type EventEntityLink struct {
	ID           string
	EventID      string
	EntityID     string
	EntityRole   string
	AssignSource string
	ReviewStatus ReviewStatus
	EvidenceNote string
}
