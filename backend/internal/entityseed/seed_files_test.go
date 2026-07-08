package entityseed

import (
	"path/filepath"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

func TestAllianceOrgSeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "data", "entity_foundation", "alliance_orgs.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 10; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	required := map[string]struct{}{
		"alliance_org:opec_plus":  {},
		"alliance_org:opec":       {},
		"alliance_org:g7":         {},
		"alliance_org:g20":        {},
		"alliance_org:wto":        {},
		"alliance_org:imf":        {},
		"alliance_org:world_bank": {},
		"alliance_org:oecd":       {},
		"alliance_org:eu":         {},
		"alliance_org:brics":      {},
	}
	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeAllianceOrg {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeAllianceOrg)
		}
		delete(required, entity.Key)
	}
	if len(required) > 0 {
		t.Fatalf("missing alliance org entities: %v", required)
	}
}
