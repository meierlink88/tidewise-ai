package evidence

import (
	"context"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationPublishRawEvidence = "data.v1.publishRawEvidence"
	OperationPublishEvidence    = "data.v1.publishEvidence"

	ErrorInvalidRequest                      = "INVALID_REQUEST"
	ErrorEvidencePublicationInvalid          = "EVIDENCE_PUBLICATION_INVALID"
	ErrorEvidencePublicationReferenceInvalid = "EVIDENCE_PUBLICATION_REFERENCE_INVALID"
	ErrorEvidencePublicationConflict         = "EVIDENCE_PUBLICATION_CONFLICT"
	ErrorEvidencePublicationTimeout          = "EVIDENCE_PUBLICATION_TIMEOUT"
	ErrorEvidencePublicationFailed           = "EVIDENCE_PUBLICATION_FAILED"
)

func BusinessOperations() []string {
	return []string{OperationPublishRawEvidence, OperationPublishEvidence}
}

type Service interface {
	PublishRawEvidence(context.Context, *RawEvidencePublicationRequest) (*v1.Response[RawEvidencePublicationResult], error)
	PublishEvidence(context.Context, *EvidencePublicationRequest) (*v1.Response[EvidencePublicationResult], error)
}

type RawEvidencePublicationRequest struct {
	RawEvidence RawEvidence `json:"raw_evidence"`
}

type RawEvidence struct {
	RawEvidenceID    string     `json:"raw_evidence_id"`
	SourceID         string     `json:"source_id"`
	SourceName       string     `json:"source_name"`
	SourceLevel      string     `json:"source_level"`
	SourceURL        string     `json:"source_url"`
	IsOriginal       bool       `json:"is_original"`
	QuotedSourceID   *string    `json:"quoted_source_id"`
	QuotedSourceName *string    `json:"quoted_source_name"`
	Title            *string    `json:"title"`
	RawText          string     `json:"raw_text"`
	PublishedAt      *time.Time `json:"published_at"`
	CollectedAt      time.Time  `json:"collected_at"`
	Keywords         []string   `json:"keywords"`
}

type EvidencePublicationRequest struct {
	RawEvidenceID string           `json:"raw_evidence_id"`
	Evidences     []AtomicEvidence `json:"evidences"`
}

type AtomicEvidence struct {
	EvidenceID            string     `json:"evidence_id"`
	SplitOrder            int        `json:"split_order"`
	LayerType             string     `json:"layer_type"`
	SourceWho             *string    `json:"source_who"`
	SourceWhat            string     `json:"source_what"`
	SourceWhen            *time.Time `json:"source_when"`
	SourceWhenRaw         *string    `json:"source_when_raw"`
	SourceWhere           *string    `json:"source_where"`
	SourceWhy             *string    `json:"source_why"`
	SourceHow             *string    `json:"source_how"`
	SourceWhoCore         *string    `json:"source_who_core"`
	SourceWhatCore        *string    `json:"source_what_core"`
	SourceWhenCore        *time.Time `json:"source_when_core"`
	SourceWhenRawCore     *string    `json:"source_when_raw_core"`
	SourceWhereCore       *string    `json:"source_where_core"`
	SourceWhyCore         *string    `json:"source_why_core"`
	SourceHowCore         *string    `json:"source_how_core"`
	ExpressionFingerprint string     `json:"expression_fingerprint"`
	ExpressionKey         string     `json:"expression_key"`
	FingerprintVersion    string     `json:"fingerprint_version"`
}

type RawEvidencePublicationResult struct {
	RawEvidenceID string `json:"raw_evidence_id"`
}

type EvidencePublicationResult struct {
	RawEvidenceID string   `json:"raw_evidence_id"`
	EvidenceIDs   []string `json:"evidence_ids"`
}
