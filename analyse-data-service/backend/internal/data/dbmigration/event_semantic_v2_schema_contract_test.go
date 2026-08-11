package dbmigration

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	eventsemanticfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/eventsemantic"
)

func TestEventSemanticV2MigrationMakesMeasurementNarrativeOnlyForNewWrites(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "000036_event_semantic_v2.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, fragment := range []string{
		"drop constraint chk_variable_signal_measurement_values",
		"drop constraint chk_variable_signal_measurement_change_unit",
		"drop constraint chk_variable_signal_measurement_units",
		"drop constraint chk_variable_signal_measurement_conversion",
		"drop constraint chk_variable_signal_measurement_raw_range",
		"drop constraint chk_variable_signal_measurement_canonical_range",
		"alter column measurement_role drop not null",
		"alter column value_shape drop not null",
		"alter column raw_unit drop not null",
		"alter column canonical_unit drop not null",
		"add column evidence_ids uuid[]",
		"set evidence_ids = array[evidence_id]",
		"create function event_semantic_measurement_evidence_ids_compat()",
		"create trigger trg_event_semantic_measurement_evidence_ids_compat",
		"new.evidence_ids := array[new.evidence_id]",
		"alter column evidence_ids set not null",
		"add column allowed_event_roles",
		"add column allowed_units",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("Event Semantic V2 migration missing %q", fragment)
		}
	}
	for _, forbidden := range []string{
		"drop table direct_impact_assertions",
		"delete from direct_impact_assertions",
		"drop table variable_signal_measurements",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("Event Semantic V2 migration contains destructive fragment %q", forbidden)
		}
	}
}

func TestEventSemanticV2MigrationBackfillsDeprecatedEntityTypeRoles(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 35)
	if _, err := db.Exec(`
		INSERT INTO entity_type_definitions(
			type_key, version, signal_subject_allowed, direct_target_mode, status
		) VALUES ('deprecated_fixture', 1, false, 'deny', 'deprecated')
	`); err != nil {
		t.Fatal(err)
	}
	applyEventPublicationMigration(t, db, 36)
	var roleCount int
	if err := db.QueryRow(`
		SELECT cardinality(allowed_event_roles) FROM entity_type_definitions
		WHERE type_key = 'deprecated_fixture' AND version = 1
	`).Scan(&roleCount); err != nil {
		t.Fatal(err)
	}
	if roleCount == 0 {
		t.Fatal("deprecated Entity Type received no Event roles during V2 migration")
	}
}

func TestEventSemanticV2MigrationBackfillsAndKeepsLegacyMeasurementWritesSafe(t *testing.T) {
	db := openEventPublicationTestDatabaseAt(t, 35)
	eventsemanticfixture.SeedScenario(t, db, false)
	const (
		leaseID       = "40000000-0000-4000-8000-000000000001"
		submissionID  = "40000000-0000-4000-8000-000000000002"
		linkID        = "40000000-0000-4000-8000-000000000003"
		signalID      = "40000000-0000-4000-8000-000000000004"
		historicalID  = "40000000-0000-4000-8000-000000000005"
		legacyWriteID = "40000000-0000-4000-8000-000000000006"
	)
	if _, err := db.Exec(`
		INSERT INTO event_semantic_context_leases(
			id,event_id,agent_execution_id,worker_id,status,lease_expires_at,context_snapshot,consumed_at
		) VALUES ($1,$2,'migration-compat','migration-test','consumed',now() + interval '1 hour','{}',now())
	`, leaseID, eventsemanticfixture.EventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO event_semantic_submissions(
			id,context_lease_id,event_id,agent_execution_id,agent_key,agent_version,
			generator_prompt_hash,generator_model,reviewer_prompt_hash,reviewer_model,
			ontology_version,acceptance_policy_key,acceptance_policy_version,
			canonical_payload_hash,status,candidate_counts,decision_summary,finalized_at
		) VALUES (
			$1,$2,$3,'migration-compat','event-semantic-enricher','legacy-version',
			$4,'legacy-generator',$5,'legacy-reviewer','legacy-ontology',
			'event-semantics.phase-one',1,$6,'accepted','{}','{}',now()
		)
	`, submissionID, leaseID, eventsemanticfixture.EventID, strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO event_entity_links(
			id,event_id,entity_id,entity_role,assign_source,review_status,evidence_note,
			semantic_submission_id,candidate_key,resolved_mention,resolution_method,
			resolution_confidence,evidence_ids,provenance
		) VALUES ($1,$2,$3,'event_subject','ai','accepted','',$4,'company','Integration Wafer Fab',
			'qdrant_exact',1.0,ARRAY[$5::uuid],'semantic')
	`, linkID, eventsemanticfixture.EventID, eventsemanticfixture.CompanyID, submissionID, eventsemanticfixture.EvidenceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO variable_signals(
			id,semantic_submission_id,candidate_key,source_event_id,subject_event_entity_link_id,
			variable_key,variable_version,direction,assertion_modality,evidence_ids,review_status
		) VALUES ($1,$2,'production',$3,$4,'production_volume',1,'decrease','actual',ARRAY[$5::uuid],'accepted')
	`, signalID, submissionID, eventsemanticfixture.EventID, linkID, eventsemanticfixture.EvidenceID); err != nil {
		t.Fatal(err)
	}
	insertLegacyMeasurement := func(id string) {
		t.Helper()
		if _, err := db.Exec(`
			INSERT INTO variable_signal_measurements(
				id,variable_signal_id,measurement_role,value_shape,raw_value,raw_unit,
				canonical_value,canonical_unit,raw_text,is_approximate,evidence_id
			) VALUES ($1,$2,'relative_change','exact',10,'%',10,'percent','production fell 10%',false,$3)
		`, id, signalID, eventsemanticfixture.EvidenceID); err != nil {
			t.Fatal(err)
		}
	}
	insertLegacyMeasurement(historicalID)

	applyEventPublicationMigration(t, db, 36)
	assertMeasurementEvidenceIDs(t, db, historicalID, eventsemanticfixture.EvidenceID)
	insertLegacyMeasurement(legacyWriteID)
	assertMeasurementEvidenceIDs(t, db, legacyWriteID, eventsemanticfixture.EvidenceID)
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
