package repositories

import (
	"context"
	"time"
)

type ResearchThemeImportStore interface {
	InResearchThemeImportTransaction(context.Context, func(ResearchThemeImportTransaction) error) error
}

type ResearchThemeImportTransaction interface {
	LockResearchThemeImportBatch(context.Context, string) error
	ResearchThemeImportReceipt(context.Context, string) (*ResearchThemeImportReceipt, error)
	ExistingResearchThemeChainNodes(context.Context, []string) (map[string]struct{}, error)
	ExistingResearchThemeEvents(context.Context, []string) (map[string]struct{}, error)
	InsertResearchTheme(context.Context, ResearchThemeImportTheme) error
	InsertResearchThemeChainNode(context.Context, ResearchThemeImportChainNode) error
	InsertResearchThemeEvent(context.Context, ResearchThemeImportEvent) error
	InsertResearchThemeImportReceipt(context.Context, ResearchThemeImportReceipt) error
	VerifyResearchThemeImportReceipt(context.Context, ResearchThemeImportReceipt) error
}

type ResearchThemeImportTheme struct {
	ID                        string
	ImportReceiptID           string
	AnalysisBatchID           string
	ThemeKey                  string
	Name                      string
	OneLineConclusion         string
	ImpactLevel               string
	TransmissionPath          string
	TradingDirection          string
	TransmissionStage         string
	NextCheckpoint            string
	MarketConfirmationSummary string
	WindowStart               time.Time
	WindowEnd                 time.Time
	PublishedAt               time.Time
}

type ResearchThemeImportChainNode struct {
	ThemeID           string
	ChainNodeEntityID string
	RelationRole      string
	ImpactSummary     string
}

type ResearchThemeImportEvent struct {
	ThemeID        string
	EventID        string
	EvidenceRole   string
	SupportedClaim string
}

type ResearchThemeImportCounts struct {
	Themes                int `json:"themes"`
	ChainNodeAssociations int `json:"chain_node_associations"`
	EventAssociations     int `json:"event_associations"`
	Receipts              int `json:"receipts"`
}

type ResearchThemeImportReceipt struct {
	ID               string
	AnalysisBatchID  string
	PublisherSubject string
	PayloadHash      string
	ThemeIDsByKey    map[string]string
	Counts           ResearchThemeImportCounts
	PublishedAt      time.Time
	ImportedAt       time.Time
}
