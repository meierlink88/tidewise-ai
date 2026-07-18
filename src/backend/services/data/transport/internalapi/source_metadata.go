package internalapi

import (
	"errors"
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/sourcemetadata"
)

func (d Dependencies) listSourceMetadata(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	limit, ok := optionalInt(response, requestID, request.URL.Query().Get("limit"), 20, 1, 100, "limit")
	if !ok {
		return
	}
	status := domain.SourceCatalogStatus(request.URL.Query().Get("status"))
	if status != "" && !oneOf(string(status), "active", "inactive", "disabled") {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "unsupported source status")
		return
	}
	if d.SourceMetadata == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "source metadata store is unavailable")
		return
	}
	page, err := d.SourceMetadata.List(request.Context(), sourcemetadata.ListRequest{
		Status: status,
		Limit:  limit,
		Cursor: request.URL.Query().Get("cursor"),
	})
	if err != nil {
		if errors.Is(err, sourcemetadata.ErrInvalidCursor) {
			writeError(response, requestID, http.StatusBadRequest, "INVALID_CURSOR", "source metadata cursor is invalid")
			return
		}
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "source metadata aggregate failed")
		return
	}
	items := make([]sourceMetadata, 0, len(page.Items))
	for _, source := range page.Items {
		items = append(items, sourceMetadataDTO(source))
	}
	writeEnvelope(response, http.StatusOK, requestID, map[string]any{"items": items, "next_cursor": page.NextCursor})
}

type sourceMetadata struct {
	ID             string         `json:"id"`
	IngestChannel  string         `json:"ingest_channel"`
	ProviderKey    string         `json:"provider_key"`
	ConnectorKey   string         `json:"connector_key"`
	ParserKey      string         `json:"parser_key"`
	SourceType     string         `json:"source_type"`
	SourceName     string         `json:"source_name"`
	SourceURL      string         `json:"source_url"`
	AuthType       string         `json:"auth_type"`
	CredentialRef  *string        `json:"credential_ref"`
	ApprovedConfig map[string]any `json:"approved_config"`
	RateLimitHint  map[string]int `json:"rate_limit_hint"`
	UsagePolicy    string         `json:"usage_policy"`
	Status         string         `json:"status"`
}

func sourceMetadataDTO(source domain.SourceCatalog) sourceMetadata {
	approved := map[string]any{}
	for _, key := range []string{"collection_mode", "route_template", "prompt_ref", "prompt_version", "language", "result_limit", "timeout_seconds"} {
		if value, ok := source.SourceConfig[key]; ok {
			approved[key] = value
		}
	}
	if source.RouteTemplate != "" {
		approved["route_template"] = source.RouteTemplate
	}
	requests := positiveInt(source.RateLimitPolicy["requests"])
	window := positiveInt(source.RateLimitPolicy["window_seconds"])
	if requests == 0 {
		requests = positiveInt(source.RateLimitPolicy["requests_per_minute"])
		if requests > 0 {
			window = 60
		}
	}
	if requests == 0 {
		requests = 1
	}
	if window == 0 {
		window = 60
	}
	var credentialRef *string
	if source.CredentialRef != "" {
		value := source.CredentialRef
		credentialRef = &value
	}
	return sourceMetadata{ID: source.ID, IngestChannel: source.IngestChannel, ProviderKey: source.ProviderKey, ConnectorKey: source.ConnectorKey, ParserKey: source.ParserKey, SourceType: source.SourceType, SourceName: source.SourceName, SourceURL: source.SourceURL, AuthType: source.AuthType, CredentialRef: credentialRef, ApprovedConfig: approved, RateLimitHint: map[string]int{"requests": requests, "window_seconds": window}, UsagePolicy: source.UsagePolicy, Status: string(source.Status)}
}

func positiveInt(value any) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	}
	return 0
}
