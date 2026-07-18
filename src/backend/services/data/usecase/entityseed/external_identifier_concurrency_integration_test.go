package seed

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
)

func TestPostgresExternalIdentifierConcurrentRebindIntegration(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_EXTERNAL_IDENTIFIER_CONCURRENCY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TIDEWISE_EXTERNAL_IDENTIFIER_CONCURRENCY_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	entityKeys := []string{"chain_node:concurrency_a_" + suffix, "chain_node:concurrency_b_" + suffix}
	entityIDs := []string{entitySeedUUID(entityKeys[0]), entitySeedUUID(entityKeys[1])}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM entity_nodes WHERE id IN ($1::uuid, $2::uuid)`, entityIDs[0], entityIDs[1])
	}()
	for index, entityID := range entityIDs {
		if _, err := db.ExecContext(ctx, `
INSERT INTO entity_nodes (id, entity_key, entity_type, layer_code, name, canonical_name, aliases, status)
VALUES ($1::uuid, $2, 'chain_node', 'chain_node', $3, $3, '{}'::text[], 'active')`, entityID, entityKeys[index], "并发测试节点"+suffix+fmt.Sprintf("-%d", index)); err != nil {
			t.Fatal(err)
		}
	}

	identifierA := firstBatchExternalIdentifier(entityKeys[0], ExternalSourceEastmoney, ExternalTaxonomyConcept, "CONCURRENCY-"+suffix, "并发标识")
	identifierB := identifierA
	identifierB.EntityID = entityIDs[1]
	repository := NewPostgresRepository(db)
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, identifier := range []domain.EntityExternalIdentifier{identifierA, identifierB} {
		workers.Add(1)
		go func(candidate domain.EntityExternalIdentifier) {
			defer workers.Done()
			<-start
			_, err := repository.UpsertExternalIdentifier(ctx, candidate)
			results <- err
		}(identifier)
	}
	close(start)
	workers.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case strings.Contains(err.Error(), "identity conflict"):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1/1", successes, conflicts)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM entity_external_identifiers
WHERE source_system=$1 AND source_taxonomy_type=$2 AND external_code=$3`, identifierA.SourceSystem, identifierA.SourceTaxonomyType, identifierA.ExternalCode).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored external identities = %d, want 1", count)
	}
}
