package v1

import (
	"context"
	"fmt"
	"net/http"
)

const (
	StatusOK                    = http.StatusOK
	StatusCreated               = http.StatusCreated
	StatusBadRequest            = http.StatusBadRequest
	StatusUnauthorized          = http.StatusUnauthorized
	StatusForbidden             = http.StatusForbidden
	StatusNotFound              = http.StatusNotFound
	StatusConflict              = http.StatusConflict
	StatusRequestEntityTooLarge = http.StatusRequestEntityTooLarge
	StatusUnprocessableEntity   = http.StatusUnprocessableEntity
	StatusTooManyRequests       = http.StatusTooManyRequests
	StatusInternalServerError   = http.StatusInternalServerError
	StatusServiceUnavailable    = http.StatusServiceUnavailable
)

type Response[T any] struct {
	Status int
	Result T
}

type ListResearchThemesRequest struct {
	WindowHours   string
	PublishedFrom string
	PublishedTo   string
	Limit         string
	Cursor        string
}

type GetResearchThemeRequest struct {
	ThemeID     string
	WindowHours string
}

type ReasoningTreeListRequest struct {
	ThemeID  string
	HasQuery bool
}

type ReasoningTreeDetailRequest struct {
	ThemeID         string
	ReasoningTreeID string
	HasQuery        bool
}

type RawDocumentListRequest struct {
	Title        string
	SourceRef    string
	IngestStatus string
	Page         string
	PageSize     string
}

type EventListRequest struct {
	Title         string
	EventStatus   string
	FactStatus    string
	EventTimeFrom string
	EventTimeTo   string
	FirstSeenFrom string
	FirstSeenTo   string
	Page          string
	PageSize      string
}

type EventTagCatalogRequest struct {
	Active bool
}

type PublicError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func NewPublicError(status int, code, message string, details any) *PublicError {
	if details == nil {
		details = map[string]any{}
	}
	return &PublicError{Status: status, Code: code, Message: message, Details: details}
}

func (e *PublicError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type Principal struct {
	Identity string
	Scopes   []string
}

func (p Principal) HasScope(scope string) bool {
	for _, candidate := range p.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

type AdminRawDocument struct {
	ID               string  `json:"id"`
	ContractVersion  int     `json:"contract_version"`
	ArtifactID       string  `json:"artifact_id,omitempty"`
	SourceRef        string  `json:"source_ref,omitempty"`
	IngestChannel    string  `json:"ingest_channel"`
	SourceType       string  `json:"source_type"`
	SourceName       string  `json:"source_name"`
	SourceURL        string  `json:"source_url"`
	SourceExternalID string  `json:"source_external_id,omitempty"`
	Title            string  `json:"title"`
	ContentText      string  `json:"content_text"`
	ContentLevel     string  `json:"content_level"`
	RawObjectURI     string  `json:"raw_object_uri"`
	RawMIMEType      string  `json:"raw_mime_type"`
	Language         string  `json:"language"`
	PublishedAt      *string `json:"published_at"`
	CollectedAt      string  `json:"collected_at"`
	IngestStatus     string  `json:"ingest_status"`
	ContentSHA256    string  `json:"content_sha256"`
}

type AdminEvent struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Summary     string  `json:"summary"`
	EventTime   *string `json:"event_time"`
	FirstSeenAt string  `json:"first_seen_at"`
	KnowableAt  *string `json:"knowable_at"`
	EventStatus string  `json:"event_status"`
	FactStatus  string  `json:"fact_status"`
	DedupeKey   string  `json:"dedupe_key"`
}
