package researchanchorimport

import (
	"context"
	"time"
)

type Store interface {
	InResearchAnchorImportTransaction(context.Context, func(Transaction) error) error
}

type Transaction interface {
	LockResearchAnchorImportTheme(context.Context, string) error
	ResearchAnchorImportReceipt(context.Context, string) (*Receipt, error)
	ResearchAnchorImportThemePublication(context.Context, string) (*ThemePublication, error)
	ResearchAnchorImportThemeChainNodes(context.Context, string) (map[string]struct{}, error)
	ResearchAnchorImportThemeEvents(context.Context, string) (map[string]struct{}, error)
	ExistingResearchAnchorChainNodes(context.Context, []string) (map[string]struct{}, error)
	ExistingResearchAnchorEvents(context.Context, []string) (map[string]struct{}, error)
	InsertResearchAnchorImportReceipt(context.Context, Receipt) error
	InsertResearchAnchor(context.Context, AnchorRecord) error
	InsertResearchAnchorEvent(context.Context, EventRecord) error
	InsertResearchAnchorPathNode(context.Context, PathNodeRecord) error
	VerifyResearchAnchorImportReceipt(context.Context, Receipt) error
}

type ThemePublication struct {
	ThemeID, ThemeImportReceiptID, PublisherSubject string
}

type AnchorRecord struct {
	ID, ThemeID, CenterChainNodeEntityID, ImportReceiptID string
	OneLineConclusion, FactSummary, NetDirectionSummary   string
	SupportSummary                                        string
	CounterSummary                                        *string
	TradingDirection, NextCheckpoint                      string
}

type EventRecord struct {
	AnchorID, EventID, EvidenceRole, EvidenceSummary string
}

type PathNodeRecord struct {
	AnchorID, ChainNodeEntityID, ChangeDirection, ChangeSummary, ImpactSummary string
	Position                                                                   int
	IncomingTransmissionMechanism                                              *string
}

type Counts struct {
	Anchors           int `json:"anchors"`
	EventAssociations int `json:"event_associations"`
	PathNodes         int `json:"path_nodes"`
	Receipts          int `json:"receipts"`
}

type Receipt struct {
	ID, ThemeID, PublisherSubject, PayloadHash string
	AnchorIDsByCenterChainNodeID               map[string]string
	Counts                                     Counts
	PublishedAt, ImportedAt                    time.Time
}
