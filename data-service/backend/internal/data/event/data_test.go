package event

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestPostgresEventAggregateCreateAndRead(t *testing.T) {
	db := openEventTestDatabase(t)
	evidenceID := publishAtomicEvidence(t, db, "event-aggregate-create")
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	strength, confidence, magnitude := 0.85, 0.92, 0.75
	created, err := useCase.Create(context.Background(), eventbiz.CreateInput{
		Title:   "Example Corp expands capacity",
		Summary: "Example Corp will add a new production line.",
		Semantic: eventbiz.Semantic{
			Who: stringPointer("Example Corp"), What: stringPointer("adds a production line"),
			When: stringPointer("2026-08-18"), Where: stringPointer("Shanghai"),
			Why: stringPointer("meet demand"), How: stringPointer("capital investment"),
		},
		Modality: eventbiz.ModalityPlan, AnnouncedAt: &publishedAt,
		Evidence: []eventbiz.EvidenceLinkInput{{EvidenceID: evidenceID, ContributionWeight: 0.80}},
		Actors: []eventbiz.ActorLinkInput{{
			ActorID: "actor:example-corp", ActorType: eventbiz.ActorTypeCompany,
			ActorName: stringPointer("Example Corp"), RelationType: eventbiz.ActorRelationOriginatesFrom,
			RelationStrength: &strength, Confidence: &confidence,
		}},
		Assets: []eventbiz.AssetLinkInput{{
			AssetID: "asset:example-corp-security", AssetType: eventbiz.AssetTypeSecurity,
			AssetName: stringPointer("Example Corp Equity"), ImpactDirection: eventbiz.ImpactDirectionPositive,
			ImpactMagnitude: &magnitude,
		}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := useCase.Get(context.Background(), created.Event.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}
}

func TestPostgresEventCreateIsAtomicAndRequiresEvidence(t *testing.T) {
	db := openEventTestDatabase(t)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = useCase.Create(context.Background(), eventbiz.CreateInput{
		Title: "Unknown evidence", Summary: "The transaction must roll back.", Semantic: eventbiz.Semantic{},
		Modality: eventbiz.ModalityFact,
		Evidence: []eventbiz.EvidenceLinkInput{{
			EvidenceID: "EVD11111111-1111-4111-8111-111111111111", ContributionWeight: 0.5,
		}},
	})
	if err == nil {
		t.Fatal("Create() error = nil")
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM events`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("Events after failed create = %d, %v", count, err)
	}
	assertPostgresCode(t, db, "23514", `INSERT INTO events (
    id,title,summary,semantic,modality,status
) VALUES (
    'EVT11111111-1111-4111-8111-111111111111','orphan','orphan summary',
    '{"who":null,"what":null,"when":null,"where":null,"why":null,"how":null}',
    'FACT','ACTIVE'
)`)
}

func TestPostgresEventSchemaReplacesLegacyContracts(t *testing.T) {
	db := openEventTestDatabase(t)
	rows, err := db.Query(`SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = current_schema() AND table_name = 'events'
ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	columns := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		if err := rows.Scan(&name, &dataType); err != nil {
			t.Fatal(err)
		}
		columns[name] = dataType
	}
	if err := errors.Join(rows.Err(), rows.Close()); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"id": "character varying", "title": "character varying", "summary": "text", "semantic": "jsonb",
		"modality": "character varying", "occurred_at": "timestamp with time zone",
		"announced_at": "timestamp with time zone", "status": "character varying",
	}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("Event columns = %#v, want %#v", columns, want)
	}
	for _, table := range []string{"event_evidence_links", "event_actor_links", "event_asset_links"} {
		if !tableExists(t, db, table) {
			t.Errorf("target table %q does not exist", table)
		}
	}
	for _, table := range []string{"event_sources", "raw_documents", "event_tag_defs", "event_tag_maps", "event_publication_receipts", "event_entity_links"} {
		if tableExists(t, db, table) {
			t.Errorf("retired table %q still exists", table)
		}
	}
	assertPostgresCode(t, db, "23514", `INSERT INTO events (
    id,title,summary,semantic,modality,status
) VALUES (
    'EVT11111111-1111-4111-8111-111111111112','bad semantic','bad semantic',
    '{"who":null,"what":null,"when":null,"where":null,"why":null,"how":null,"extra":true}',
    'FACT','ACTIVE'
)`)
}

func TestPostgresEventListUsesNewFiltersAndStableFields(t *testing.T) {
	db := openEventTestDatabase(t)
	evidenceID := publishAtomicEvidence(t, db, "event-list")
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	announced := time.Date(2026, 8, 18, 2, 0, 0, 0, time.UTC)
	created, err := useCase.Create(context.Background(), eventbiz.CreateInput{
		Title: "Filtered Event", Summary: "Filtered Event summary.", Semantic: eventbiz.Semantic{},
		Modality: eventbiz.ModalitySpec, AnnouncedAt: &announced,
		Evidence: []eventbiz.EvidenceLinkInput{{EvidenceID: evidenceID, ContributionWeight: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := useCase.ListEvents(context.Background(), eventbiz.EventListRequest{
		Title: "Filtered", Modality: eventbiz.ModalitySpec, Status: eventbiz.LifecycleStatusActive,
		AnnouncedFrom: timePointer(announced.Add(-time.Minute)), AnnouncedTo: timePointer(announced.Add(time.Minute)),
		Page: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0], created.Event) {
		t.Fatalf("ListEvents() = %#v, want created Event", page)
	}
}

func publishAtomicEvidence(t *testing.T, db *sql.DB, publicationKey string) string {
	t.Helper()
	store, err := evidencedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := evidencebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Date(2026, 8, 18, 1, 0, 0, 0, time.UTC)
	raw, err := useCase.PublishRawEvidence(context.Background(), evidencebiz.RawEvidence{
		PublicationKey: publicationKey, SourceID: "SRC_event_aggregate", SourceName: "Example Wire",
		SourceLevel: evidencebiz.SourceLevelWire, SourceURL: "https://example.test/event-aggregate", IsOriginal: true,
		RawText: "Example Corp announced a new production line.", PublishedAt: &publishedAt,
		CollectedAt: publishedAt.Add(5 * time.Minute), Keywords: []string{"production"},
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := useCase.PublishEvidence(context.Background(), raw.ID, []evidencebiz.Evidence{{
		Summary:  "Example Corp announced a new production line.",
		Semantic: evidencebiz.Semantic{Who: stringPointer("Example Corp"), What: "announced a new production line"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return evidence.IDs[0]
}

func assertPostgresCode(t *testing.T, db *sql.DB, wantCode, query string, args ...any) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), query, args...)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != wantCode {
		t.Fatalf("PostgreSQL error = %T %v, want SQLSTATE %s", err, err, wantCode)
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass(current_schema() || '.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

func openEventTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event", migrationDir, 0)
}

func stringPointer(value string) *string { return &value }

func timePointer(value time.Time) *time.Time { return &value }

func validEventID(t *testing.T, id string) {
	t.Helper()
	if !coreid.Is(id, coreid.Event) {
		t.Fatalf("invalid Event ID %q", id)
	}
}
