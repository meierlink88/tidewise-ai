package evidencepublication

import "time"

type Disposition string

type SourceLevel string
type LayerType string
type IssueCode string

const (
	DispositionCreated Disposition = "created"
	DispositionReused  Disposition = "reused"

	SourceLevelOfficial SourceLevel = "L1_OFFICIAL"
	SourceLevelWire     SourceLevel = "L2_WIRE"
	SourceLevelMedia    SourceLevel = "L3_MEDIA"
	SourceLevelSocial   SourceLevel = "L4_SOCIAL"

	LayerTypeSingle LayerType = "SINGLE"
	LayerTypeDouble LayerType = "DOUBLE"

	IssueRequired                IssueCode = "REQUIRED"
	IssueTooLong                 IssueCode = "TOO_LONG"
	IssueInvalidEnum             IssueCode = "INVALID_ENUM"
	IssueInvalidURL              IssueCode = "INVALID_URL"
	IssueInvalidOrigin           IssueCode = "INVALID_ORIGIN"
	IssueInvalidTimestamp        IssueCode = "INVALID_TIMESTAMP"
	IssueDuplicate               IssueCode = "DUPLICATE"
	IssueOutOfRange              IssueCode = "OUT_OF_RANGE"
	IssueInvalidLayer            IssueCode = "INVALID_LAYER"
	IssueNonContinuousSplitOrder IssueCode = "NON_CONTINUOUS_SPLIT_ORDER"
	IssueRawEvidenceConflict     IssueCode = "RAW_EVIDENCE_CONFLICT"
	IssueRawEvidenceNotFound     IssueCode = "RAW_EVIDENCE_NOT_FOUND"
	IssueEvidenceIDConflict      IssueCode = "EVIDENCE_ID_CONFLICT"
	IssueEvidenceSetConflict     IssueCode = "EVIDENCE_SET_CONFLICT"
)

type RawEvidence struct {
	RawEvidenceID    string
	SourceID         string
	SourceName       string
	SourceLevel      SourceLevel
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
	LayerType             LayerType
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
	Path    string    `json:"path"`
	Code    IssueCode `json:"code"`
	Message string    `json:"message"`
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
