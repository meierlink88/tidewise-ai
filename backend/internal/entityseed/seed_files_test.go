package entityseed

import (
	"path/filepath"
	"strings"
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

func TestEconomySeedFile(t *testing.T) {
	manifest, err := LoadFile(filepath.Join("..", "..", "data", "entity_foundation", "economies.json"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if got, want := len(manifest.Entities), 50; got != want {
		t.Fatalf("entities = %d, want %d", got, want)
	}

	names := map[string]string{}
	for _, entity := range manifest.Entities {
		if entity.EntityType != domain.EntityTypeEconomy {
			t.Fatalf("entity %q type = %q, want %q", entity.Key, entity.EntityType, domain.EntityTypeEconomy)
		}
		names[entity.Key] = entity.Name
		if strings.Contains(entity.Name, "香港") && entity.Name != "中国香港" {
			t.Fatalf("entity %q name = %q, want 中国香港 political naming rule", entity.Key, entity.Name)
		}
		if strings.Contains(entity.Name, "台湾") && entity.Name != "中国台湾" {
			t.Fatalf("entity %q name = %q, want 中国台湾 political naming rule", entity.Key, entity.Name)
		}
	}
	if names["economy:hk"] != "中国香港" {
		t.Fatalf("economy:hk name = %q, want 中国香港", names["economy:hk"])
	}
	if names["economy:tw"] != "中国台湾" {
		t.Fatalf("economy:tw name = %q, want 中国台湾", names["economy:tw"])
	}
}
