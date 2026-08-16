package eventsemantic

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/eventsemantic"
	eventsemanticfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/eventsemantic"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestResearchSemanticDictionariesRejectMalformedPersistedDefinitions(t *testing.T) {
	value := eventbiz.ResearchSemanticDictionaries{
		VariableDefinitions: []eventbiz.ResearchVariableDefinition{{Key: "metric", Version: 1, NameZH: "指标", BusinessDefinition: "definition", Status: "invalid", AllowedDirections: []string{"increase"}}},
	}
	if err := validateResearchSemanticDictionaries(value); err == nil {
		t.Fatal("validateResearchSemanticDictionaries() error = nil")
	}
}

func TestPersistedEventSemanticReferencesUseDomainObjectIdentities(t *testing.T) {
	relation := eventbiz.EntityRelation{
		ID:           "ERL11111111-1111-4111-8111-111111111111",
		FromEntityID: "ENT22222222-2222-4222-8222-222222222222",
		ToEntityID:   "ENT33333333-3333-4333-8333-333333333333",
		Type:         "supplies", Status: "active",
	}
	if err := validatePersistedEntityRelation(relation); err != nil {
		t.Fatalf("canonical Entity Relation rejected: %v", err)
	}
	if !validPersistedEntityIDSet([]string{relation.FromEntityID, relation.ToEntityID}, true) {
		t.Fatal("canonical Entity ID set rejected")
	}
	impact := eventbiz.DirectImpactCandidate{
		Key: "impact", SourceSignalKey: "signal", TargetEntityID: relation.ToEntityID,
		AffectedVariableKey: "market_supply", AffectedVariableVersion: 1,
		AffectedDirection: "increase", DerivationType: "event_explicit",
		EntityRelationID: relation.ID,
		EvidenceIDs:      []string{"EEL44444444-4444-4444-8444-444444444444"},
	}
	if err := validatePersistedDirectImpactCandidate(impact); err != nil {
		t.Fatalf("canonical Direct Impact references rejected: %v", err)
	}
	relation.ID = "11111111-1111-4111-8111-111111111111"
	if err := validatePersistedEntityRelation(relation); err == nil {
		t.Fatal("bare UUID Entity Relation was accepted")
	}
}

func TestEventSemanticEligibilityAllowsUnknownOccurredAt(t *testing.T) {
	if strings.Contains(strings.ToLower(eventSemanticInputEligibilitySQL), "event_time is not null") {
		t.Fatal("Event Semantic eligibility still excludes Events with unknown occurred_at")
	}
}

func TestEventSemanticRoutePartitionsApplyStableDatabaseBudget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery("(?s)SELECT profile.entity_id::text, entity.name.*LIMIT \\$1").
		WithArgs(eventSemanticsRoutePartitionLimit).
		WillReturnRows(sqlmock.NewRows([]string{"entity_id", "name"}).
			AddRow("ENT11111111-1111-4111-8111-111111111111", "Industry"))
	if partitions, _, err := repository.eventSemanticIndustryPartitions(context.Background()); err != nil || len(partitions) != 1 {
		t.Fatalf("industry partitions = %v err = %v", partitions, err)
	}

	mock.ExpectQuery("(?s)SELECT DISTINCT concept_type.*LIMIT \\$1").
		WithArgs(eventSemanticsRoutePartitionLimit).
		WillReturnRows(sqlmock.NewRows([]string{"concept_type"}).AddRow("technology"))
	if partitions, _, err := repository.eventSemanticConceptPartitions(context.Background()); err != nil || len(partitions) != 1 {
		t.Fatalf("concept partitions = %v err = %v", partitions, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestEventSemanticManifestFingerprintCoversLeaseExpiry(t *testing.T) {
	manifest := eventbiz.ContextManifest{
		ContextLeaseID: "lease-1", AgentExecutionID: "execution-1", WorkerID: "worker-1",
		LeaseStatus: "active", LeaseExpiresAt: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ManifestContractVersion: eventSemanticsManifestVersion,
	}
	fingerprint, err := eventSemanticManifestFingerprint(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.ManifestFingerprint = fingerprint
	manifest.LeaseExpiresAt = manifest.LeaseExpiresAt.Add(time.Minute)
	mutated, err := eventSemanticManifestFingerprint(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if mutated == fingerprint {
		t.Fatal("lease expiry did not affect manifest fingerprint")
	}
}

func openEventPublicationTestDatabase(t *testing.T) *sql.DB {
	return openEventPublicationTestDatabaseAt(t, 0)
}

func openEventPublicationTestDatabaseAt(t *testing.T, version int64) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_event_semantic", migrationDir, version)
}

func TestHydrateSubmissionContextResolvesCountryWithoutShadowEntity(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO countries (id, code, name, name_en)
		VALUES ('COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b', 'CN', '中国', 'China')`); err != nil {
		t.Fatal(err)
	}
	for _, lockSelectedFacts := range []bool{false, true} {
		result, err := hydrateEventSemanticSubmissionContext(
			context.Background(), db, eventbiz.Context{}, eventbiz.Submission{
				EntityLinks: []eventbiz.EntityLinkCandidate{{
					Key: "country", Mention: "中国", EntityID: "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b", ProjectedEntityType: "country",
				}},
			}, lockSelectedFacts,
		)
		if err != nil {
			t.Fatalf("lockSelectedFacts=%v: %v", lockSelectedFacts, err)
		}
		if len(result.Entities) != 1 || result.Entities[0].ID != "COUc7cb6173-13d0-5ffe-b12d-fad8b49bed1b" ||
			result.Entities[0].Type != "country" || result.Entities[0].Aliases[0] != "China" {
			t.Fatalf("lockSelectedFacts=%v entities=%#v", lockSelectedFacts, result.Entities)
		}
	}
	var shadowRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE entity_type = 'country'`).Scan(&shadowRows); err != nil {
		t.Fatal(err)
	}
	if shadowRows != 0 {
		t.Fatalf("Country hydration created %d shadow Entity rows", shadowRows)
	}
}

func TestHydrateSubmissionContextResolvesRegionWithoutShadowEntity(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	const regionID = "REG7306e1ec-d56e-5381-aabc-103d702af12b"
	if _, err := db.ExecContext(context.Background(), `
		INSERT INTO regions (id, code, name, name_en, region_type)
		VALUES ($1, 'M49_015', '北非', 'Northern Africa', 'GEOGRAPHIC')`, regionID); err != nil {
		t.Fatal(err)
	}
	result, err := hydrateEventSemanticSubmissionContext(
		context.Background(), db, eventbiz.Context{}, eventbiz.Submission{
			EntityLinks: []eventbiz.EntityLinkCandidate{{
				Key: "region", Mention: "北非", EntityID: regionID, ProjectedEntityType: "region",
			}},
		}, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 1 || result.Entities[0].ID != regionID ||
		result.Entities[0].Type != "region" || result.Entities[0].Aliases[0] != "Northern Africa" {
		t.Fatalf("Region entities = %#v", result.Entities)
	}
	if err := validatePersistedEntity(result.Entities[0]); err != nil {
		t.Fatalf("hydrated Region rejected: %v", err)
	}
}

func TestHydrateSubmissionContextResolvesOrganizationWithoutLegacyEntity(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	if _, err := db.ExecContext(context.Background(), `INSERT INTO organization_categories(id,code,name_zh) VALUES('OCA7cf04802-4d04-5a8c-9a10-7d805cf29a4d','INTERGOVERNMENTAL','政府间国际组织'); INSERT INTO organization_functions(id,code,name_zh) VALUES('OFN1cd93122-d11c-5059-87aa-ddb8b2f2d25b','SECURITY','安全与防务'); INSERT INTO organizations(id,code,name,name_en,category_code,function_code) VALUES('ORG3fb9e7ff-2222-57fa-b306-c223ce3af549','UN','联合国','United Nations','INTERGOVERNMENTAL','SECURITY')`); err != nil {
		t.Fatal(err)
	}
	result, err := hydrateEventSemanticSubmissionContext(context.Background(), db, eventbiz.Context{}, eventbiz.Submission{EntityLinks: []eventbiz.EntityLinkCandidate{{Key: "organization", Mention: "联合国", EntityID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", ProjectedEntityType: "organization"}}}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entities) != 1 || result.Entities[0].ID != "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549" || result.Entities[0].Type != "organization" || result.Entities[0].Aliases[0] != "United Nations" {
		t.Fatalf("Organization entities = %#v", result.Entities)
	}
	var legacyRows int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM entity_nodes WHERE entity_type='alliance_org'`).Scan(&legacyRows); err != nil {
		t.Fatal(err)
	}
	if legacyRows != 0 {
		t.Fatalf("found %d legacy Alliance rows", legacyRows)
	}
}

func TestPersistedOrganizationIdentityRejectsLegacyAllianceUUID(t *testing.T) {
	valid := eventbiz.Entity{ID: "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549", Type: "organization", Name: "联合国", CanonicalName: "联合国", Aliases: []string{"United Nations"}, Status: "active"}
	if err := validatePersistedEntity(valid); err != nil {
		t.Fatalf("validatePersistedEntity(stable Organization) error = %v", err)
	}
	legacy := valid
	legacy.ID = "6f845f9f-10e2-44dd-b08a-e482e32d3558"
	if err := validatePersistedEntity(legacy); err == nil {
		t.Fatal("legacy Alliance UUID was accepted as an independent Organization")
	}
	legacy.ID = "ORG3fb9e7ff-2222-57fa-b306-c223ce3af549"
	legacy.Type = "alliance_org"
	if err := validatePersistedEntity(legacy); err == nil {
		t.Fatal("retired alliance_org type was accepted")
	}
}

func assertMeasurementEvidenceIDs(t *testing.T, db *sql.DB, measurementID, evidenceID string) {
	t.Helper()
	var evidenceIDs string
	if err := db.QueryRow(`
		SELECT evidence_ids::text FROM variable_signal_measurements WHERE id = $1
	`, measurementID).Scan(&evidenceIDs); err != nil {
		t.Fatal(err)
	}
	if evidenceIDs != "{"+evidenceID+"}" {
		t.Fatalf("measurement %s evidence_ids = %q", measurementID, evidenceIDs)
	}
}

func TestPostgresEventSemanticV2PersistsNarrativeMeasurementWithoutDirectImpact(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	service, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := service.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-v2-persistence",
		WorkerID: "semantic-v2-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	submission, err := service.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		lease.ID, "semantic-v2-persistence", "",
	))
	if err != nil {
		t.Fatal(err)
	}

	var text string
	var evidenceIDs string
	var role, shape, rawValue, rawUnit, canonicalValue, canonicalUnit sql.NullString
	if err := db.QueryRow(`
		SELECT measurement.raw_text, measurement.evidence_ids,
		       measurement.measurement_role, measurement.value_shape,
		       measurement.raw_value::text, measurement.raw_unit,
		       measurement.canonical_value::text, measurement.canonical_unit
		FROM variable_signal_measurements measurement
		JOIN variable_signals signal ON signal.id = measurement.variable_signal_id
		WHERE signal.semantic_submission_id = $1
	`, submission.SubmissionID).Scan(
		&text, &evidenceIDs, &role, &shape, &rawValue, &rawUnit, &canonicalValue, &canonicalUnit,
	); err != nil {
		t.Fatal(err)
	}
	if text != "production fell 10%" || evidenceIDs != "{"+eventsemanticfixture.EvidenceID+"}" ||
		role.Valid || shape.Valid || rawValue.Valid || rawUnit.Valid || canonicalValue.Valid || canonicalUnit.Valid {
		t.Fatalf("narrative measurement text=%q evidence=%v structured=%#v/%#v/%#v/%#v/%#v/%#v",
			text, evidenceIDs, role, shape, rawValue, rawUnit, canonicalValue, canonicalUnit)
	}
	var directImpacts int
	if err := db.QueryRow(`SELECT count(*) FROM direct_impact_assertions WHERE semantic_submission_id = $1`, submission.SubmissionID).Scan(&directImpacts); err != nil {
		t.Fatal(err)
	}
	if directImpacts != 0 {
		t.Fatalf("direct impacts = %d, want 0", directImpacts)
	}
	finalized, err := service.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID: submission.SubmissionID, ReviewerExecutionKey: "semantic-v2-persistence:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems("pass"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Status != eventbiz.StatusAccepted {
		t.Fatalf("finalized status = %q, want accepted", finalized.Status)
	}
	reanalysisLease, err := service.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, SupersedesSubmissionID: submission.SubmissionID,
		AgentExecutionID: "semantic-v2-reanalysis", WorkerID: "semantic-v2-fixture",
		Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	reanalysis, err := service.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		reanalysisLease.ID, "semantic-v2-reanalysis", submission.SubmissionID,
	))
	if err != nil {
		t.Fatal(err)
	}
	refinalized, err := service.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID: reanalysis.SubmissionID, ReviewerExecutionKey: "semantic-v2-reanalysis:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems("fail"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if refinalized.Status != eventbiz.StatusRejected {
		t.Fatalf("reanalysis status = %q, want rejected", refinalized.Status)
	}
	thirdLease, err := service.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, SupersedesSubmissionID: reanalysis.SubmissionID,
		AgentExecutionID: "semantic-v2-third", WorkerID: "semantic-v2-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	third, err := service.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		thirdLease.ID, "semantic-v2-third", reanalysis.SubmissionID,
	))
	if err != nil {
		t.Fatal(err)
	}
	thirdFinalized, err := service.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID: third.SubmissionID, ReviewerExecutionKey: "semantic-v2-third:reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer", Items: eventsemanticfixture.ReviewItems("pass"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if thirdFinalized.Status != eventbiz.StatusAccepted {
		t.Fatalf("third status = %q, want accepted", thirdFinalized.Status)
	}
	var firstStatus, secondStatus string
	if err := db.QueryRow(`SELECT status FROM event_semantic_submissions WHERE id = $1`, submission.SubmissionID).Scan(&firstStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM event_semantic_submissions WHERE id = $1`, reanalysis.SubmissionID).Scan(&secondStatus); err != nil {
		t.Fatal(err)
	}
	if firstStatus != "superseded" || secondStatus != "superseded" {
		t.Fatalf("ancestor statuses = %q/%q, want superseded/superseded", firstStatus, secondStatus)
	}
	indeterminateLease, err := service.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, SupersedesSubmissionID: third.SubmissionID,
		AgentExecutionID: "semantic-v2-indeterminate", WorkerID: "semantic-v2-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	indeterminate, err := service.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		indeterminateLease.ID, "semantic-v2-indeterminate", third.SubmissionID,
	))
	if err != nil {
		t.Fatal(err)
	}
	needsReanalysis, err := service.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID:         indeterminate.SubmissionID,
		ReviewerExecutionKey: "semantic-v2-indeterminate:reviewer",
		PromptHash:           strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems("indeterminate"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if needsReanalysis.Status != eventbiz.StatusNeedsReanalysis {
		t.Fatalf("first indeterminate status = %q", needsReanalysis.Status)
	}
	quarantined, err := service.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
		SubmissionID:         indeterminate.SubmissionID,
		ReviewerExecutionKey: "semantic-v2-indeterminate:adjudicator",
		PromptHash:           strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems("indeterminate"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if quarantined.Status != eventbiz.StatusQuarantined {
		t.Fatalf("second indeterminate status = %q, want quarantined", quarantined.Status)
	}
}

func TestPostgresEventSemanticAdapterRejectsCorruptedPersistedRows(t *testing.T) {
	tests := []struct {
		name    string
		corrupt func(*testing.T, *sql.DB, string)
	}{
		{
			name: "submission status",
			corrupt: func(t *testing.T, db *sql.DB, submissionID string) {
				t.Helper()
				if _, err := db.Exec(`ALTER TABLE event_semantic_submissions DROP CONSTRAINT chk_event_semantic_submission_status`); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`UPDATE event_semantic_submissions SET status = 'corrupted' WHERE id = $1`, submissionID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "decision candidate set",
			corrupt: func(t *testing.T, db *sql.DB, submissionID string) {
				t.Helper()
				if _, err := db.Exec(`
					UPDATE event_semantic_submissions
					SET decision_summary = jsonb_set(decision_summary, '{entity_links,0,candidate_key}', '"missing"')
					WHERE id = $1
				`, submissionID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "candidate record reference",
			corrupt: func(t *testing.T, db *sql.DB, submissionID string) {
				t.Helper()
				if _, err := db.Exec(`
					UPDATE event_entity_links SET candidate_key = 'drifted'
					WHERE semantic_submission_id = $1
				`, submissionID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "reviewer work package candidate content",
			corrupt: func(t *testing.T, db *sql.DB, submissionID string) {
				t.Helper()
				if _, err := db.Exec(`
					UPDATE event_semantic_submissions
					SET decision_summary = jsonb_set(
						decision_summary,
						'{reviewer_work_package,entity_links,0,entity_role}',
						'"drifted"'
					)
					WHERE id = $1
				`, submissionID); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "duplicate reviewer work package candidate",
			corrupt: func(t *testing.T, db *sql.DB, submissionID string) {
				t.Helper()
				if _, err := db.Exec(`
					UPDATE event_semantic_submissions
					SET decision_summary = jsonb_set(
						decision_summary,
						'{reviewer_work_package,entity_links}',
						(decision_summary #> '{reviewer_work_package,entity_links}') ||
						jsonb_build_array(decision_summary #> '{reviewer_work_package,entity_links,0}')
					)
					WHERE id = $1
				`, submissionID); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := openEventPublicationTestDatabase(t)
			eventsemanticfixture.SeedScenario(t, db, true)
			store, err := NewStore(db)
			if err != nil {
				t.Fatal(err)
			}
			useCase, err := eventbiz.NewUseCase(store)
			if err != nil {
				t.Fatal(err)
			}
			executionID := "corrupted-persisted-" + strings.ReplaceAll(test.name, " ", "-")
			lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
				EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
				WorkerID: "corruption-fixture", Lease: 15 * time.Minute,
			})
			if err != nil {
				t.Fatal(err)
			}
			submission, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
				lease.ID, executionID, "",
			))
			if err != nil {
				t.Fatal(err)
			}
			test.corrupt(t, db, submission.SubmissionID)
			if _, err := store.GetEventSemantics(context.Background(), eventsemanticfixture.EventID); err == nil {
				t.Fatal("corrupted persisted Event Semantic row reached Biz")
			}
			if _, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
				lease.ID, executionID, "",
			)); err == nil {
				t.Fatal("corrupted persisted Event Semantic row was replayed")
			}
			if _, err := useCase.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
				SubmissionID: submission.SubmissionID, ReviewerExecutionKey: executionID + ":reviewer",
				PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
				Items: eventsemanticfixture.ReviewItems(eventbiz.ReviewDecisionPass),
			}); err == nil {
				t.Fatal("review consumed a corrupted persisted Event Semantic aggregate")
			}
		})
	}
}

func TestPostgresEventSemanticTransactionReadsRejectCorruptedState(t *testing.T) {
	t.Run("Event enum", func(t *testing.T) {
		db := openEventPublicationTestDatabase(t)
		eventsemanticfixture.SeedScenario(t, db, true)
		store, err := NewStore(db)
		if err != nil {
			t.Fatal(err)
		}
		useCase, err := eventbiz.NewUseCase(store)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`ALTER TABLE events DROP CONSTRAINT chk_events_event_status`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE events SET event_status = 'broken' WHERE id = $1`, eventsemanticfixture.EventID); err != nil {
			t.Fatal(err)
		}
		_, err = useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
			EventID: eventsemanticfixture.EventID, AgentExecutionID: "corrupt-event-state",
			WorkerID: "corruption-fixture", Lease: 15 * time.Minute,
		})
		assertPersistedEventSemanticError(t, err)
	})

	t.Run("Context Lease enum", func(t *testing.T) {
		db := openEventPublicationTestDatabase(t)
		eventsemanticfixture.SeedScenario(t, db, true)
		store, err := NewStore(db)
		if err != nil {
			t.Fatal(err)
		}
		useCase, err := eventbiz.NewUseCase(store)
		if err != nil {
			t.Fatal(err)
		}
		lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
			EventID: eventsemanticfixture.EventID, AgentExecutionID: "corrupt-lease-state",
			WorkerID: "corruption-fixture", Lease: 15 * time.Minute,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`ALTER TABLE event_semantic_context_leases DROP CONSTRAINT chk_event_semantic_context_lease_status`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE event_semantic_context_leases SET status = 'broken' WHERE id = $1`, lease.ID); err != nil {
			t.Fatal(err)
		}
		_, err = useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
			lease.ID, "corrupt-lease-state", "",
		))
		assertPersistedEventSemanticError(t, err)
	})

	t.Run("Submission reference enum", func(t *testing.T) {
		db := openEventPublicationTestDatabase(t)
		eventsemanticfixture.SeedScenario(t, db, true)
		store, err := NewStore(db)
		if err != nil {
			t.Fatal(err)
		}
		useCase, err := eventbiz.NewUseCase(store)
		if err != nil {
			t.Fatal(err)
		}
		lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
			EventID: eventsemanticfixture.EventID, AgentExecutionID: "corrupt-reference-source",
			WorkerID: "corruption-fixture", Lease: 15 * time.Minute,
		})
		if err != nil {
			t.Fatal(err)
		}
		submission, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
			lease.ID, "corrupt-reference-source", "",
		))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`ALTER TABLE event_semantic_submissions DROP CONSTRAINT chk_event_semantic_submission_status`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE event_semantic_submissions SET status = 'broken' WHERE id = $1`, submission.SubmissionID); err != nil {
			t.Fatal(err)
		}
		_, err = useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
			EventID: eventsemanticfixture.EventID, SupersedesSubmissionID: submission.SubmissionID,
			AgentExecutionID: "corrupt-reference-target", WorkerID: "corruption-fixture", Lease: 15 * time.Minute,
		})
		assertPersistedEventSemanticError(t, err)
	})

	t.Run("Review retry range", func(t *testing.T) {
		db := openEventPublicationTestDatabase(t)
		eventsemanticfixture.SeedScenario(t, db, true)
		store, err := NewStore(db)
		if err != nil {
			t.Fatal(err)
		}
		useCase, err := eventbiz.NewUseCase(store)
		if err != nil {
			t.Fatal(err)
		}
		lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
			EventID: eventsemanticfixture.EventID, AgentExecutionID: "corrupt-retry-state",
			WorkerID: "corruption-fixture", Lease: 15 * time.Minute,
		})
		if err != nil {
			t.Fatal(err)
		}
		submission, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
			lease.ID, "corrupt-retry-state", "",
		))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`ALTER TABLE event_semantic_acceptance_policies DROP CONSTRAINT chk_event_semantic_policy_retry`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE event_semantic_acceptance_policies SET retry_budget = -1`); err != nil {
			t.Fatal(err)
		}
		_, err = useCase.SubmitReview(context.Background(), eventbiz.ReviewSubmission{
			SubmissionID: submission.SubmissionID, ReviewerExecutionKey: "corrupt-retry-state:reviewer",
			PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
			Items: eventsemanticfixture.ReviewItems(eventbiz.ReviewDecisionPass),
		})
		assertPersistedEventSemanticError(t, err)
	})
}

func assertPersistedEventSemanticError(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "persisted Event Semantic") {
		t.Fatalf("error = %v, want persisted Event Semantic boundary failure", err)
	}
}

func TestPostgresEventSemanticRejectedPrecheckRemainsReadableAndReplayable(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "semantic-rejected-precheck"
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
		WorkerID: "rejected-precheck-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := eventsemanticfixture.Submission(lease.ID, executionID, "")
	input.EntityLinks[0].EntityID = "ENT00000000-0000-4000-8000-000000000099"
	created, err := useCase.CreateSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != eventbiz.StatusRejected {
		t.Fatalf("created status = %q, want rejected", created.Status)
	}
	result, err := useCase.Get(context.Background(), eventsemanticfixture.EventID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Submissions) != 1 || result.Submissions[0].Status != eventbiz.StatusRejected {
		t.Fatalf("read submissions = %#v", result.Submissions)
	}
	replayed, err := useCase.CreateSubmission(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.Status != eventbiz.StatusRejected {
		t.Fatalf("replayed = %#v", replayed)
	}
}

func TestPostgresEventSemanticContextRejectsEvidenceHashDrift(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-evidence-hash-drift",
		WorkerID: "hash-drift-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE event_sources SET evidence_hash = $2 WHERE id = $1`,
		eventsemanticfixture.EvidenceID, strings.Repeat("f", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Context(context.Background(), lease.ID); err == nil {
		t.Fatal("Evidence hash drift reached Biz")
	}
}

func TestPostgresEventSemanticReviewReplayRejectsCorruptedSnapshot(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	const executionID = "semantic-corrupted-review-replay"
	lease, err := useCase.CreateContextLease(context.Background(), eventbiz.ContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: executionID,
		WorkerID: "review-replay-fixture", Lease: 15 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	submission, err := useCase.CreateSubmission(context.Background(), eventsemanticfixture.Submission(
		lease.ID, executionID, "",
	))
	if err != nil {
		t.Fatal(err)
	}
	review := eventbiz.ReviewSubmission{
		SubmissionID: submission.SubmissionID, ReviewerExecutionKey: executionID + ":reviewer",
		PromptHash: strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: eventsemanticfixture.ReviewItems(eventbiz.ReviewDecisionPass),
	}
	if _, err := useCase.SubmitReview(context.Background(), review); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		UPDATE event_semantic_review_snapshots
		SET payload = jsonb_set(payload, '{Items,0,Decision}', '"fail"')
		WHERE semantic_submission_id = $1 AND reviewer_execution_key = $2
	`, submission.SubmissionID, review.ReviewerExecutionKey); err != nil {
		t.Fatal(err)
	}
	if _, err := useCase.SubmitReview(context.Background(), review); err == nil {
		t.Fatal("corrupted Review Snapshot was replayed")
	}
}
