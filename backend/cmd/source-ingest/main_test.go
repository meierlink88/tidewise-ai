package main

import (
	"reflect"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

func TestSourceCatalogFilterIncludesSourceID(t *testing.T) {
	filter := sourceCatalogFilter(sourceIngestOptions{
		sourceID:      "tidewise:ai-web-research:cn-finance-daily",
		providerKey:   "llm_web_research",
		ingestChannel: "ai_web_research",
		sourceType:    "news",
	})

	want := repositories.SourceCatalogFilter{
		SourceID:      "tidewise:ai-web-research:cn-finance-daily",
		ProviderKey:   "llm_web_research",
		IngestChannel: "ai_web_research",
		SourceType:    "news",
	}
	if !reflect.DeepEqual(filter, want) {
		t.Fatalf("filter = %+v, want %+v", filter, want)
	}
}

func TestRequiredEnvParsingAndMissingDetection(t *testing.T) {
	names := parseRequiredEnvNames(" TAVILY_API_KEY, BOCHA_API_KEY,,DEEPSEEK_API_KEY ")
	if want := []string{"TAVILY_API_KEY", "BOCHA_API_KEY", "DEEPSEEK_API_KEY"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}

	missing := missingRequiredEnvNames(names, func(name string) string {
		if name == "TAVILY_API_KEY" {
			return "present"
		}
		return ""
	})
	if want := []string{"BOCHA_API_KEY", "DEEPSEEK_API_KEY"}; !reflect.DeepEqual(missing, want) {
		t.Fatalf("missing = %v, want %v", missing, want)
	}
}

func TestIngestionTimeoutSecondsUsesOverrideWhenProvided(t *testing.T) {
	if got := ingestionTimeoutSeconds(sourceIngestOptions{timeoutSeconds: 45}, 10); got != 45 {
		t.Fatalf("timeout seconds = %d, want override", got)
	}
	if got := ingestionTimeoutSeconds(sourceIngestOptions{timeoutSeconds: 0}, 10); got != 10 {
		t.Fatalf("timeout seconds = %d, want config default", got)
	}
	if got := ingestionTimeoutSeconds(sourceIngestOptions{timeoutSeconds: -1}, 10); got != 10 {
		t.Fatalf("timeout seconds = %d, want config default for negative override", got)
	}
}
