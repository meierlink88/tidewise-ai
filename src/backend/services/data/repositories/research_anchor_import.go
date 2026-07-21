package repositories

import (
	"context"
	"time"
)

type ResearchAnchorImportStore interface {
	InResearchAnchorImportTransaction(context.Context, func(ResearchAnchorImportTransaction) error) error
}

type ResearchAnchorImportTransaction interface {
	LockResearchAnchorImportTheme(context.Context, string) error
	ResearchAnchorImportReceipt(context.Context, string) (*ResearchAnchorImportReceipt, error)
	ResearchAnchorImportThemePublication(context.Context, string) (*ResearchAnchorImportThemePublication, error)
	ResearchAnchorImportThemeChainNodes(context.Context, string) (map[string]struct{}, error)
	ResearchAnchorImportThemeEvents(context.Context, string) (map[string]struct{}, error)
	ExistingResearchAnchorChainNodes(context.Context, []string) (map[string]struct{}, error)
	ExistingResearchAnchorEvents(context.Context, []string) (map[string]struct{}, error)
	InsertResearchAnchorImportReceipt(context.Context, ResearchAnchorImportReceipt) error
	InsertResearchAnchor(context.Context, ResearchAnchorImportAnchor) error
	InsertResearchAnchorEvent(context.Context, ResearchAnchorImportEvent) error
	InsertResearchAnchorPathNode(context.Context, ResearchAnchorImportPathNode) error
	VerifyResearchAnchorImportReceipt(context.Context, ResearchAnchorImportReceipt) error
}

type ResearchAnchorImportThemePublication struct {
	ThemeID              string
	ThemeImportReceiptID string
	PublisherSubject     string
}

type ResearchAnchorImportAnchor struct {
	ID                      string
	ThemeID                 string
	CenterChainNodeEntityID string
	ImportReceiptID         string
	OneLineConclusion       string
	FactSummary             string
	NetDirectionSummary     string
	SupportSummary          string
	CounterSummary          *string
	TradingDirection        string
	NextCheckpoint          string
}

type ResearchAnchorImportEvent struct {
	AnchorID        string
	EventID         string
	EvidenceRole    string
	EvidenceSummary string
}

type ResearchAnchorImportPathNode struct {
	AnchorID                      string
	Position                      int
	ChainNodeEntityID             string
	ChangeDirection               string
	ChangeSummary                 string
	ImpactSummary                 string
	IncomingTransmissionMechanism *string
}

type ResearchAnchorImportCounts struct {
	Anchors           int `json:"anchors"`
	EventAssociations int `json:"event_associations"`
	PathNodes         int `json:"path_nodes"`
	Receipts          int `json:"receipts"`
}

type ResearchAnchorImportReceipt struct {
	ID                           string
	ThemeID                      string
	PublisherSubject             string
	PayloadHash                  string
	AnchorIDsByCenterChainNodeID map[string]string
	Counts                       ResearchAnchorImportCounts
	PublishedAt                  time.Time
	ImportedAt                   time.Time
}
