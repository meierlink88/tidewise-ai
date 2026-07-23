package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

func TestProfessionalConnectorsPreserveObservedNumericIdentityAndHighlightedText(t *testing.T) {
	t.Run("CLS numeric ID", func(t *testing.T) {
		payload, err := os.ReadFile("testdata/cls_numeric_id.json")
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write(payload)
		}))
		defer server.Close()

		results, err := (CLSTelegraph{Endpoint: server.URL, Client: server.Client()}).Collect(context.Background(), collector.Request{CandidateLimit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 {
			t.Fatalf("results = %#v", results)
		}
		if results[0].SourceExternalID != "2434241" || results[0].URL != "https://www.cls.cn/detail/2434241" {
			t.Fatalf("numeric identity was not preserved: %#v", results[0])
		}
	})

	t.Run("Eastmoney highlight tags", func(t *testing.T) {
		payload, err := os.ReadFile("testdata/eastmoney_highlight.jsonp")
		if err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write(payload)
		}))
		defer server.Close()

		results, err := (EastmoneyStockNews{Endpoint: server.URL, Client: server.Client()}).Collect(context.Background(), collector.Request{
			CombinedQuery: "产业链", CandidateLimit: 10,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 {
			t.Fatalf("results = %#v", results)
		}
		if results[0].Title != "融信中国：6月合同销售金额1.80亿元" {
			t.Fatalf("title = %q", results[0].Title)
		}
		if results[0].Content != "公司上半年实现销售金额1.80亿元，拟回购股份。" {
			t.Fatalf("content = %q", results[0].Content)
		}
	})
}

func TestProfessionalFinanceConnectorsPreserveDirectResults(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	request := collector.Request{
		CombinedQuery: "商业航天", CandidateLimit: 10,
		TimeWindowHours: 48, CollectedAt: now,
	}

	t.Run("CLS Telegraph", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, incoming *http.Request) {
			if incoming.URL.Query().Get("sign") == "" || incoming.URL.Query().Get("rn") != "10" {
				t.Fatalf("CLS query = %s", incoming.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"data": map[string]any{"roll_data": []map[string]any{
				{"id": 101, "ctime": 1784199600, "title": "政策快讯", "content": "候选内容"},
				{"ctime": 1784199600, "title": "缺少 ID", "content": "不得成为结果"},
				{"id": 99, "ctime": 1783681200, "title": "窗口外快讯", "content": "旧内容"},
			}}})
		}))
		defer server.Close()

		results, err := (CLSTelegraph{Endpoint: server.URL, Client: server.Client()}).Collect(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 3 || results[0].URL != "https://www.cls.cn/detail/101" || results[0].Content != "候选内容" || results[0].ContentLevel != collector.LevelFullText {
			t.Fatalf("CLS results = %#v", results)
		}
		if results[1].URL != "" || results[2].PublishedAtHint != "2026-07-10T11:00:00Z" {
			t.Fatalf("CLS did not preserve invalid and out-of-window direct results: %#v", results)
		}
	})

	t.Run("Eastmoney Fast News", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, incoming *http.Request) {
			if incoming.URL.Query().Get("pageSize") != "10" {
				t.Fatalf("Eastmoney fast-news query = %s", incoming.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"data": map[string]any{"fastNewsList": []map[string]any{
				{"code": "202607160001", "title": "产业快讯", "summary": "产业快讯正文片段", "showTime": "2026-07-16 18:30:00"},
				{"code": "202607100001", "title": "旧快讯", "summary": "旧内容", "showTime": "2026-07-10 18:30:00"},
			}}})
		}))
		defer server.Close()

		results, err := (EastmoneyFastNews{Endpoint: server.URL, Client: server.Client()}).Collect(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 || results[0].URL != "https://finance.eastmoney.com/a/202607160001.html" || results[0].PublishedAtHint != "2026-07-16T10:30:00Z" || results[0].ContentLevel != collector.LevelSummary || results[1].PublishedAtHint != "2026-07-10T10:30:00Z" {
			t.Fatalf("Eastmoney fast-news results = %#v", results)
		}
	})

	t.Run("Eastmoney Stock News", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, incoming *http.Request) {
			if !strings.Contains(incoming.URL.Query().Get("param"), `"keyword":"商业航天"`) {
				t.Fatalf("Eastmoney stock-news query = %s", incoming.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`callback({"result":{"cmsArticleWebOld":[{"code":"EM001","title":"商业航天公司新闻","url":"https://example.com/em001","date":"2026-07-16 17:00:00","content":"候选新闻片段","mediaName":"证券时报"}]}})`))
		}))
		defer server.Close()

		results, err := (EastmoneyStockNews{Endpoint: server.URL, Client: server.Client()}).Collect(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].URL != "https://example.com/em001" || results[0].PublishedAtHint != "2026-07-16T09:00:00Z" || results[0].Content != "候选新闻片段" || results[0].SourceName != "证券时报" || results[0].ContentLevel != collector.LevelSnippet {
			t.Fatalf("Eastmoney stock-news results = %#v", results)
		}
	})

	t.Run("STCN Quick News", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, incoming *http.Request) {
			if incoming.URL.Query().Get("type") != "kx" {
				t.Fatalf("STCN query = %s", incoming.URL.RawQuery)
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"state": 1, "data": []map[string]any{
				{"id": "4022599", "url": "/article/detail/4022599.html", "title": "算力基础设施快讯", "source": "人民财讯", "time": int64(1784197800000), "content": "候选快讯内容"},
				{"id": "3900000", "url": "/article/detail/3900000.html", "title": "旧快讯", "source": "证券时报", "time": int64(1783679400000), "content": "旧内容"},
			}})
		}))
		defer server.Close()

		results, err := (STCNQuickNews{Endpoint: server.URL, Client: server.Client()}).Collect(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 2 || results[0].URL != "https://www.stcn.com/article/detail/4022599.html" || results[0].PublishedAtHint != "2026-07-16T10:30:00Z" || results[0].Content != "候选快讯内容" || results[0].SourceName != "人民财讯" || results[0].ContentLevel != collector.LevelFullText || results[1].PublishedAtHint != "2026-07-10T10:30:00Z" {
			t.Fatalf("STCN results = %#v", results)
		}
	})
}
