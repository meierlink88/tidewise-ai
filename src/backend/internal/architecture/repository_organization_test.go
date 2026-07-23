package architecture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryFilesFollowBusinessResponsibilities(t *testing.T) {
	root := filepath.Join("..", "..", "services", "data", "repositories")
	for _, name := range []string{
		"admin_query.go",
		"benchmark_observation.go",
		"event_publication.go",
		"graph_projection.go",
		"identity.go",
		"memory.go",
		"postgres.go",
		"research_read.go",
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("repository business file %q is missing: %v", name, err)
		}
	}

	for _, name := range []string{"doc.go", "ingestion_run.go", "postgres_repository.go", "raw_document.go", "repository.go", "scheduler.go", "source_catalog.go", "uuid.go"} {
		_, err := os.Stat(filepath.Join(root, name))
		if err == nil {
			t.Fatalf("legacy repository file %q still exists", name)
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stat legacy repository file %q: %v", name, err)
		}
	}
}
