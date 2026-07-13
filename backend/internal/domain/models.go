package domain

import (
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusMerged   Status = "merged"
)

type EntityType string

const (
	EntityTypeAllianceOrg   EntityType = "alliance_org"
	EntityTypeEconomy       EntityType = "economy"
	EntityTypePolicyBody    EntityType = "policy_body"
	EntityTypeMarket        EntityType = "market"
	EntityTypeIndex         EntityType = "index"
	EntityTypeBenchmark     EntityType = "benchmark"
	EntityTypeSector        EntityType = "sector"
	EntityTypeIndustryChain EntityType = "industry_chain"
	EntityTypeChainNode     EntityType = "chain_node"
	EntityTypeTheme         EntityType = "theme"
	EntityTypeCompany       EntityType = "company"
	EntityTypeSecurity      EntityType = "security"
	EntityTypeInstrument    EntityType = "instrument"
	EntityTypeMetric        EntityType = "metric"
	EntityTypeCommodity     EntityType = "commodity"
	EntityTypePerson        EntityType = "person"
)

type EntityNode struct {
	ID            string
	EntityType    EntityType
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
	if !validEntityType(e.EntityType) {
		return fmt.Errorf("unsupported entity type %q", e.EntityType)
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

type AllianceOrgProfile struct {
	EntityID      string
	OrgCode       string
	OrgType       string
	PrimaryDomain string
	ScopeRegion   string
	OfficialURL   string
}

func (p AllianceOrgProfile) Validate() error {
	if p.EntityID == "" {
		return fmt.Errorf("entity id is required")
	}
	if p.OrgCode == "" {
		return fmt.Errorf("org code is required")
	}
	if p.OrgType == "" {
		return fmt.Errorf("org type is required")
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

type BenchmarkType string

const (
	BenchmarkTypeGovernmentBondYield BenchmarkType = "government_bond_yield"
	BenchmarkTypeFuturesPrice        BenchmarkType = "futures_price"
	BenchmarkTypeSpotPrice           BenchmarkType = "spot_price"
	BenchmarkTypeReferenceRate       BenchmarkType = "reference_rate"
)

type BenchmarkProfile struct {
	EntityID           string
	BenchmarkType      BenchmarkType
	OfficialSeriesCode string
	Provider           string
	Tenor              string
	UnderlyingSymbol   string
	CurrencyCode       string
	Unit               string
	Frequency          string
	SourceURL          string
}

func (p BenchmarkProfile) Validate() error {
	if p.EntityID == "" {
		return fmt.Errorf("entity id is required")
	}
	if !validStatus(
		p.BenchmarkType,
		BenchmarkTypeGovernmentBondYield,
		BenchmarkTypeFuturesPrice,
		BenchmarkTypeSpotPrice,
		BenchmarkTypeReferenceRate,
	) {
		return fmt.Errorf("unsupported benchmark type %q", p.BenchmarkType)
	}
	if p.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if p.CurrencyCode == "" {
		return fmt.Errorf("currency code is required")
	}
	if p.Unit == "" {
		return fmt.Errorf("unit is required")
	}
	if p.Frequency == "" {
		return fmt.Errorf("frequency is required")
	}
	if p.SourceURL == "" {
		return fmt.Errorf("source url is required")
	}
	return nil
}

type BenchmarkObservationQualityStatus string

const (
	BenchmarkObservationQualityRaw       BenchmarkObservationQualityStatus = "raw"
	BenchmarkObservationQualityValidated BenchmarkObservationQualityStatus = "validated"
	BenchmarkObservationQualitySuspect   BenchmarkObservationQualityStatus = "suspect"
	BenchmarkObservationQualityRejected  BenchmarkObservationQualityStatus = "rejected"
)

type BenchmarkObservation struct {
	ID                 string
	BenchmarkEntityID  string
	ObservedAt         time.Time
	Value              string
	Unit               string
	SourceName         string
	SourceURL          string
	ExternalSeriesCode string
	QualityStatus      BenchmarkObservationQualityStatus
}

func (o BenchmarkObservation) Validate() error {
	if o.ID == "" {
		return fmt.Errorf("observation id is required")
	}
	if o.BenchmarkEntityID == "" {
		return fmt.Errorf("benchmark entity id is required")
	}
	if o.ObservedAt.IsZero() {
		return fmt.Errorf("observed at is required")
	}
	if o.Value == "" {
		return fmt.Errorf("value is required")
	}
	if o.Unit == "" {
		return fmt.Errorf("unit is required")
	}
	if o.SourceName == "" {
		return fmt.Errorf("source name is required")
	}
	if !validStatus(
		o.QualityStatus,
		BenchmarkObservationQualityRaw,
		BenchmarkObservationQualityValidated,
		BenchmarkObservationQualitySuspect,
		BenchmarkObservationQualityRejected,
	) {
		return fmt.Errorf("unsupported benchmark observation quality status %q", o.QualityStatus)
	}
	return nil
}

type SectorProfile struct {
	EntityID               string
	SectorSystem           string
	SectorCode             string
	SectorType             string
	ExchangeScope          string
	ConstituentCount       int
	ListDate               *time.Time
	ParentSectorEntityID   string
	ClassificationCode     SectorClassification
	PrimaryMarketEntityID  string
	PrimaryEconomyEntityID string
	MethodologyURL         string
	ReviewStatus           SectorReviewStatus
}

type SectorClassification string

const (
	SectorClassificationIndustry SectorClassification = "industry_sector"
	SectorClassificationTheme    SectorClassification = "theme_sector"
	SectorClassificationMarket   SectorClassification = "market_sector"
	SectorClassificationStyle    SectorClassification = "style_sector"
	SectorClassificationRegion   SectorClassification = "region_sector"
)

type SectorReviewStatus string

const (
	SectorReviewCandidate SectorReviewStatus = "candidate"
	SectorReviewApproved  SectorReviewStatus = "approved"
	SectorReviewRejected  SectorReviewStatus = "rejected"
)

func (p SectorProfile) Validate() error {
	if p.EntityID == "" {
		return fmt.Errorf("entity id is required")
	}
	if !validStatus(p.ClassificationCode, SectorClassificationIndustry, SectorClassificationTheme, SectorClassificationMarket, SectorClassificationStyle, SectorClassificationRegion) {
		return fmt.Errorf("unsupported sector classification %q", p.ClassificationCode)
	}
	if !validStatus(p.ReviewStatus, SectorReviewCandidate, SectorReviewApproved, SectorReviewRejected) {
		return fmt.Errorf("unsupported sector review status %q", p.ReviewStatus)
	}
	return nil
}

type SectorSourceTaxonomyType string

const (
	SectorSourceTaxonomyConcept     SectorSourceTaxonomyType = "concept"
	SectorSourceTaxonomyIndustry    SectorSourceTaxonomyType = "industry"
	SectorSourceTaxonomyIndexSector SectorSourceTaxonomyType = "index_sector"
)

type SectorSourceMappingStatus string

const (
	SectorSourceMappingCandidate SectorSourceMappingStatus = "candidate"
	SectorSourceMappingApproved  SectorSourceMappingStatus = "approved"
	SectorSourceMappingRejected  SectorSourceMappingStatus = "rejected"
	SectorSourceMappingMerged    SectorSourceMappingStatus = "merged"
)

type SectorSourceMapping struct {
	ID                         string
	SectorEntityID             string
	SourceSystem               string
	SourceTaxonomyType         SectorSourceTaxonomyType
	SourceSectorCode           string
	SourceSectorName           string
	SourceSectorNameNormalized string
	SourceMarketScope          string
	SourceURL                  string
	RankSnapshot               int
	SnapshotDate               *time.Time
	MappingStatus              SectorSourceMappingStatus
	ReviewNote                 string
}

func (m SectorSourceMapping) Validate() error {
	if m.ID == "" || m.SectorEntityID == "" || m.SourceSystem == "" || m.SourceSectorName == "" || m.SourceSectorNameNormalized == "" {
		return fmt.Errorf("sector source mapping identity fields are required")
	}
	if !validStatus(m.SourceTaxonomyType, SectorSourceTaxonomyConcept, SectorSourceTaxonomyIndustry, SectorSourceTaxonomyIndexSector) {
		return fmt.Errorf("unsupported source taxonomy type %q", m.SourceTaxonomyType)
	}
	if !validStatus(m.MappingStatus, SectorSourceMappingCandidate, SectorSourceMappingApproved, SectorSourceMappingRejected, SectorSourceMappingMerged) {
		return fmt.Errorf("unsupported sector source mapping status %q", m.MappingStatus)
	}
	return nil
}

type ChainNodeProfile struct {
	EntityID     string
	Definition   string
	BoundaryNote string
}

func (p ChainNodeProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" || strings.TrimSpace(p.Definition) == "" {
		return fmt.Errorf("chain node identity and definition are required")
	}
	if p.BoundaryNote != "" && strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("chain node boundary note must be nonblank when present")
	}
	return nil
}

type Theme struct {
	EntityNode
}

type ThemeProfile struct {
	EntityID     string
	Definition   string
	BoundaryNote string
}

func (p ThemeProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" || strings.TrimSpace(p.Definition) == "" || strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("theme identity, definition, and boundary note are required")
	}
	return nil
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
	SourceConfig    map[string]any
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

type SchedulerMode string

const (
	SchedulerModeInterval   SchedulerMode = "interval"
	SchedulerModeFixedTimes SchedulerMode = "fixed_times"
)

type SchedulerTriggerType string

const (
	SchedulerTriggerManualOnce SchedulerTriggerType = "manual_once"
	SchedulerTriggerInterval   SchedulerTriggerType = "interval"
	SchedulerTriggerFixedTime  SchedulerTriggerType = "fixed_time"
)

type SchedulerRunStatus string

const (
	SchedulerRunStatusRunning   SchedulerRunStatus = "running"
	SchedulerRunStatusSucceeded SchedulerRunStatus = "succeeded"
	SchedulerRunStatusFailed    SchedulerRunStatus = "failed"
	SchedulerRunStatusPartial   SchedulerRunStatus = "partial"
	SchedulerRunStatusSkipped   SchedulerRunStatus = "skipped"
)

type SchedulerSourceRunStatus string

const (
	SchedulerSourceRunStatusSucceeded SchedulerSourceRunStatus = "succeeded"
	SchedulerSourceRunStatusFailed    SchedulerSourceRunStatus = "failed"
	SchedulerSourceRunStatusSkipped   SchedulerSourceRunStatus = "skipped"
)

type SchedulerSourceFilter struct {
	ProviderKey   string
	IngestChannel string
	SourceType    string
}

type SchedulerConfig struct {
	ID              string
	Enabled         bool
	Mode            SchedulerMode
	IntervalMinutes int
	FixedTimes      []string
	Concurrency     int
	BatchSize       int
	TimeoutSeconds  int
	SourceFilter    SchedulerSourceFilter
	Timezone        string
	ConfigVersion   int
	LastRunID       string
	LastRunAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (c SchedulerConfig) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("scheduler config id is required")
	}
	if !validStatus(c.Mode, SchedulerModeInterval, SchedulerModeFixedTimes) {
		return fmt.Errorf("unsupported scheduler mode %q", c.Mode)
	}
	if c.Mode == SchedulerModeInterval && c.IntervalMinutes <= 0 {
		return fmt.Errorf("interval minutes must be positive")
	}
	if c.Mode == SchedulerModeFixedTimes {
		if err := validateFixedTimes(c.FixedTimes); err != nil {
			return err
		}
	}
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if c.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout seconds must be positive")
	}
	if strings.TrimSpace(c.Timezone) == "" {
		return fmt.Errorf("timezone is required")
	}
	return nil
}

type IngestionRun struct {
	ID               string
	TriggerType      SchedulerTriggerType
	Status           SchedulerRunStatus
	StartedAt        time.Time
	FinishedAt       *time.Time
	TotalSources     int
	SucceededSources int
	FailedSources    int
	SkippedSources   int
	SchedulerConfig  map[string]any
	ErrorSummary     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (r IngestionRun) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("ingestion run id is required")
	}
	if !validStatus(r.TriggerType, SchedulerTriggerManualOnce, SchedulerTriggerInterval, SchedulerTriggerFixedTime) {
		return fmt.Errorf("unsupported scheduler trigger type %q", r.TriggerType)
	}
	if !validStatus(r.Status, SchedulerRunStatusRunning, SchedulerRunStatusSucceeded, SchedulerRunStatusFailed, SchedulerRunStatusPartial, SchedulerRunStatusSkipped) {
		return fmt.Errorf("unsupported scheduler run status %q", r.Status)
	}
	if r.StartedAt.IsZero() {
		return fmt.Errorf("started at is required")
	}
	if r.TotalSources < 0 || r.SucceededSources < 0 || r.FailedSources < 0 || r.SkippedSources < 0 {
		return fmt.Errorf("run counts must be non-negative")
	}
	return nil
}

type IngestionRunSource struct {
	ID                 string
	RunID              string
	SourceID           string
	Status             SchedulerSourceRunStatus
	DocumentsWritten   int
	DocumentsDuplicate int
	ErrorMessage       string
	StartedAt          time.Time
	FinishedAt         *time.Time
	DurationMillis     int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (s IngestionRunSource) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("ingestion run source id is required")
	}
	if s.RunID == "" {
		return fmt.Errorf("run id is required")
	}
	if s.SourceID == "" {
		return fmt.Errorf("source id is required")
	}
	if !validStatus(s.Status, SchedulerSourceRunStatusSucceeded, SchedulerSourceRunStatusFailed, SchedulerSourceRunStatusSkipped) {
		return fmt.Errorf("unsupported source run status %q", s.Status)
	}
	if s.StartedAt.IsZero() {
		return fmt.Errorf("started at is required")
	}
	if s.DocumentsWritten < 0 || s.DocumentsDuplicate < 0 || s.DurationMillis < 0 {
		return fmt.Errorf("source run counts must be non-negative")
	}
	return nil
}

func validateFixedTimes(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("fixed times are required")
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, err := time.Parse("15:04", value); err != nil {
			return fmt.Errorf("fixed time %q must use HH:mm format", value)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("fixed time %q is duplicated", value)
		}
		seen[value] = struct{}{}
	}
	return nil
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
	ReviewStatusCandidate ReviewStatus = "candidate"
	ReviewStatusReviewed  ReviewStatus = "reviewed"
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusApproved  ReviewStatus = "approved"
	ReviewStatusRejected  ReviewStatus = "rejected"
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

func validEntityType(value EntityType) bool {
	return validStatus(
		value,
		EntityTypeAllianceOrg,
		EntityTypeEconomy,
		EntityTypePolicyBody,
		EntityTypeMarket,
		EntityTypeIndex,
		EntityTypeBenchmark,
		EntityTypeSector,
		EntityTypeIndustryChain,
		EntityTypeChainNode,
		EntityTypeTheme,
		EntityTypeCompany,
		EntityTypeSecurity,
		EntityTypeInstrument,
		EntityTypeMetric,
		EntityTypeCommodity,
		EntityTypePerson,
	)
}
