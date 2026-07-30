package researchanalysiscontext

import (
	"context"
	"errors"
	"time"
)

var ErrHistoricalSemanticsUnavailable = errors.New(
	"strict historical Event semantics are unavailable because a selected Event was superseded after analysis_as_of",
)

var ErrReferenceClosureInconsistent = errors.New(
	"Research Analysis Context reference closure is inconsistent; restart from the first page",
)

type Store interface {
	ListBundles(context.Context, StoreQuery) (StorePage, error)
	ReferenceClosure(context.Context, ReferenceClosureQuery) (Dictionaries, error)
}

type StoreQuery struct {
	DiscoveryWindowStart      time.Time
	DiscoveryWindowEnd        time.Time
	AnalysisAsOf              time.Time
	PredictionHorizonStart    *time.Time
	PredictionHorizonEnd      *time.Time
	PageSize                  int
	AfterKnowledgeAvailableAt *time.Time
	AfterEventID              string
}

type BundleRecord struct {
	KnowledgeAvailableAt time.Time
	EventID              string
	Bundle               EventSemanticBundle
}

type StorePage struct {
	Bundles      []BundleRecord
	Dictionaries Dictionaries
	HasMore      bool
}

type VersionedReference struct {
	Key     string
	Version int
}

type ReferenceClosureQuery struct {
	AnalysisAsOf            time.Time
	EntityIDs               []string
	EntityRelationIDs       []string
	VariableDefinitions     []VersionedReference
	DirectTransmissionRules []VersionedReference
	SemanticSubmissionIDs   []string
}
