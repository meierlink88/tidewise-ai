package connectors

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/collector"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

type CLSTelegraph struct {
	Endpoint string
	Client   HTTPClient
}

func (c CLSTelegraph) Name() string { return "cls_telegraph" }

func (c CLSTelegraph) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	params := map[string]string{
		"appName": "CailianpressWeb", "os": "web", "sv": "7.7.5",
		"last_time": "", "refresh_type": "1", "rn": strconv.Itoa(request.CandidateLimit),
	}
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}
	query.Set("sign", signCLS(params))
	var payload struct {
		Data struct {
			Rows []struct {
				ID      any    `json:"id"`
				CTime   any    `json:"ctime"`
				Title   string `json:"title"`
				Brief   string `json:"brief"`
				Content string `json:"content"`
			} `json:"roll_data"`
		} `json:"data"`
	}
	if err := getJSON(ctx, connectorClient(c.Client), c.Endpoint, query, map[string]string{"Referer": "https://www.cls.cn/"}, &payload); err != nil {
		return nil, err
	}
	results := make([]collector.Candidate, 0, request.CandidateLimit)
	for _, row := range payload.Data.Rows {
		id := plainText(row.ID)
		published := timestampISO(row.CTime)
		itemURL := ""
		if id != "" {
			itemURL = "https://www.cls.cn/detail/" + id
		}
		content := plainText(row.Content)
		level := collector.LevelFullText
		if content == "" {
			content = plainText(row.Brief)
			level = levelFor(content, collector.LevelSnippet)
		}
		results = appendCandidate(results, collector.Candidate{
			Title: firstNonEmpty(plainText(row.Title), plainText(row.Brief)),
			URL:   itemURL, PublishedAtHint: published,
			SourceName: "财联社", SourceExternalID: id, SourceType: "fast_news",
			Content: content, ContentLevel: level,
		})
		if len(results) >= request.CandidateLimit {
			break
		}
	}
	return results, nil
}

type EastmoneyFastNews struct {
	Endpoint string
	Client   HTTPClient
}

func (c EastmoneyFastNews) Name() string { return "eastmoney_fastnews" }

func (c EastmoneyFastNews) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	query := url.Values{
		"client": {"web"}, "biz": {"web_724"}, "fastColumn": {"102"},
		"sortEnd": {""}, "pageSize": {strconv.Itoa(request.CandidateLimit)}, "req_trace": {uuid.NewString()},
	}
	var payload struct {
		Data struct {
			Rows []struct {
				Code, Title, Summary, ShowTime string
			} `json:"fastNewsList"`
		} `json:"data"`
	}
	if err := getJSON(ctx, connectorClient(c.Client), c.Endpoint, query, map[string]string{"Referer": "https://kuaixun.eastmoney.com/"}, &payload); err != nil {
		return nil, err
	}
	results := make([]collector.Candidate, 0, request.CandidateLimit)
	for _, row := range payload.Data.Rows {
		published := chinaDateTimeISO(row.ShowTime)
		itemURL := ""
		if row.Code != "" {
			itemURL = "https://finance.eastmoney.com/a/" + row.Code + ".html"
		}
		content := plainText(row.Summary)
		results = appendCandidate(results, collector.Candidate{
			Title: plainText(row.Title), URL: itemURL,
			PublishedAtHint: published, SourceName: "东方财富", SourceExternalID: row.Code,
			SourceType: "fast_news", Content: content, ContentLevel: levelFor(content, collector.LevelSummary),
		})
		if len(results) >= request.CandidateLimit {
			break
		}
	}
	return results, nil
}

type EastmoneyStockNews struct {
	Endpoint string
	Client   HTTPClient
}

func (c EastmoneyStockNews) Name() string { return "eastmoney_stock_news" }

func (c EastmoneyStockNews) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	searchParam, _ := json.Marshal(map[string]any{
		"uid": "", "keyword": request.CombinedQuery, "type": []string{"cmsArticleWebOld"},
		"client": "web", "clientType": "web",
		"param": map[string]any{"cmsArticleWebOld": map[string]any{
			"searchScope": "default", "sort": "time", "pageIndex": 1, "pageSize": request.CandidateLimit,
		}},
	})
	query := url.Values{"cb": {"callback"}, "param": {string(searchParam)}, "_": {"0"}}
	var payload struct {
		Result struct {
			Rows []struct {
				Code, ArticleID, Title, URL, Date, Content, MediaName string
			} `json:"cmsArticleWebOld"`
		} `json:"result"`
	}
	if err := getJSON(ctx, connectorClient(c.Client), c.Endpoint, query, map[string]string{"Referer": "https://so.eastmoney.com/"}, &payload); err != nil {
		return nil, err
	}
	results := make([]collector.Candidate, 0, request.CandidateLimit)
	for _, row := range payload.Result.Rows {
		published := chinaDateTimeISO(row.Date)
		content := plainText(row.Content)
		if media := plainText(row.MediaName); media != "" && content != "" {
			content = "来源：" + media + "；" + content
		}
		externalID := firstNonEmpty(row.Code, row.ArticleID)
		results = appendCandidate(results, collector.Candidate{
			Title: plainText(row.Title), URL: strings.TrimSpace(row.URL), PublishedAtHint: published,
			SourceName: plainText(row.MediaName), SourceExternalID: externalID, SourceType: "news",
			Content: content, ContentLevel: levelFor(content, collector.LevelSnippet),
		})
		if len(results) >= request.CandidateLimit {
			break
		}
	}
	return results, nil
}

type STCNQuickNews struct {
	Endpoint string
	Client   HTTPClient
}

func (c STCNQuickNews) Name() string { return "stcn_quicknews" }

func (c STCNQuickNews) Collect(ctx context.Context, request collector.Request) ([]collector.Candidate, error) {
	var payload struct {
		State any `json:"state"`
		Rows  []struct {
			ID, URL, Title, Source, Content string
			Time                            any `json:"time"`
		} `json:"data"`
	}
	if err := getJSON(ctx, connectorClient(c.Client), c.Endpoint, url.Values{"type": {"kx"}}, map[string]string{
		"Referer": "https://www.stcn.com/article/list/kx.html", "X-Requested-With": "XMLHttpRequest",
	}, &payload); err != nil {
		return nil, err
	}
	results := make([]collector.Candidate, 0, request.CandidateLimit)
	for _, row := range payload.Rows {
		published := timestampISO(row.Time)
		content := plainText(row.Content)
		if source := plainText(row.Source); source != "" && content != "" {
			content = "来源：" + source + "；" + content
		}
		itemURL := ""
		if strings.TrimSpace(row.URL) != "" {
			itemURL, _ = url.JoinPath("https://www.stcn.com", row.URL)
		}
		results = appendCandidate(results, collector.Candidate{
			Title: plainText(row.Title), URL: itemURL, PublishedAtHint: published,
			SourceName: firstNonEmpty(plainText(row.Source), "证券时报"), SourceExternalID: row.ID,
			SourceType: "fast_news", Content: content, ContentLevel: levelFor(content, collector.LevelFullText),
		})
		if len(results) >= request.CandidateLimit {
			break
		}
	}
	return results, nil
}

func signCLS(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	first := sha1.Sum([]byte(strings.Join(parts, "&")))
	second := md5.Sum([]byte(hex.EncodeToString(first[:])))
	return hex.EncodeToString(second[:])
}

func connectorClient(client HTTPClient) HTTPClient {
	if client != nil {
		return client
	}
	return defaultClient()
}

func plainText(value any) string {
	if value == nil {
		return ""
	}
	text := html.UnescapeString(fmt.Sprint(value))
	return strings.Join(strings.Fields(htmlTagPattern.ReplaceAllString(text, " ")), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func timestampISO(value any) string {
	text := strings.TrimSpace(fmt.Sprint(value))
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return ""
	}
	if number > 10_000_000_000 {
		number /= 1000
	}
	return time.Unix(int64(number), 0).UTC().Format(time.RFC3339)
}

func chinaDateTimeISO(value string) string {
	location := time.FixedZone("CST", 8*60*60)
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		parsed, err := time.ParseInLocation(layout, strings.TrimSpace(value), location)
		if err == nil {
			return parsed.UTC().Format(time.RFC3339)
		}
	}
	return ""
}
