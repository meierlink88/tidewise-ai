package source_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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
	fixed, err := useCase.PublishFixed(ctx, sourcebiz.CurrentFixedManifest(sourcebiz.FixedManifestOptions{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed) != 7 {
		t.Fatalf("fixed Source count = %d, want 7", len(fixed))
	}
	if _, err := useCase.PublishFixed(ctx, sourcebiz.CurrentFixedManifest(sourcebiz.FixedManifestOptions{})); err != nil {
		t.Fatalf("idempotent fixed publication: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE sources SET name='Operator override' WHERE code='bocha'`); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.PublishFixed(ctx, sourcebiz.CurrentFixedManifest(sourcebiz.FixedManifestOptions{})); err != nil {
		t.Fatalf("missing-only fixed publication: %v", err)
	}
	var fixedName string
	if err := db.QueryRowContext(ctx, `SELECT name FROM sources WHERE code='bocha'`).Scan(&fixedName); err != nil || fixedName != "Operator override" {
		t.Fatalf("fixed publication overwrote mutable state: name=%q err=%v", fixedName, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE sources SET enabled=true WHERE code='tavily'`); err == nil {
		t.Fatal("database accepted a second active web_search Source")
	} else {
		var postgres *pgconn.PgError
		if !errors.As(err, &postgres) || postgres.Code != "23505" {
			t.Fatalf("active web_search uniqueness error = %v, want PostgreSQL 23505", err)
		}
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

func TestPostgresSourceImportReplayAndConcurrentCapacity(t *testing.T) {
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	db := postgresfixture.OpenIsolated(t, "tw_source_capacity", migrationDir, 0)
	store, err := sourcedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := sourcebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 8, 19, 1, 2, 3, 123_456_400, time.UTC)
	input := make([]sourcebiz.Source, 0, 199)
	for index := 0; index < 199; index++ {
		code := fmt.Sprintf("import-%03d", index)
		config := []byte(`{}`)
		if index == 0 {
			config = []byte(`{"nested":{"z":1,"a":2},"max_bytes":5000000}`)
		}
		input = append(input, sourcebiz.Source{
			Code: code, Name: code, OwnershipType: sourcebiz.OwnershipDynamic, ChannelType: sourcebiz.ChannelRSS,
			AdapterKey: sourcebiz.AdapterGenericRSS, Enabled: false, Endpoint: "https://example.com/feed.xml",
			Config: config, Priority: 1, TimeoutSeconds: 30, MaxResults: 10,
			DefaultSourceLevel: sourcebiz.SourceLevelMedia, CreatedAt: stamp, UpdatedAt: stamp,
		})
	}
	if _, err := useCase.Import(context.Background(), input); err != nil {
		t.Fatalf("initial import: %v", err)
	}
	if _, err := useCase.Import(context.Background(), input); err != nil {
		t.Fatalf("semantic JSONB/microsecond replay: %v", err)
	}
	drift := append([]sourcebiz.Source(nil), input...)
	drift[0].Name = "drifted"
	if _, err := useCase.Import(context.Background(), drift); !errors.Is(err, sourcebiz.ErrConflict) {
		t.Fatalf("drift replay error = %v", err)
	}
	var storedName string
	if err := db.QueryRow(`SELECT name FROM sources WHERE code='import-000'`).Scan(&storedName); err != nil || storedName != "import-000" {
		t.Fatalf("drift replay was not atomic: name=%q err=%v", storedName, err)
	}

	var wait sync.WaitGroup
	errorsByCreate := make(chan error, 2)
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			_, createErr := useCase.CreateDynamic(context.Background(), sourcebiz.MutableSource{
				Code: fmt.Sprintf("concurrent-%d", index), Name: "Concurrent", Enabled: false,
				Endpoint: "https://example.com/concurrent.xml", Config: []byte(`{}`), Priority: 1,
				TimeoutSeconds: 30, MaxResults: 10, DefaultSourceLevel: sourcebiz.SourceLevelMedia,
			})
			errorsByCreate <- createErr
		}(index)
	}
	wait.Wait()
	close(errorsByCreate)
	successes, capacityFailures := 0, 0
	for createErr := range errorsByCreate {
		switch {
		case createErr == nil:
			successes++
		case errors.Is(createErr, sourcebiz.ErrCapacityExceeded):
			capacityFailures++
		default:
			t.Fatalf("concurrent create error = %v", createErr)
		}
	}
	if successes != 1 || capacityFailures != 1 {
		t.Fatalf("concurrent capacity outcomes: successes=%d capacity=%d", successes, capacityFailures)
	}
}

func TestDecodeImportRejectsUnknownFieldsAndAcceptsPlaintextAppKey(t *testing.T) {
	input := `{"sources":[{"code":"feed","name":"Feed","ownership_type":"dynamic","channel_type":"rss","adapter_key":"generic_rss","enabled":true,"endpoint":"https://example.com/feed","app_key":"plain","config":{"max_bytes":5000000},"priority":1,"timeout_seconds":30,"max_results":10,"default_source_level":"L3_MEDIA","created_at":"2026-08-19T00:00:00Z","updated_at":"2026-08-19T00:00:00Z"}]}`
	items, err := sourcedata.DecodeImport(strings.NewReader(input))
	if err != nil {
		t.Fatalf("DecodeImport: %v", err)
	}
	if len(items) != 1 || items[0].AppKey == nil || *items[0].AppKey != "plain" {
		t.Fatalf("decoded import = %+v", items)
	}

	if _, err := sourcedata.DecodeImport(strings.NewReader(`{"sources":[],"unexpected":true}`)); err == nil {
		t.Fatal("DecodeImport accepted an unknown top-level field")
	}
}
