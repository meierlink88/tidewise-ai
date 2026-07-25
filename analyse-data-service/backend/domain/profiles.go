package domain

import (
	"fmt"
	"strings"
	"time"
)

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
	ReviewStatus ReviewStatus
}

func (p ChainNodeProfile) Validate() error {
	if strings.TrimSpace(p.EntityID) == "" || strings.TrimSpace(p.Definition) == "" {
		return fmt.Errorf("chain node identity and definition are required")
	}
	if p.BoundaryNote != "" && strings.TrimSpace(p.BoundaryNote) == "" {
		return fmt.Errorf("chain node boundary note must be nonblank when present")
	}
	if p.ReviewStatus != "" && !validMasterDataReviewStatus(p.ReviewStatus) {
		return fmt.Errorf("unsupported chain node review status %q", p.ReviewStatus)
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
