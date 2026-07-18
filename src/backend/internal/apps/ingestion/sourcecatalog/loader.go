package sourcecatalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type Manifest struct {
	Sources []Source `json:"sources"`
}

type Source struct {
	ID              string                     `json:"id"`
	OriginSystem    string                     `json:"origin_system"`
	Stage           string                     `json:"stage"`
	IngestChannel   string                     `json:"ingest_channel"`
	ProviderKey     string                     `json:"provider_key"`
	ConnectorKey    string                     `json:"connector_key"`
	ParserKey       string                     `json:"parser_key"`
	SourceType      string                     `json:"source_type"`
	SourceGroup     string                     `json:"source_group"`
	SourceName      string                     `json:"source_name"`
	SourceURL       string                     `json:"source_url"`
	SourceLevel     string                     `json:"source_level"`
	TopicHint       string                     `json:"topic_hint"`
	RouteTemplate   string                     `json:"route_template,omitempty"`
	CodeStyle       string                     `json:"code_style,omitempty"`
	AuthRequired    bool                       `json:"auth_required,omitempty"`
	AuthType        string                     `json:"auth_type,omitempty"`
	CredentialRef   string                     `json:"credential_ref,omitempty"`
	SourceConfig    map[string]any             `json:"source_config,omitempty"`
	RateLimitPolicy map[string]any             `json:"rate_limit_policy,omitempty"`
	UsagePolicy     string                     `json:"usage_policy"`
	Status          domain.SourceCatalogStatus `json:"status"`
}

func LoadFile(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read source catalog manifest: %w", err)
	}
	return Load(data)
}

func LoadFiles(paths ...string) (Manifest, error) {
	var merged Manifest
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return Manifest{}, fmt.Errorf("read source catalog manifest %q: %w", path, err)
		}
		if err := rejectSensitiveFields(data); err != nil {
			return Manifest{}, fmt.Errorf("%s: %w", path, err)
		}
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return Manifest{}, fmt.Errorf("decode source catalog manifest %q: %w", path, err)
		}
		merged.Sources = append(merged.Sources, manifest.Sources...)
	}
	if err := Validate(merged); err != nil {
		return Manifest{}, err
	}
	normalizeDefaults(&merged)
	return merged, nil
}

func Load(data []byte) (Manifest, error) {
	if err := rejectSensitiveFields(data); err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode source catalog manifest: %w", err)
	}
	if err := Validate(manifest); err != nil {
		return Manifest{}, err
	}
	normalizeDefaults(&manifest)
	return manifest, nil
}

func Validate(manifest Manifest) error {
	sourceIDs := make(map[string]struct{}, len(manifest.Sources))
	for _, source := range manifest.Sources {
		if source.ID == "" {
			return fmt.Errorf("source id is required")
		}
		if _, ok := sourceIDs[source.ID]; ok {
			return fmt.Errorf("duplicate source id %q", source.ID)
		}
		sourceIDs[source.ID] = struct{}{}
		if err := validateSource(source); err != nil {
			return fmt.Errorf("source %q: %w", source.ID, err)
		}
	}
	return nil
}

func (s Source) SourceCatalog() domain.SourceCatalog {
	return domain.SourceCatalog{
		ID:              s.ID,
		IngestChannel:   s.IngestChannel,
		ProviderKey:     s.ProviderKey,
		ConnectorKey:    s.ConnectorKey,
		ParserKey:       s.ParserKey,
		SourceType:      s.SourceType,
		SourceName:      s.SourceName,
		SourceURL:       s.SourceURL,
		SourceLevel:     s.SourceLevel,
		TopicHint:       s.TopicHint,
		RouteTemplate:   s.RouteTemplate,
		CodeStyle:       s.CodeStyle,
		AuthRequired:    s.AuthRequired,
		AuthType:        s.AuthType,
		CredentialRef:   s.CredentialRef,
		SourceConfig:    cloneMap(s.SourceConfig),
		RateLimitPolicy: cloneMap(s.RateLimitPolicy),
		UsagePolicy:     s.UsagePolicy,
		Status:          s.Status,
	}
}

func validateSource(source Source) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "origin_system", value: source.OriginSystem},
		{name: "stage", value: source.Stage},
		{name: "ingest_channel", value: source.IngestChannel},
		{name: "provider_key", value: source.ProviderKey},
		{name: "connector_key", value: source.ConnectorKey},
		{name: "parser_key", value: source.ParserKey},
		{name: "source_type", value: source.SourceType},
		{name: "source_group", value: source.SourceGroup},
		{name: "source_name", value: source.SourceName},
		{name: "source_level", value: source.SourceLevel},
		{name: "topic_hint", value: source.TopicHint},
		{name: "usage_policy", value: source.UsagePolicy},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}

	if source.Status != domain.SourceCatalogStatusActive &&
		source.Status != domain.SourceCatalogStatusInactive &&
		source.Status != domain.SourceCatalogStatusDisabled {
		return fmt.Errorf("unsupported status %q", source.Status)
	}
	if !validConnectorParser(source.ConnectorKey, source.ParserKey) {
		return fmt.Errorf("unsupported connector/parser combination %q/%q", source.ConnectorKey, source.ParserKey)
	}
	if source.ConnectorKey == "llm_web_research" {
		if err := validateAIWebResearchSourceConfig(source.SourceConfig); err != nil {
			return err
		}
	}
	if err := validateSourceLocator(source); err != nil {
		return err
	}
	return nil
}

func validateSourceLocator(source Source) error {
	switch source.ConnectorKey {
	case "rss_feed", "web_fetch", "eastmoney":
		parsed, err := url.ParseRequestURI(source.SourceURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("source_url must be an absolute http url")
		}
	case "rsshub_feed":
		if strings.TrimSpace(source.RouteTemplate) == "" {
			return fmt.Errorf("route_template is required")
		}
	case "local_file":
		if strings.TrimSpace(source.SourceURL) == "" {
			return fmt.Errorf("source_url is required")
		}
	}
	return nil
}

func validConnectorParser(connector string, parser string) bool {
	valid := map[string]map[string]struct{}{
		"rss_feed":         {"rss_item": {}},
		"rsshub_feed":      {"rss_item": {}},
		"web_fetch":        {"text": {}},
		"local_file":       {"text": {}},
		"eastmoney":        {"eastmoney_json": {}, "text": {}},
		"market_provider":  {"provider_metadata": {}},
		"local_backfill":   {"file_manifest": {}},
		"llm_web_research": {"llm_research_items": {}},
	}
	parsers, ok := valid[connector]
	if !ok {
		return false
	}
	_, ok = parsers[parser]
	return ok
}

func validateAIWebResearchSourceConfig(config map[string]any) error {
	for _, key := range []string{
		"kind",
		"web_search_plan",
		"max_results",
		"output_schema",
		"source_preferences",
		"trusted_domains",
	} {
		if _, ok := config[key]; !ok {
			return fmt.Errorf("%s is required", key)
		}
	}
	if jsonStringValue(config["kind"]) != "llm_web_research" {
		return fmt.Errorf("source_config kind must be llm_web_research")
	}
	isStaticSearchPlan := jsonStringValue(config["collection_mode"]) == "search_results" && jsonStringValue(config["search_plan_mode"]) == "static_query_plan"
	if isStaticSearchPlan {
		if err := validateStaticSearchQueries(config["search_queries"]); err != nil {
			return err
		}
	} else {
		for _, key := range []string{
			"credential_refs",
			"llm_provider",
			"api_base_url",
			"api_protocol",
			"model",
			"prompt_ref",
			"prompt_version",
		} {
			if _, ok := config[key]; !ok {
				return fmt.Errorf("%s is required", key)
			}
		}
	}
	plan, ok := config["web_search_plan"].(map[string]any)
	if !ok {
		return fmt.Errorf("web_search_plan must be an object")
	}
	mode := jsonStringValue(plan["mode"])
	if mode == "" {
		return fmt.Errorf("web_search_plan mode is required")
	}
	if mode != "parallel" && mode != "fallback" && mode != "sequential" {
		return fmt.Errorf("unsupported web_search_plan mode %q", mode)
	}
	tools, ok := plan["tools"].([]any)
	if !ok || len(tools) == 0 {
		return fmt.Errorf("web_search_plan tools are required")
	}
	for _, tool := range tools {
		toolConfig, ok := tool.(map[string]any)
		if !ok {
			return fmt.Errorf("web_search_plan tool must be an object")
		}
		provider := jsonStringValue(toolConfig["provider"])
		if provider == "" {
			return fmt.Errorf("web_search_plan tool provider is required")
		}
		if provider != "tavily" && provider != "bocha_web_search" {
			return fmt.Errorf("unsupported web_search provider %q", provider)
		}
		if jsonStringValue(toolConfig["credential_ref"]) == "" {
			return fmt.Errorf("web_search_plan tool credential_ref is required")
		}
		if configPositiveInt(toolConfig["max_results"]) <= 0 {
			return fmt.Errorf("web_search_plan tool max_results must be positive")
		}
	}
	if refs, ok := config["credential_refs"]; ok {
		if _, ok := refs.(map[string]any); !ok {
			return fmt.Errorf("credential_refs must be an object")
		}
	}
	if !isStaticSearchPlan {
		if _, ok := config["credential_refs"].(map[string]any); !ok {
			return fmt.Errorf("credential_refs must be an object")
		}
	}
	if configPositiveInt(config["max_results"]) <= 0 {
		return fmt.Errorf("max_results must be positive")
	}
	if _, ok := config["output_schema"].(map[string]any); !ok {
		return fmt.Errorf("output_schema must be an object")
	}
	if _, ok := config["source_preferences"].(map[string]any); !ok {
		return fmt.Errorf("source_preferences must be an object")
	}
	if _, ok := config["trusted_domains"].([]any); !ok {
		return fmt.Errorf("trusted_domains must be an array")
	}
	return nil
}

func validateStaticSearchQueries(value any) error {
	queries, ok := value.([]any)
	if !ok || len(queries) == 0 {
		return fmt.Errorf("search_queries are required")
	}
	for _, query := range queries {
		queryConfig, ok := query.(map[string]any)
		if !ok {
			return fmt.Errorf("search query must be an object")
		}
		if jsonStringValue(queryConfig["query"]) == "" {
			return fmt.Errorf("search query is required")
		}
		if configPositiveInt(queryConfig["max_results"]) <= 0 {
			return fmt.Errorf("search query max_results must be positive")
		}
		if providers, ok := queryConfig["providers"]; ok {
			providerItems, ok := providers.([]any)
			if !ok {
				return fmt.Errorf("search query providers must be an array")
			}
			for _, item := range providerItems {
				provider := jsonStringValue(item)
				if provider != "tavily" && provider != "bocha_web_search" {
					return fmt.Errorf("unsupported search query provider %q", provider)
				}
			}
		}
	}
	return nil
}

func normalizeDefaults(manifest *Manifest) {
	for i := range manifest.Sources {
		if manifest.Sources[i].AuthType == "" {
			manifest.Sources[i].AuthType = "none"
		}
		if manifest.Sources[i].SourceConfig == nil {
			manifest.Sources[i].SourceConfig = map[string]any{}
		}
		if manifest.Sources[i].RateLimitPolicy == nil {
			manifest.Sources[i].RateLimitPolicy = map[string]any{}
		}
	}
}

func jsonStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%g", v))
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func configPositiveInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func rejectSensitiveFields(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("decode source catalog manifest: %w", err)
	}
	return scanSensitiveFields(value)
}

func scanSensitiveFields(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if forbiddenSensitiveField(key) {
				return fmt.Errorf("forbidden sensitive field %q", key)
			}
			if err := scanSensitiveFields(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range typed {
			if err := scanSensitiveFields(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func forbiddenSensitiveField(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	if normalized == "credential_ref" {
		return false
	}
	for _, token := range []string{"api_key", "apikey", "token", "bearer", "cookie", "password", "secret"} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}
