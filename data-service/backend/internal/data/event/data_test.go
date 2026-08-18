package event

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
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

	firstEvidenceID := publishAtomicEvidence(t, db, "event-evidence-move-first")
	secondEvidenceID := publishAtomicEvidence(t, db, "event-evidence-move-second")
	first, err := useCase.Create(context.Background(), validCreateInput("First Event", firstEvidenceID))
	if err != nil {
		t.Fatal(err)
	}
	second, err := useCase.Create(context.Background(), validCreateInput("Second Event", secondEvidenceID))
	if err != nil {
		t.Fatal(err)
	}
	assertPostgresCode(t, db, "23514", `UPDATE event_evidence_links SET event_id = $2 WHERE id = $1`,
		first.Evidence[0].ID, second.Event.ID)
	var retainedEventID string
	if err := db.QueryRow(`SELECT event_id FROM event_evidence_links WHERE id = $1`, first.Evidence[0].ID).Scan(&retainedEventID); err != nil || retainedEventID != first.Event.ID {
		t.Fatalf("failed EEL move persisted event_id = %q, error = %v", retainedEventID, err)
	}

	failedEventID := mustDomainID(t, coreid.Event)
	failedEELID := mustDomainID(t, coreid.EventEvidenceLink)
	failedActorID := mustDomainID(t, coreid.EventActorLink)
	err = store.CreateEvent(context.Background(), eventbiz.Aggregate{
		Event:    eventbiz.Event{ID: failedEventID, Title: "Rollback Event", Summary: "Rollback summary", Semantic: eventbiz.Semantic{}, Modality: eventbiz.ModalityFact, Status: eventbiz.LifecycleStatusActive},
		Evidence: []eventbiz.EvidenceLink{{ID: failedEELID, EventID: failedEventID, EvidenceID: firstEvidenceID, ContributionWeight: 1}},
		Actors:   []eventbiz.ActorLink{{ID: failedActorID, EventID: failedEventID, ActorID: "actor:rollback", RelationType: eventbiz.ActorRelationMentions, Confidence: 1}},
	})
	if err == nil {
		t.Fatal("Store.CreateEvent() accepted invalid Actor confidence")
	}
	if err := db.QueryRow(`SELECT count(*) FROM events WHERE id = $1`, failedEventID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed relationship transaction retained Event = %d, %v", count, err)
	}
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
	for name, semantic := range map[string]string{
		"missing":    `{"who":null,"what":null,"when":null,"where":null,"why":null}`,
		"non-string": `{"who":1,"what":null,"when":null,"where":null,"why":null,"how":null}`,
	} {
		t.Run("semantic "+name, func(t *testing.T) {
			assertPostgresCode(t, db, "23514", `INSERT INTO events (id,title,summary,semantic,modality,status) VALUES ($1,'bad semantic','bad semantic',$2,'FACT','ACTIVE')`,
				mustDomainID(t, coreid.Event), semantic)
		})
	}
	validSemantic := `{"who":"","what":"","when":"","where":"","why":"","how":""}`
	assertPostgresCode(t, db, "23514", `INSERT INTO events (id,title,summary,semantic,modality,status) VALUES ($1,'bad modality','bad modality',$2,'ACTUAL','ACTIVE')`,
		mustDomainID(t, coreid.Event), validSemantic)
	assertPostgresCode(t, db, "23514", `INSERT INTO events (id,title,summary,semantic,modality,status) VALUES ($1,'bad status','bad status',$2,'FACT','CURRENT')`,
		mustDomainID(t, coreid.Event), validSemantic)

	evidenceID := publishAtomicEvidence(t, db, "event-boundary-primary")
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	zero, one, ninetyNine := 0.0, 1.0, 0.99
	actorTypes := []eventbiz.ActorType{eventbiz.ActorTypeCountry, eventbiz.ActorTypePerson, eventbiz.ActorTypeOrganization, eventbiz.ActorTypeCompany}
	relations := []eventbiz.ActorRelationType{eventbiz.ActorRelationMentions, eventbiz.ActorRelationAffects, eventbiz.ActorRelationOriginatesFrom, eventbiz.ActorRelationTargets}
	actors := make([]eventbiz.ActorLinkInput, 0, len(actorTypes))
	for index := range actorTypes {
		strength, confidence := &zero, &zero
		if index%2 == 1 {
			strength, confidence = &one, &ninetyNine
		}
		actors = append(actors, eventbiz.ActorLinkInput{ActorID: fmt.Sprintf("actor:%d", index), ActorType: actorTypes[index], RelationType: relations[index], RelationStrength: strength, Confidence: confidence})
	}
	assetTypes := []eventbiz.AssetType{eventbiz.AssetTypeSecurity, eventbiz.AssetTypeCommodity, eventbiz.AssetTypeIndex, eventbiz.AssetTypeRate, eventbiz.AssetTypeForex, eventbiz.AssetTypeDerivative}
	directions := []eventbiz.ImpactDirection{eventbiz.ImpactDirectionPositive, eventbiz.ImpactDirectionNegative, eventbiz.ImpactDirectionNeutral}
	assets := make([]eventbiz.AssetLinkInput, 0, len(assetTypes))
	for index, assetType := range assetTypes {
		magnitude := &zero
		if index%2 == 1 {
			magnitude = &one
		}
		assets = append(assets, eventbiz.AssetLinkInput{AssetID: fmt.Sprintf("asset:%d", index), AssetType: assetType, ImpactDirection: directions[index%len(directions)], ImpactMagnitude: magnitude})
	}
	created, err := useCase.Create(context.Background(), eventbiz.CreateInput{
		Title: "Boundary Event", Summary: "Boundary summary",
		Semantic: eventbiz.Semantic{Who: stringPointer(""), What: stringPointer(""), When: stringPointer(""), Where: stringPointer(""), Why: stringPointer(""), How: stringPointer("")},
		Modality: eventbiz.ModalityFact, Status: eventbiz.LifecycleStatusActive,
		Evidence: []eventbiz.EvidenceLinkInput{{EvidenceID: evidenceID, ContributionWeight: 0}}, Actors: actors, Assets: assets,
	})
	if err != nil || len(created.Actors) != 4 || len(created.Assets) != 6 {
		t.Fatalf("accepted relationship boundaries = %#v, error = %v", created, err)
	}
	for index, pair := range []struct {
		modality eventbiz.Modality
		status   eventbiz.LifecycleStatus
	}{{eventbiz.ModalityPlan, eventbiz.LifecycleStatusDeprecated}, {eventbiz.ModalitySpec, eventbiz.LifecycleStatusArchived}} {
		input := validCreateInput(fmt.Sprintf("Enum Event %d", index), evidenceID)
		input.Modality, input.Status = pair.modality, pair.status
		if _, err := useCase.Create(context.Background(), input); err != nil {
			t.Fatalf("accepted Event enum pair %#v: %v", pair, err)
		}
	}

	extraEvidenceID := publishAtomicEvidence(t, db, "event-boundary-secondary")
	for _, weight := range []float64{-0.01, 1.01} {
		assertPostgresCode(t, db, "23514", `INSERT INTO event_evidence_links (id,event_id,evidence_id,contribution_weight) VALUES ($1,$2,$3,$4)`,
			mustDomainID(t, coreid.EventEvidenceLink), created.Event.ID, extraEvidenceID, weight)
	}
	assertPostgresCode(t, db, "23505", `INSERT INTO event_evidence_links (id,event_id,evidence_id,contribution_weight) VALUES ($1,$2,$3,1)`,
		mustDomainID(t, coreid.EventEvidenceLink), created.Event.ID, evidenceID)

	actorInsert := `INSERT INTO event_actor_links (id,event_id,actor_id,actor_type,relation_type,relation_strength,confidence) VALUES ($1,$2,$3,$4,$5,$6,$7)`
	for _, test := range []struct {
		code                         string
		actorID, actorType, relation string
		strength, confidence         float64
	}{
		{code: "23514", actorID: " ", actorType: "COUNTRY", relation: "MENTIONS", confidence: 0.7},
		{code: "22001", actorID: strings.Repeat("a", 65), actorType: "COUNTRY", relation: "MENTIONS", confidence: 0.7},
		{code: "23514", actorID: "actor:bad-type", actorType: "ECONOMY", relation: "MENTIONS", confidence: 0.7},
		{code: "23514", actorID: "actor:bad-relation", actorType: "COUNTRY", relation: "OWNS", confidence: 0.7},
		{code: "23514", actorID: "actor:low-strength", actorType: "COUNTRY", relation: "MENTIONS", strength: -0.01, confidence: 0.7},
		{code: "23514", actorID: "actor:high-strength", actorType: "COUNTRY", relation: "MENTIONS", strength: 1.01, confidence: 0.7},
		{code: "23514", actorID: "actor:low-confidence", actorType: "COUNTRY", relation: "MENTIONS", confidence: -0.01},
		{code: "23514", actorID: "actor:high-confidence", actorType: "COUNTRY", relation: "MENTIONS", confidence: 1},
	} {
		assertPostgresCode(t, db, test.code, actorInsert, mustDomainID(t, coreid.EventActorLink), created.Event.ID,
			test.actorID, test.actorType, test.relation, test.strength, test.confidence)
	}
	assertPostgresCode(t, db, "23505", actorInsert, mustDomainID(t, coreid.EventActorLink), created.Event.ID,
		"actor:0", "COUNTRY", "MENTIONS", 0.5, 0.7)

	assetInsert := `INSERT INTO event_asset_links (id,event_id,asset_id,asset_type,impact_direction,impact_magnitude) VALUES ($1,$2,$3,$4,$5,$6)`
	for _, test := range []struct {
		code                          string
		assetID, assetType, direction string
		magnitude                     float64
	}{
		{code: "23514", assetID: " ", assetType: "SECURITY", direction: "POSITIVE"},
		{code: "22001", assetID: strings.Repeat("a", 65), assetType: "SECURITY", direction: "POSITIVE"},
		{code: "23514", assetID: "asset:bad-type", assetType: "ECONOMY", direction: "POSITIVE"},
		{code: "23514", assetID: "asset:bad-direction", assetType: "SECURITY", direction: "UP"},
		{code: "23514", assetID: "asset:low-magnitude", assetType: "SECURITY", direction: "POSITIVE", magnitude: -0.01},
		{code: "23514", assetID: "asset:high-magnitude", assetType: "SECURITY", direction: "POSITIVE", magnitude: 1.01},
	} {
		assertPostgresCode(t, db, test.code, assetInsert, mustDomainID(t, coreid.EventAssetLink), created.Event.ID,
			test.assetID, test.assetType, test.direction, test.magnitude)
	}
	assertPostgresCode(t, db, "23505", assetInsert, mustDomainID(t, coreid.EventAssetLink), created.Event.ID,
		"asset:0", "SECURITY", "POSITIVE", 0.5)
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

func validCreateInput(title, evidenceID string) eventbiz.CreateInput {
	return eventbiz.CreateInput{Title: title, Summary: title + " summary", Semantic: eventbiz.Semantic{}, Modality: eventbiz.ModalityFact,
		Evidence: []eventbiz.EvidenceLinkInput{{EvidenceID: evidenceID, ContributionWeight: 1}}}
}

func mustDomainID(t *testing.T, kind coreid.Kind) string {
	t.Helper()
	id, err := coreid.New(kind)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func validEventID(t *testing.T, id string) {
	t.Helper()
	if !coreid.Is(id, coreid.Event) {
		t.Fatalf("invalid Event ID %q", id)
	}
}
