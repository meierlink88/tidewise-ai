package domain

import (
	"fmt"
	"time"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusMerged   Status = "merged"
)

type EntityNode struct {
	ID            string
	EntityType    string
	LayerCode     string
	Name          string
	CanonicalName string
	Aliases       []string
	Status        Status
}

func (e EntityNode) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("entity id is required")
	}
	if e.EntityType == "" {
		return fmt.Errorf("entity type is required")
	}
	if e.LayerCode == "" {
		return fmt.Errorf("layer code is required")
	}
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.CanonicalName == "" {
		return fmt.Errorf("canonical name is required")
	}
	if !validStatus(e.Status, StatusActive, StatusInactive, StatusMerged) {
		return fmt.Errorf("unsupported entity status %q", e.Status)
	}
	return nil
}

type EntityEdge struct {
	ID           string
	FromEntityID string
	ToEntityID   string
	RelationType string
	EvidenceNote string
	Status       Status
}

type EconomyProfile struct {
	EntityID     string
	CountryCode  string
	CurrencyCode string
	Region       string
}

type PolicyBodyProfile struct {
	EntityID     string
	BodyType     string
	Jurisdiction string
	PolicyDomain string
}

type MarketProfile struct {
	EntityID        string
	MarketType      string
	EconomyEntityID string
	CurrencyCode    string
	Timezone        string
}

type IndexProfile struct {
	EntityID       string
	IndexCode      string
	IndexType      string
	MarketEntityID string
	Provider       string
	CurrencyCode   string
	ListDate       *time.Time
}

type SectorProfile struct {
	EntityID             string
	SectorSystem         string
	SectorCode           string
	SectorType           string
	ExchangeScope        string
	ConstituentCount     int
	ListDate             *time.Time
	ParentSectorEntityID string
}

type ChainNodeProfile struct {
	EntityID      string
	ChainPosition string
}

type CompanyProfile struct {
	EntityID                    string
	RegistrationEconomyEntityID string
	Area                        string
	IndustryName                string
	ControllerName              string
	ControllerType              string
}

type SecurityProfile struct {
	EntityID              string
	Ticker                string
	Symbol                string
	Exchange              string
	MarketBoard           string
	SecurityType          string
	IssuerCompanyEntityID string
	ListDate              *time.Time
	DelistDate            *time.Time
	ListStatus            string
	CurrencyCode          string
}

type InstrumentProfile struct {
	EntityID           string
	InstrumentType     string
	UnderlyingEntityID string
	Exchange           string
	CurrencyCode       string
}

type MetricProfile struct {
	EntityID   string
	MetricType string
	Unit       string
	Frequency  string
}

type CommodityProfile struct {
	EntityID      string
	CommodityType string
}

type PersonProfile struct {
	EntityID             string
	RoleTitle            string
	OrganizationEntityID string
	EconomyEntityID      string
}

type SourceCatalogStatus string

const (
	SourceCatalogStatusActive   SourceCatalogStatus = "active"
	SourceCatalogStatusInactive SourceCatalogStatus = "inactive"
	SourceCatalogStatusDisabled SourceCatalogStatus = "disabled"
)

type SourceCatalog struct {
	ID              string
	IngestChannel   string
	ProviderKey     string
	ConnectorKey    string
	ParserKey       string
	SourceType      string
	SourceName      string
	SourceURL       string
	SourceLevel     string
	TopicHint       string
	RouteTemplate   string
	CodeStyle       string
	AuthRequired    bool
	AuthType        string
	CredentialRef   string
	RateLimitPolicy map[string]any
	UsagePolicy     string
	Status          SourceCatalogStatus
}

type IngestStatus string

const (
	IngestStatusCollected      IngestStatus = "collected"
	IngestStatusDuplicate      IngestStatus = "duplicate"
	IngestStatusFailed         IngestStatus = "failed"
	IngestStatusPendingExtract IngestStatus = "pending_extract"
)

type RawDocument struct {
	ID               string
	SourceID         string
	IngestChannel    string
	SourceType       string
	SourceName       string
	SourceURL        string
	SourceExternalID string
	Title            string
	ContentText      string
	RawObjectURI     string
	RawMIMEType      string
	Language         string
	PublishedAt      *time.Time
	CollectedAt      time.Time
	ContentHash      string
	IngestStatus     IngestStatus
}

func (d RawDocument) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("raw document id is required")
	}
	if d.SourceID == "" {
		return fmt.Errorf("source id is required")
	}
	if d.IngestChannel == "" {
		return fmt.Errorf("ingest channel is required")
	}
	if d.SourceType == "" {
		return fmt.Errorf("source type is required")
	}
	if d.SourceName == "" {
		return fmt.Errorf("source name is required")
	}
	if d.Title == "" {
		return fmt.Errorf("title is required")
	}
	if d.ContentHash == "" {
		return fmt.Errorf("content hash is required")
	}
	if d.CollectedAt.IsZero() {
		return fmt.Errorf("collected at is required")
	}
	if !validStatus(d.IngestStatus, IngestStatusCollected, IngestStatusDuplicate, IngestStatusFailed, IngestStatusPendingExtract) {
		return fmt.Errorf("unsupported ingest status %q", d.IngestStatus)
	}
	return nil
}

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
	return nil
}

type EventSource struct {
	ID              string
	EventID         string
	RawDocumentID   string
	SourceLevel     string
	EvidenceExcerpt string
	EvidenceHash    string
}

type EventTagDef struct {
	ID      string
	TagKind string
	Code    string
	Name    string
}

type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusRejected ReviewStatus = "rejected"
)

type EventTagMap struct {
	ID           string
	EventID      string
	TagID        string
	AssignSource string
	ReviewStatus ReviewStatus
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

func validStatus[T comparable](value T, allowed ...T) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
