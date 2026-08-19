package source

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

const maxImportBytes = 2_000_000

type FixedManifestOptions struct {
	Endpoints map[string]string
	AppKeys   map[string]string
}

func CurrentFixedManifest(options FixedManifestOptions) []sourcebiz.Source {
	type definition struct {
		code, name, endpoint string
		channel              sourcebiz.ChannelType
		adapter              sourcebiz.AdapterKey
		enabled              bool
		level                sourcebiz.SourceLevel
		credential           bool
	}
	definitions := []definition{
		{"bocha", "博查", "https://api.bochaai.com/v1/web-search", sourcebiz.ChannelWebSearch, sourcebiz.AdapterBocha, true, sourcebiz.SourceLevelMedia, true},
		{"tavily", "Tavily", "https://api.tavily.com/search", sourcebiz.ChannelWebSearch, sourcebiz.AdapterTavily, false, sourcebiz.SourceLevelMedia, true},
		{"parallel_search", "Parallel Search", "https://api.parallel.ai/v1/search", sourcebiz.ChannelWebSearch, sourcebiz.AdapterParallel, false, sourcebiz.SourceLevelMedia, true},
		{"cls_telegraph", "财联社电报", "https://www.cls.cn/v1/roll/get_roll_list", sourcebiz.ChannelAPI, sourcebiz.AdapterCLS, true, sourcebiz.SourceLevelWire, false},
		{"eastmoney_fastnews", "东方财富 7x24", "https://np-weblist.eastmoney.com/comm/web/getFastNewsList", sourcebiz.ChannelAPI, sourcebiz.AdapterEastmoneyFast, true, sourcebiz.SourceLevelMedia, false},
		{"eastmoney_stock_news", "东方财富个股新闻", "https://search-api-web.eastmoney.com/search/jsonp", sourcebiz.ChannelAPI, sourcebiz.AdapterEastmoneyStock, true, sourcebiz.SourceLevelMedia, false},
		{"stcn_quicknews", "证券时报快讯", "https://www.stcn.com/article/list.html", sourcebiz.ChannelAPI, sourcebiz.AdapterSTCN, true, sourcebiz.SourceLevelMedia, false},
	}
	result := make([]sourcebiz.Source, 0, len(definitions))
	for _, item := range definitions {
		endpoint := item.endpoint
		if override := strings.TrimSpace(options.Endpoints[item.code]); override != "" {
			endpoint = override
		}
		var appKey *string
		if item.credential {
			if value := strings.TrimSpace(options.AppKeys[item.code]); value != "" {
				appKey = &value
			}
		}
		result = append(result, sourcebiz.Source{
			Code: item.code, Name: item.name, OwnershipType: sourcebiz.OwnershipFixed,
			ChannelType: item.channel, AdapterKey: item.adapter, Enabled: item.enabled,
			Endpoint: endpoint, AppKey: appKey, Config: json.RawMessage(`{}`), Priority: 1,
			TimeoutSeconds: 30, MaxResults: 10, DefaultSourceLevel: item.level,
		})
	}
	return result
}

func DecodeImport(reader io.Reader) ([]sourcebiz.Source, error) {
	if reader == nil {
		return nil, fmt.Errorf("Source import reader is required")
	}
	decoder := json.NewDecoder(io.LimitReader(reader, maxImportBytes+1))
	decoder.DisallowUnknownFields()
	var document struct {
		Sources []sourcebiz.Source `json:"sources"`
	}
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode Source import: %w", err)
	}
	if document.Sources == nil {
		return nil, fmt.Errorf("decode Source import: sources is required")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode Source import: trailing JSON value")
		}
		return nil, fmt.Errorf("decode Source import: %w", err)
	}
	return document.Sources, nil
}
