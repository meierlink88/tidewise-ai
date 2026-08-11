package evidencepublication

import "time"

type Disposition string

const (
	DispositionCreated Disposition = "created"
	DispositionReused  Disposition = "reused"
)

type RawEvidence struct {
	RawEvidenceID    string
	SourceID         string
	SourceName       string
	SourceLevel      string
	SourceURL        string
	IsOriginal       bool
	QuotedSourceID   *string
	QuotedSourceName *string
	Title            *string
	RawText          string
	PublishedAt      *time.Time
	CollectedAt      time.Time
	Keywords         []string
}

type StoredRawEvidence struct {
	RawEvidence
	ContentHash string
}

type Evidence struct {
	EvidenceID            string
	SplitOrder            int
	LayerType             string
	SourceWho             *string
	SourceWhat            string
	SourceWhen            *time.Time
	SourceWhenRaw         *string
	SourceWhere           *string
	SourceWhy             *string
	SourceHow             *string
	SourceWhoCore         *string
	SourceWhatCore        *string
	SourceWhenCore        *time.Time
	SourceWhenRawCore     *string
	SourceWhereCore       *string
	SourceWhyCore         *string
	SourceHowCore         *string
	ExpressionFingerprint string
	ExpressionKey         string
	FingerprintVersion    string
}

type StoredEvidence struct {
	Evidence
	RawEvidenceID string
	IsSplit       bool
}

type RawEvidenceResult struct {
	ReceiptID   string
	ImportedAt  time.Time
	RawEvidence RawEvidenceItemResult
}

type RawEvidenceItemResult struct {
	RawEvidenceID string
	ContentHash   string
	Keywords      []string
	Disposition   Disposition
}

type EvidenceResult struct {
	ReceiptID     string
	RawEvidenceID string
	ImportedAt    time.Time
	Evidences     []EvidenceItemResult
	Counts        EvidenceCounts
}

type EvidenceItemResult struct {
	EvidenceID  string
	SplitOrder  int
	IsSplit     bool
	Disposition Disposition
}

type EvidenceCounts struct {
	Created int
	Reused  int
}

type RawEvidencePublicationReceipt struct {
	ID            string
	CallerSubject string
	RawEvidenceID string
	Disposition   Disposition
	ImportedAt    time.Time
}

type EvidencePublicationReceipt struct {
	ID            string
	CallerSubject string
	RawEvidenceID string
	EvidenceIDs   []string
	Counts        EvidenceCounts
	ImportedAt    time.Time
}

type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct{ Issues []Issue }

func (e *ValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Evidence Publication failed validation"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

type ConflictError struct{ Issues []Issue }

func (e *ConflictError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Evidence Publication conflicts with stored data"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

type ReferenceError struct{ Issues []Issue }

func (e *ReferenceError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Evidence Publication references unavailable data"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}
