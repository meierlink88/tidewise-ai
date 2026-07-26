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
	ExistingResearchThemeChainNodes(context.Context, []string) (map[string]struct{}, error)
	ExistingResearchThemeEvents(context.Context, []string) (map[string]struct{}, error)
	InsertResearchTheme(context.Context, ThemeRecord) error
	InsertResearchThemeChainNode(context.Context, ChainNodeRecord) error
	InsertResearchThemeEvent(context.Context, EventRecord) error
	InsertResearchThemeImportReceipt(context.Context, Receipt) error
	VerifyResearchThemeImportReceipt(context.Context, Receipt) error
}

type ThemeRecord struct {
	ID, ImportReceiptID, AnalysisBatchID, ThemeKey, Name, OneLineConclusion string
	ImpactLevel, TransmissionPath, TradingDirection, TransmissionStage      string
	NextCheckpoint, MarketConfirmationSummary                               string
	WindowStart, WindowEnd, PublishedAt                                     time.Time
}

type ChainNodeRecord struct {
	ThemeID, ChainNodeEntityID, RelationRole, ImpactSummary string
}

type EventRecord struct {
	ThemeID, EventID, EvidenceRole, SupportedClaim string
}

type Counts struct {
	Themes                int `json:"themes"`
	ChainNodeAssociations int `json:"chain_node_associations"`
	EventAssociations     int `json:"event_associations"`
	Receipts              int `json:"receipts"`
}

type Receipt struct {
	ID, AnalysisBatchID, PublisherSubject, PayloadHash string
	ThemeIDsByKey                                      map[string]string
	Counts                                             Counts
	PublishedAt, ImportedAt                            time.Time
}
