package entityseed

import (
	"path/filepath"
	"testing"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entityseed"
)

func approvedManifest(t *testing.T) AllianceEconomyManifest {
	t.Helper()

	path := filepath.Join("..", "..", "..", "data", "entity_foundation", "alliance_economy", "approved_manifest_v1.json")
	manifest, err := biz.LoadApprovedAllianceEconomyManifest(path)
	if err != nil {
		t.Fatalf("load approved alliance economy manifest: %v", err)
	}
	return manifest
}
