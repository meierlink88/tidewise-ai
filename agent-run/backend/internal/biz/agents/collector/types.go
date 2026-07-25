package collector

import (
	"context"
	"time"
)

const ContentOrigin = "connector_response"

type ContentLevel string

type CandidateDisposition string

const (
	LevelFullText ContentLevel = "full_text"
	LevelSummary  ContentLevel = "summary"
	LevelSnippet  ContentLevel = "snippet"
	LevelTitle    ContentLevel = "title_only"
)

const (
	DispositionAccepted       CandidateDisposition = "accepted"
	DispositionKnownURL       CandidateDisposition = "known_url"
	DispositionOutOfWindow    CandidateDisposition = "out_of_window"
	DispositionInvalidResult  CandidateDisposition = "invalid_result"
	DispositionExactDuplicate CandidateDisposition = "exact_duplicate"
	DispositionNearDuplicate  CandidateDisposition = "near_duplicate"
)

type Request struct {
	RunID           string
	Prompt          string
	SearchQueries   []string
	CombinedQuery   string
	CandidateLimit  int
	TimeWindowHours int
	CollectedAt     time.Time
}

type Candidate struct {
	Connector        string       `json:"connector"`
	Title            string       `json:"title"`
	URL              string       `json:"url"`
	PublishedAtHint  string       `json:"published_at_hint,omitempty"`
	SourceName       string       `json:"source_name,omitempty"`
	SourceExternalID string       `json:"source_external_id,omitempty"`
	SourceType       string       `json:"source_type,omitempty"`
	Content          string       `json:"content"`
	ContentLevel     ContentLevel `json:"content_level"`
	ContentOrigin    string       `json:"content_origin"`
	Connectors       []string     `json:"connectors,omitempty"`
	PrimaryConnector string       `json:"primary_connector,omitempty"`
	ResultPosition   int          `json:"-"`
}

type ConnectorRun struct {
	Connector    string
	Request      *Request
	Results      []Candidate
	ErrorCode    string
	ErrorSummary string
	Err          error
}

type Stats struct {
	RawResults      int                  `json:"raw_results"`
	MergedResults   int                  `json:"merged_results"`
	ResultsTerminal int                  `json:"results_terminal"`
	ResultsPending  int                  `json:"results_pending"`
	Accepted        int                  `json:"accepted"`
	KnownURL        int                  `json:"known_url"`
	OutOfWindow     int                  `json:"out_of_window"`
	InvalidResult   int                  `json:"invalid_result"`
	ExactDuplicate  int                  `json:"exact_duplicate"`
	NearDuplicate   int                  `json:"near_duplicate"`
	ConnectorCounts map[string]int       `json:"connector_counts"`
	ConnectorErrors map[string]string    `json:"connector_errors,omitempty"`
	ContentLevels   map[ContentLevel]int `json:"content_levels"`
}

type Result struct {
	RunID             string   `json:"run_id"`
	StopReason        string   `json:"stop_reason"`
	Documents         string   `json:"documents_path"`
	AcceptedDocuments []string `json:"accepted_documents"`
	Index             string   `json:"index_path"`
	Candidates        string   `json:"candidates_path"`
	Manifest          string   `json:"manifest_path"`
	Summary           string   `json:"summary_path"`
	Stats             Stats    `json:"stats"`
}

type Connector interface {
	Name() string
	Collect(context.Context, Request) ([]Candidate, error)
}

type Materializer interface {
	Materialize(context.Context, Request, map[string]ConnectorRun) (*Result, error)
}
