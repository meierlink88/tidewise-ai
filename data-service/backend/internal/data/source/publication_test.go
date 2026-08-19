package source

import (
	"strings"
	"testing"

	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

func TestCurrentFixedManifestPreservesAgentOSDefaults(t *testing.T) {
	manifest := CurrentFixedManifest(FixedManifestOptions{
		Endpoints: map[string]string{"bocha": "https://override.example/search"},
		AppKeys:   map[string]string{"bocha": "plain-key"},
	})
	if len(manifest) != 7 {
		t.Fatalf("manifest length = %d, want 7", len(manifest))
	}
	activeWeb := 0
	for _, item := range manifest {
		if item.OwnershipType != sourcebiz.OwnershipFixed {
			t.Errorf("%s ownership = %q, want fixed", item.Code, item.OwnershipType)
		}
		if item.Enabled && item.ChannelType == sourcebiz.ChannelWebSearch {
			activeWeb++
		}
	}
	if activeWeb != 1 {
		t.Fatalf("active web Sources = %d, want 1", activeWeb)
	}
	if manifest[0].Endpoint != "https://override.example/search" || manifest[0].AppKey == nil || *manifest[0].AppKey != "plain-key" {
		t.Fatalf("bocha deployment override not applied: %+v", manifest[0])
	}
}

func TestDecodeImportRejectsUnknownFieldsAndAcceptsPlaintextAppKey(t *testing.T) {
	input := `{"sources":[{"code":"feed","name":"Feed","ownership_type":"dynamic","channel_type":"rss","adapter_key":"generic_rss","enabled":true,"endpoint":"https://example.com/feed","app_key":"plain","config":{"max_bytes":5000000},"priority":1,"timeout_seconds":30,"max_results":10,"default_source_level":"L3_MEDIA","created_at":"2026-08-19T00:00:00Z","updated_at":"2026-08-19T00:00:00Z"}]}`
	items, err := DecodeImport(strings.NewReader(input))
	if err != nil {
		t.Fatalf("DecodeImport: %v", err)
	}
	if len(items) != 1 || items[0].AppKey == nil || *items[0].AppKey != "plain" {
		t.Fatalf("decoded import = %+v", items)
	}

	_, err = DecodeImport(strings.NewReader(`{"sources":[],"unexpected":true}`))
	if err == nil {
		t.Fatal("DecodeImport accepted an unknown top-level field")
	}
}
