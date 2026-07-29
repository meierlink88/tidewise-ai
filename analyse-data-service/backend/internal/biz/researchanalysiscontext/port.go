package researchanalysiscontext

import (
	"context"
	"errors"
	"time"
)

var ErrHistoricalSemanticsUnavailable = errors.New(
	"strict historical Event semantics are unavailable because a selected Event was superseded after analysis_as_of",
)

type Store interface {
	List(context.Context, StoreQuery) (StorePage, error)
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
	Bundles               []BundleRecord
	Dictionaries          Dictionaries
	DictionaryFingerprint string
	HasMore               bool
}
