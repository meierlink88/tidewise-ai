package parsers

import (
	"context"
	"strings"
	"testing"
	"time"

	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestLLMResearchItemsParserParsesValidItems(t *testing.T) {
	source := llmResearchParserFixtureSource(map[string]any{"max_results": 5})
	response := coreingestion.RawResponse{
		ContentType: "application/json",
		Content: []byte(`{
			"items": [
				{
					"title": "美联储释放政策信号",
					"content_text": "美联储官员表示将关注通胀和就业数据。",
					"source_name": "Reuters",
					"source_url": "https://www.reuters.com/markets/fed-policy",
					"published_at": "2026-07-09T08:00:00Z",
					"language": "zh-CN",
					"topic_tags": ["central_bank"],
					"evidence_excerpt": "关注通胀和就业数据",
					"relevance_reason": "货币政策影响全球风险资产",
					"content_origin": "search_snippet"
				}
			],
			"meta": {"actual_results": 1}
		}`),
	}

	docs, err := LLMResearchItemsParser{}.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs len = %d, want 1", len(docs))
	}
	doc := docs[0]
	if doc.Title != "美联储释放政策信号" {
		t.Fatalf("Title = %q, want event title", doc.Title)
	}
	if doc.SourceURL != "https://www.reuters.com/markets/fed-policy" {
		t.Fatalf("SourceURL = %q, want item url", doc.SourceURL)
	}
	if doc.SourceName != "Reuters" {
		t.Fatalf("SourceName = %q, want item source", doc.SourceName)
	}
	if doc.Language != "zh-CN" {
		t.Fatalf("Language = %q, want zh-CN", doc.Language)
	}
	if doc.PublishedAt == nil || !doc.PublishedAt.Equal(time.Date(2026, 7, 9, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("PublishedAt = %v, want parsed time", doc.PublishedAt)
	}
	if !strings.Contains(doc.ContentText, "美联储官员") {
		t.Fatalf("ContentText = %q, want content", doc.ContentText)
	}
}

func TestLLMResearchItemsParserAcceptsNamedSourceAttribution(t *testing.T) {
	source := llmResearchParserFixtureSource(map[string]any{"max_results": 5})
	response := coreingestion.RawResponse{
		ContentType: "application/json",
		Content: []byte(`{
			"items": [
				{
					"title": "来源名称归因材料",
					"content_text": "没有 URL 但有来源名称和引用文本。",
					"source_name": "新华社",
					"citation_text": "新华社报道称...",
					"source_attribution_type": "named_source",
					"content_origin": "llm_generated_summary"
				}
			]
		}`),
	}

	docs, err := LLMResearchItemsParser{}.Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs len = %d, want 1", len(docs))
	}
	if docs[0].SourceName != "新华社" {
		t.Fatalf("SourceName = %q, want 新华社", docs[0].SourceName)
	}
	if docs[0].SourceURL != source.SourceURL {
		t.Fatalf("SourceURL = %q, want fallback source url", docs[0].SourceURL)
	}
}

func TestLLMResearchItemsParserRejectsInvalidOutput(t *testing.T) {
	cases := []struct {
		name      string
		payload   string
		wantError string
	}{
		{name: "bare array", payload: `[{"title":"bad"}]`, wantError: "items is required"},
		{name: "missing items", payload: `{"meta":{}}`, wantError: "items is required"},
		{name: "unknown content origin", payload: `{"items":[{"title":"A","content_text":"B","source_url":"https://example.com","content_origin":"unknown"}]}`, wantError: "unsupported content_origin"},
		{name: "missing attribution", payload: `{"items":[{"title":"A","content_text":"B","content_origin":"search_snippet"}]}`, wantError: "source attribution is required"},
		{name: "investment boundary", payload: `{"items":[{"title":"A","content_text":"B","source_url":"https://example.com","content_origin":"search_snippet","price_prediction":"上涨"}]}`, wantError: "disallowed investment field"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LLMResearchItemsParser{}.Parse(context.Background(), llmResearchParserFixtureSource(map[string]any{"max_results": 5}), coreingestion.RawResponse{
				ContentType: "application/json",
				Content:     []byte(tc.payload),
			})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("Parse() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

func TestLLMResearchItemsParserRejectsTooManyItems(t *testing.T) {
	_, err := LLMResearchItemsParser{}.Parse(context.Background(), llmResearchParserFixtureSource(map[string]any{"max_results": 1}), coreingestion.RawResponse{
		ContentType: "application/json",
		Content: []byte(`{
			"items": [
				{"title":"A","content_text":"B","source_url":"https://example.com/a","content_origin":"search_snippet"},
				{"title":"C","content_text":"D","source_url":"https://example.com/c","content_origin":"search_snippet"}
			]
		}`),
	})
	if err == nil || !strings.Contains(err.Error(), "items exceeds max_results") {
		t.Fatalf("Parse() error = %v, want max results error", err)
	}
}

func llmResearchParserFixtureSource(config map[string]any) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:            "tidewise:ai-web-research:fixture",
		IngestChannel: "ai_web_research",
		ProviderKey:   "llm_web_research",
		ConnectorKey:  "llm_web_research",
		ParserKey:     "llm_research_items",
		SourceType:    "news",
		SourceName:    "AI Web Research",
		SourceURL:     "tidewise://ai-web-research/fixture",
		SourceConfig:  config,
	}
}
