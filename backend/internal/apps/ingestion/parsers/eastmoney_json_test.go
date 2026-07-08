package parsers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
)

func TestEastmoneyJSONParserParsesMarketFixture(t *testing.T) {
	source := domain.SourceCatalog{
		ID:            "eastmoney-stock-list",
		IngestChannel: "eastmoney",
		ProviderKey:   "eastmoney",
		ConnectorKey:  "eastmoney",
		ParserKey:     "eastmoney_json",
		SourceType:    "market",
		SourceName:    "东方财富股票列表",
		SourceURL:     "https://example.com/eastmoney.json",
		SourceConfig:  map[string]any{"kind": "eastmoney_stock_list"},
	}
	response := coreingestion.RawResponse{
		ContentType: "application/json",
		Content:     []byte(`{"data":{"diff":[{"f12":"600519","f14":"贵州茅台","f2":1500.5,"f3":1.23}]}}`),
		CollectedAt: time.Now(),
	}

	docs, err := (EastmoneyJSONParser{}).Parse(context.Background(), source, response)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs len = %d, want 1", len(docs))
	}
	if docs[0].Title != "贵州茅台 600519" {
		t.Fatalf("Title = %q, want 贵州茅台 600519", docs[0].Title)
	}
	if docs[0].SourceExternalID != "600519" {
		t.Fatalf("SourceExternalID = %q, want 600519", docs[0].SourceExternalID)
	}
}

func TestEastmoneyJSONParserParsesMarketAndSectorFixtures(t *testing.T) {
	fixtures := []struct {
		name           string
		source         domain.SourceCatalog
		payload        string
		wantTitle      string
		wantExternalID string
		wantContent    string
	}{
		{
			name: "stock-list",
			source: eastmoneyParserFixtureSource("stock:eastmoney:stock-list", "market", "东方财富 A股股票列表", map[string]any{
				"kind": "eastmoney_stock_list",
				"fs":   "m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23",
			}),
			payload:        `{"data":{"diff":[{"f12":"600519","f14":"贵州茅台","f2":1688.88,"f3":1.23}]}}`,
			wantTitle:      "贵州茅台 600519",
			wantExternalID: "600519",
			wantContent:    "1688.88",
		},
		{
			name: "concept-board-list",
			source: eastmoneyParserFixtureSource("stock:eastmoney:concept-board-list", "sector", "东方财富概念板块列表", map[string]any{
				"kind": "eastmoney_concept_board_list",
				"fs":   "m:90 t:3 f:!50",
			}),
			payload:        `{"data":{"diff":[{"f12":"BK0588","f14":"光伏概念","f3":2.34,"f104":178}]}}`,
			wantTitle:      "光伏概念 BK0588",
			wantExternalID: "BK0588",
			wantContent:    "178",
		},
		{
			name: "stock-kline",
			source: eastmoneyParserFixtureSource("stock:eastmoney:stock-kline:600519", "market", "东方财富个股日K - 贵州茅台", map[string]any{
				"kind":   "eastmoney_stock_kline",
				"symbol": "600519",
				"market": "1",
				"klt":    "101",
				"fqt":    "1",
			}),
			payload:        `{"data":{"code":"600519","name":"贵州茅台","klines":["2026-07-08,1680,1690,1700,1670,100000,200000"]}}`,
			wantTitle:      "贵州茅台 600519 K线",
			wantExternalID: "600519:2026-07-08",
			wantContent:    "2026-07-08",
		},
		{
			name: "index-kline",
			source: eastmoneyParserFixtureSource("stock:eastmoney:index-kline:000001", "index", "东方财富指数日K - 上证指数", map[string]any{
				"kind":  "eastmoney_index_kline",
				"secid": "1.000001",
				"klt":   "101",
				"fqt":   "1",
			}),
			payload:        `{"data":{"code":"000001","name":"上证指数","klines":["2026-07-08,3500,3510,3520,3490,300000,400000"]}}`,
			wantTitle:      "上证指数 000001 K线",
			wantExternalID: "000001:2026-07-08",
			wantContent:    "上证指数",
		},
		{
			name: "concept-board-kline",
			source: eastmoneyParserFixtureSource("stock:eastmoney:concept-board-kline:bk0588", "sector", "东方财富概念板块周K - 光伏概念", map[string]any{
				"kind":       "eastmoney_concept_board_kline",
				"board_code": "BK0588",
				"board_name": "光伏概念",
				"secid":      "90.BK0588",
				"klt":        "102",
				"fqt":        "1",
			}),
			payload:        `{"data":{"code":"BK0588","name":"光伏概念","klines":["2026-07-03,900,920,930,890,50000,60000"]}}`,
			wantTitle:      "光伏概念 BK0588 K线",
			wantExternalID: "BK0588:2026-07-03",
			wantContent:    "光伏概念",
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			docs, err := EastmoneyJSONParser{}.Parse(context.Background(), fixture.source, coreingestion.RawResponse{
				ContentType: "application/json",
				Content:     []byte(fixture.payload),
			})
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(docs) != 1 {
				t.Fatalf("docs len = %d, want 1", len(docs))
			}
			doc := docs[0]
			if doc.Title != fixture.wantTitle {
				t.Fatalf("Title = %q, want %q", doc.Title, fixture.wantTitle)
			}
			if doc.SourceExternalID != fixture.wantExternalID {
				t.Fatalf("SourceExternalID = %q, want %q", doc.SourceExternalID, fixture.wantExternalID)
			}
			if !strings.Contains(doc.ContentText, fixture.wantContent) {
				t.Fatalf("ContentText = %q, want containing %q", doc.ContentText, fixture.wantContent)
			}
		})
	}
}

func TestEastmoneyJSONParserHandlesEmptyResponse(t *testing.T) {
	source := eastmoneyParserFixtureSource("stock:eastmoney:stock-list", "market", "东方财富 A股股票列表", map[string]any{
		"kind": "eastmoney_stock_list",
	})

	docs, err := EastmoneyJSONParser{}.Parse(context.Background(), source, coreingestion.RawResponse{
		ContentType: "application/json",
		Content:     []byte(`{"data":{"diff":[]}}`),
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("docs len = %d, want 0", len(docs))
	}
}

func TestEastmoneyJSONParserReturnsParseError(t *testing.T) {
	source := eastmoneyParserFixtureSource("stock:eastmoney:stock-list", "market", "东方财富 A股股票列表", map[string]any{
		"kind": "eastmoney_stock_list",
	})

	_, err := EastmoneyJSONParser{}.Parse(context.Background(), source, coreingestion.RawResponse{
		ContentType: "application/json",
		Content:     []byte(`{"data":`),
	})
	if err == nil || !strings.Contains(err.Error(), "parse eastmoney json") {
		t.Fatalf("Parse() error = %v, want parse eastmoney json", err)
	}
}

func eastmoneyParserFixtureSource(id string, sourceType string, sourceName string, config map[string]any) domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:            id,
		IngestChannel: "eastmoney",
		ProviderKey:   "eastmoney",
		ConnectorKey:  "eastmoney",
		ParserKey:     "eastmoney_json",
		SourceType:    sourceType,
		SourceName:    sourceName,
		SourceURL:     "https://push2.eastmoney.com/api/qt/clist/get",
		SourceConfig:  config,
	}
}
