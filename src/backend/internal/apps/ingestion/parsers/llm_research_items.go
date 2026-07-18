package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type LLMResearchItemsParser struct{}

func (LLMResearchItemsParser) Parse(_ context.Context, source domain.SourceCatalog, response coreingestion.RawResponse) ([]coreingestion.RawDocumentCandidate, error) {
	var topLevel map[string]json.RawMessage
	if err := json.Unmarshal(response.Content, &topLevel); err != nil {
		return nil, fmt.Errorf("items is required")
	}
	if _, ok := topLevel["items"]; !ok {
		return nil, fmt.Errorf("items is required")
	}

	var envelope llmResearchItemsEnvelope
	if err := json.Unmarshal(response.Content, &envelope); err != nil {
		return nil, fmt.Errorf("parse llm research items: %w", err)
	}

	maxResults := intConfigValue(source.SourceConfig["max_results"])
	if maxResults > 0 && len(envelope.Items) > maxResults {
		return nil, fmt.Errorf("items exceeds max_results")
	}

	collectedAt := response.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now()
	}

	docs := make([]coreingestion.RawDocumentCandidate, 0, len(envelope.Items))
	for index, item := range envelope.Items {
		if err := validateLLMResearchItem(item); err != nil {
			return nil, fmt.Errorf("item %d: %w", index, err)
		}

		sourceName := firstNonEmpty(item.SourceName, source.SourceName)
		sourceURL := firstNonEmpty(item.SourceURL, source.SourceURL)
		externalID := firstNonEmpty(item.SourceURL, item.SourceReference, item.CitationText, item.Title)
		publishedAt := parseTime(item.PublishedAt)

		docs = append(docs, coreingestion.RawDocumentCandidate{
			ID:               "raw-" + ingestionID(source.ID, externalID, item.Title),
			SourceID:         source.ID,
			IngestChannel:    source.IngestChannel,
			SourceType:       source.SourceType,
			SourceName:       sourceName,
			SourceURL:        sourceURL,
			SourceExternalID: externalID,
			Title:            strings.TrimSpace(item.Title),
			ContentText:      strings.TrimSpace(item.ContentText),
			RawMIMEType:      response.ContentType,
			Language:         strings.TrimSpace(item.Language),
			PublishedAt:      publishedAt,
			CollectedAt:      collectedAt,
			IngestStatus:     domain.IngestStatusCollected,
		})
	}

	return docs, nil
}

func validateLLMResearchItem(item llmResearchItem) error {
	if err := rejectDisallowedInvestmentFields(item.Raw); err != nil {
		return err
	}
	if strings.TrimSpace(item.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if strings.TrimSpace(item.ContentText) == "" {
		return fmt.Errorf("content_text is required")
	}
	if !validContentOrigin(item.ContentOrigin) {
		return fmt.Errorf("unsupported content_origin %q", item.ContentOrigin)
	}
	if firstNonEmpty(item.SourceURL, item.SourceName, item.SourceReference, item.CitationText, item.ProviderSourceNote) == "" {
		return fmt.Errorf("source attribution is required")
	}
	return nil
}

func rejectDisallowedInvestmentFields(raw map[string]json.RawMessage) error {
	for _, field := range []string{
		"buy_sell",
		"event_score",
		"impact_score",
		"investment_advice",
		"price_prediction",
		"sentiment",
		"transmission_strength",
	} {
		if _, ok := raw[field]; ok {
			return fmt.Errorf("disallowed investment field")
		}
	}
	return nil
}

func validContentOrigin(value string) bool {
	switch strings.TrimSpace(value) {
	case "fetched_source_text", "search_snippet", "llm_generated_summary", "web_content":
		return true
	default:
		return false
	}
}

func intConfigValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	case string:
		var n int
		_, _ = fmt.Sscanf(strings.TrimSpace(v), "%d", &n)
		return n
	default:
		return 0
	}
}

type llmResearchItemsEnvelope struct {
	Items []llmResearchItem `json:"items"`
}

type llmResearchItem struct {
	Raw                   map[string]json.RawMessage
	Title                 string   `json:"title"`
	ContentText           string   `json:"content_text"`
	SourceURL             string   `json:"source_url"`
	SourceName            string   `json:"source_name"`
	SourceReference       string   `json:"source_reference"`
	CitationText          string   `json:"citation_text"`
	ProviderSourceNote    string   `json:"provider_source_note"`
	SourceAttributionType string   `json:"source_attribution_type"`
	PublishedAt           string   `json:"published_at"`
	Language              string   `json:"language"`
	Region                string   `json:"region"`
	ContentOrigin         string   `json:"content_origin"`
	EvidenceExcerpt       string   `json:"evidence_excerpt"`
	RelevanceReason       string   `json:"relevance_reason"`
	TopicTags             []string `json:"topic_tags"`
}

func (i *llmResearchItem) UnmarshalJSON(data []byte) error {
	type alias llmResearchItem
	var parsed alias
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*i = llmResearchItem(parsed)
	i.Raw = raw
	return nil
}
