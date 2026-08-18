package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActiveEventContractsExcludePrimaryEvidenceFields(t *testing.T) {
	root := repositoryRoot()
	for _, name := range []string{
		"data-service/backend/api/data/v1/openapi.yaml",
		"admin-portal/backend/api/admin/v1/openapi.yaml",
	} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"is_primary", "primary_source_id", "evidence_excerpt"} {
			if strings.Contains(string(content), forbidden) {
				t.Fatalf("active contract %s contains retired field %q", name, forbidden)
			}
		}
	}
}
