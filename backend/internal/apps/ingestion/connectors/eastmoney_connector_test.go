package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	coreingestion "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/core"
)

func TestEastmoneyConnectorReturnsLimitStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
	}))
	defer server.Close()

	source := eastmoneyConnectorFixtureSource("stock:eastmoney:stock-list", "market", "东方财富 A股股票列表", map[string]any{
		"kind": "eastmoney_stock_list",
	})
	source.SourceURL = server.URL

	_, err := EastmoneyConnector{Client: server.Client()}.Fetch(context.Background(), source, coreingestion.Credential{})
	if err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("Fetch() error = %v, want status 429", err)
	}
}

func TestEastmoneyConnectorBuildsQueryFromSourceConfig(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.RawQuery
		_, _ = w.Write([]byte(`{"data":{"diff":[]}}`))
	}))
	defer server.Close()

	listSource := eastmoneyConnectorFixtureSource("stock:eastmoney:stock-list", "market", "东方财富 A股股票列表", map[string]any{
		"kind": "eastmoney_stock_list",
		"fs":   "m:0+t:6",
	})
	listSource.SourceURL = server.URL
	if _, err := (EastmoneyConnector{Client: server.Client()}).Fetch(context.Background(), listSource, coreingestion.Credential{}); err != nil {
		t.Fatalf("Fetch(stock list) error = %v", err)
	}
	listQuery := waitForEastmoneyQuery(t, requests)
	if !strings.Contains(listQuery, "fs=m%3A0%2Bt%3A6") || !strings.Contains(listQuery, "fields=") {
		t.Fatalf("stock list query = %q, want fs and fields", listQuery)
	}

	klineSource := eastmoneyConnectorFixtureSource("stock:eastmoney:stock-kline:600519", "market", "东方财富个股日K - 贵州茅台", map[string]any{
		"kind":   "eastmoney_stock_kline",
		"symbol": "600519",
		"market": "1",
		"klt":    "101",
		"fqt":    "1",
	})
	klineSource.SourceURL = server.URL
	if _, err := (EastmoneyConnector{Client: server.Client()}).Fetch(context.Background(), klineSource, coreingestion.Credential{}); err != nil {
		t.Fatalf("Fetch(kline) error = %v", err)
	}
	klineQuery := waitForEastmoneyQuery(t, requests)
	if !strings.Contains(klineQuery, "secid=1.600519") || !strings.Contains(klineQuery, "klt=101") || !strings.Contains(klineQuery, "fqt=1") {
		t.Fatalf("kline query = %q, want secid/klt/fqt", klineQuery)
	}
}

func TestSDKStubConnectorReturnsBoundaryError(t *testing.T) {
	_, err := SDKStubConnector{Name: "sdk_tushare"}.Fetch(context.Background(), domain.SourceCatalog{}, coreingestion.Credential{})
	if err == nil || !strings.Contains(err.Error(), "worker") {
		t.Fatalf("Fetch() error = %v, want worker boundary error", err)
	}
}

func waitForEastmoneyQuery(t *testing.T, requests <-chan string) string {
	t.Helper()
	select {
	case query := <-requests:
		return query
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for eastmoney request")
		return ""
	}
}

func eastmoneyConnectorFixtureSource(id string, sourceType string, sourceName string, config map[string]any) domain.SourceCatalog {
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
