package connectors

import (
	"fmt"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type AIWebResearchConfig struct {
	WebSearchPlan     WebSearchPlan
	CredentialRefs    map[string]string
	LLMProvider       string
	APIBaseURL        string
	APIProtocol       string
	Model             string
	PromptRef         string
	PromptVersion     string
	PromptVariables   map[string]any
	MaxResults        int
	OutputSchema      map[string]any
	SourcePreferences map[string]any
	TrustedDomains    []string
}

type WebSearchPlan struct {
	Mode  string
	Tools []WebSearchToolConfig
}

type WebSearchToolConfig struct {
	Provider      string
	Role          string
	CredentialRef string
	MaxResults    int
	Options       map[string]any
}

func ParseAIWebResearchConfig(source domain.SourceCatalog) (AIWebResearchConfig, error) {
	config := source.SourceConfig
	if _, ok := config["web_search_plan"]; !ok {
		return AIWebResearchConfig{}, fmt.Errorf("web_search_plan is required")
	}
	plan, err := parseWebSearchPlan(config["web_search_plan"])
	if err != nil {
		return AIWebResearchConfig{}, err
	}

	credentialRefs, err := parseStringMap(config, "credential_refs")
	if err != nil {
		return AIWebResearchConfig{}, err
	}
	outputSchema, err := parseAnyMap(config, "output_schema")
	if err != nil {
		return AIWebResearchConfig{}, err
	}
	sourcePreferences, err := parseAnyMap(config, "source_preferences")
	if err != nil {
		return AIWebResearchConfig{}, err
	}
	trustedDomains, err := parseStringSlice(config, "trusted_domains")
	if err != nil {
		return AIWebResearchConfig{}, err
	}

	parsed := AIWebResearchConfig{
		WebSearchPlan:     plan,
		CredentialRefs:    credentialRefs,
		LLMProvider:       requiredConfigString(config, "llm_provider"),
		APIBaseURL:        requiredConfigString(config, "api_base_url"),
		APIProtocol:       requiredConfigString(config, "api_protocol"),
		Model:             requiredConfigString(config, "model"),
		PromptRef:         requiredConfigString(config, "prompt_ref"),
		PromptVersion:     requiredConfigString(config, "prompt_version"),
		PromptVariables:   optionalAnyMap(config["prompt_variables"]),
		MaxResults:        connectorPositiveInt(config["max_results"]),
		OutputSchema:      outputSchema,
		SourcePreferences: sourcePreferences,
		TrustedDomains:    trustedDomains,
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{name: "llm_provider", value: parsed.LLMProvider},
		{name: "api_base_url", value: parsed.APIBaseURL},
		{name: "api_protocol", value: parsed.APIProtocol},
		{name: "model", value: parsed.Model},
		{name: "prompt_ref", value: parsed.PromptRef},
		{name: "prompt_version", value: parsed.PromptVersion},
	} {
		if check.value == "" {
			return AIWebResearchConfig{}, fmt.Errorf("%s is required", check.name)
		}
	}
	if parsed.MaxResults <= 0 {
		return AIWebResearchConfig{}, fmt.Errorf("max_results must be positive")
	}
	return parsed, nil
}

func parseWebSearchPlan(value any) (WebSearchPlan, error) {
	planMap, ok := value.(map[string]any)
	if !ok {
		return WebSearchPlan{}, fmt.Errorf("web_search_plan must be an object")
	}
	mode := jsonStringValue(planMap["mode"])
	if mode == "" {
		return WebSearchPlan{}, fmt.Errorf("web_search_plan mode is required")
	}
	toolsValue, ok := planMap["tools"].([]any)
	if !ok || len(toolsValue) == 0 {
		return WebSearchPlan{}, fmt.Errorf("web_search_plan tools are required")
	}

	tools := make([]WebSearchToolConfig, 0, len(toolsValue))
	for _, item := range toolsValue {
		toolMap, ok := item.(map[string]any)
		if !ok {
			return WebSearchPlan{}, fmt.Errorf("web_search_plan tool must be an object")
		}
		tool := WebSearchToolConfig{
			Provider:      jsonStringValue(toolMap["provider"]),
			Role:          jsonStringValue(toolMap["role"]),
			CredentialRef: jsonStringValue(toolMap["credential_ref"]),
			MaxResults:    connectorPositiveInt(toolMap["max_results"]),
			Options:       optionalAnyMap(toolMap["options"]),
		}
		if tool.Provider == "" {
			return WebSearchPlan{}, fmt.Errorf("tool provider is required")
		}
		if tool.CredentialRef == "" {
			return WebSearchPlan{}, fmt.Errorf("tool credential_ref is required")
		}
		if tool.MaxResults <= 0 {
			return WebSearchPlan{}, fmt.Errorf("tool max_results must be positive")
		}
		tools = append(tools, tool)
	}
	return WebSearchPlan{Mode: mode, Tools: tools}, nil
}

func requiredConfigString(config map[string]any, key string) string {
	if _, ok := config[key]; !ok {
		return ""
	}
	return jsonStringValue(config[key])
}

func parseStringMap(config map[string]any, key string) (map[string]string, error) {
	value, ok := config[key]
	if !ok {
		return nil, fmt.Errorf("%s is required", key)
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	result := make(map[string]string, len(raw))
	for itemKey, itemValue := range raw {
		result[itemKey] = jsonStringValue(itemValue)
	}
	return result, nil
}

func parseAnyMap(config map[string]any, key string) (map[string]any, error) {
	value, ok := config[key]
	if !ok {
		return nil, fmt.Errorf("%s is required", key)
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	return raw, nil
}

func optionalAnyMap(value any) map[string]any {
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return raw
}

func parseStringSlice(config map[string]any, key string) ([]string, error) {
	value, ok := config[key]
	if !ok {
		return nil, fmt.Errorf("%s is required", key)
	}
	raw, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		text := jsonStringValue(item)
		if text != "" {
			result = append(result, text)
		}
	}
	return result, nil
}

func connectorPositiveInt(value any) int {
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
