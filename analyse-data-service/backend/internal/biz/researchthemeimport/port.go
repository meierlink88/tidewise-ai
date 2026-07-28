package researchthemeimport

import (
	"context"
	"time"
)

type Store interface {
	InResearchThemeImportTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockResearchThemeImportBatch(context.Context, string) error
	ResearchThemeImportReceipt(context.Context, string) (*Receipt, error)
	ExistingResearchThemeImpactNodes(context.Context, []string) (map[string]struct{}, error)
	ExistingResearchThemeEvents(context.Context, []string) (map[string]struct{}, error)
	InsertResearchTheme(context.Context, ThemeRecord) error
	InsertResearchThemeImpact(context.Context, ImpactRecord) error
	InsertResearchThemeEvent(context.Context, EventRecord) error
	InsertResearchThemeImportReceipt(context.Context, Receipt) error
	VerifyResearchThemeImportReceipt(context.Context, Receipt) error
}

type ThemeRecord struct {
	ID, ImportReceiptID, AnalysisBatchID, ThemeKey, Title, OneLineConclusion string
	ConclusionDirection, ImpactStrength, TransmissionStage                   string
	InvestmentGuidanceAction, InvestmentGuidanceSummary                      string
	TimeHorizonCategory                                                      string
	AttentionLevel, ConclusionStatus                                         *string
	TimeHorizonSummary, TransmissionSummary, CheckpointSummary, RiskSummary  *string
	AnalysisAsOf, WindowStart, WindowEnd, PublishedAt                        time.Time
}

type ImpactRecord struct {
	ThemeID, ChainNodeEntityID, RelationRole, ImpactDirection string
	ImpactSummary                                             *string
	DisplayOrder                                              int
}

type EventRecord struct {
	ThemeID, EventID, EvidenceRole string
	SupportedClaim                 *string
}

type Counts struct {
	Themes            int `json:"themes"`
	Impacts           int `json:"impacts"`
	EventAssociations int `json:"event_associations"`
	Receipts          int `json:"receipts"`
}

type Receipt struct {
	ID, AnalysisBatchID, PublisherSubject, PayloadHash string
	ThemeIDsByKey                                      map[string]string
	Counts                                             Counts
	PublishedAt, ImportedAt                            time.Time
}
