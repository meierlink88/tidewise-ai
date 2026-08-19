package source_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
	sourcedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/source"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestPostgresSourceLifecycleAndDatabaseIdentityGuards(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_source", migrationDir, 0)
	store, err := sourcedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := sourcebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	fixed, err := useCase.PublishFixed(ctx, sourcedata.CurrentFixedManifest(sourcedata.FixedManifestOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed) != 7 {
		t.Fatalf("fixed Source count = %d, want 7", len(fixed))
	}
	if _, err := useCase.PublishFixed(ctx, sourcedata.CurrentFixedManifest(sourcedata.FixedManifestOptions{})); err != nil {
		t.Fatalf("idempotent fixed publication: %v", err)
	}

	created, err := useCase.CreateDynamic(ctx, sourcebiz.MutableSource{
		Code: "example_feed", Name: "Example Feed", Enabled: true, Endpoint: "https://example.com/feed.xml",
		Config:   []byte(`{"max_bytes":5000000,"source_levels":{"example.com":"L3_MEDIA"}}`),
		Priority: 2, TimeoutSeconds: 20, MaxResults: 25, DefaultSourceLevel: sourcebiz.SourceLevelMedia,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.OwnershipType != sourcebiz.OwnershipDynamic || created.AdapterKey != sourcebiz.AdapterGenericRSS {
		t.Fatalf("created dynamic Source = %+v", created)
	}
	snapshot, err := useCase.ActiveSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot) != 6 || snapshot[0].ChannelType != sourcebiz.ChannelAPI || snapshot[len(snapshot)-1].ChannelType != sourcebiz.ChannelWebSearch {
		t.Fatalf("snapshot order/count = %+v", snapshot)
	}
	if err := useCase.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM sources WHERE id=$1`, fixed[0].ID); err == nil {
		t.Fatal("database accepted fixed Source deletion")
	} else {
		var postgres *pgconn.PgError
		if !errors.As(err, &postgres) || postgres.Code != "55000" {
			t.Fatalf("fixed delete error = %v, want PostgreSQL 55000", err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE sources SET code='changed' WHERE id=$1`, fixed[0].ID); err == nil {
		t.Fatal("database accepted fixed Source code mutation")
	}
}
