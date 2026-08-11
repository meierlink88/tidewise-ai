package eventsemantic

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	eventapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/event"
	eventsemanticapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/eventsemantic"
	evidenceapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/evidence"
	eventbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantic"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventsemantic"
	serverpkg "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/server"
	parentservice "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/service"
	eventsemanticfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/eventsemantic"
	postgresfixture "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/testsupport/postgres"
)

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

func applyEventPublicationMigration(t *testing.T, db *sql.DB, version int64) {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	postgresfixture.ApplyMigration(t, db, migrationDir, version)
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

func TestPostgresEventSemanticV2PersistsNarrativeMeasurementWithoutDirectImpact(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := eventsemanticdata.NewStore(db)
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

func TestPostgresEventSemanticHTTPFlowPreservesLeaseSubmissionReviewAndRead(t *testing.T) {
	db := openEventPublicationTestDatabase(t)
	eventsemanticfixture.SeedScenario(t, db, true)
	store, err := eventsemanticdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	useCase, err := eventbiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	application, err := NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	handler := newEventSemanticHTTPHandler(t, application)

	leaseRequest := eventsemanticapi.EventSemanticContextLeaseRequest{
		EventID: eventsemanticfixture.EventID, AgentExecutionID: "semantic-http-flow",
		WorkerID: "semantic-http-fixture", LeaseSeconds: 900,
	}
	var leaseEnvelope struct {
		Result eventsemanticapi.EventSemanticContextLease `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodPost, "/event-semantics/context-leases", leaseRequest, http.StatusCreated, &leaseEnvelope)
	if leaseEnvelope.Result.ContextLeaseID == "" || leaseEnvelope.Result.EventID != eventsemanticfixture.EventID {
		t.Fatalf("lease = %#v", leaseEnvelope.Result)
	}

	var contextEnvelope struct {
		Result eventsemanticapi.EventSemanticContext `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodGet,
		"/event-semantics/context-leases/"+leaseEnvelope.Result.ContextLeaseID+"/context",
		nil, http.StatusOK, &contextEnvelope)
	if contextEnvelope.Result.Event.ID != eventsemanticfixture.EventID || len(contextEnvelope.Result.Evidence) != 1 {
		t.Fatalf("context = %#v", contextEnvelope.Result)
	}

	submissionInput := eventsemanticfixture.Submission(leaseEnvelope.Result.ContextLeaseID, "semantic-http-flow", "")
	submissionRequest := eventSemanticSubmissionRequest(submissionInput)
	var submissionEnvelope struct {
		Result eventsemanticapi.EventSemanticSubmissionResult `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodPost, "/event-semantics/submissions", submissionRequest, http.StatusCreated, &submissionEnvelope)
	if submissionEnvelope.Result.Status != string(eventbiz.StatusPendingReview) {
		t.Fatalf("submission = %#v", submissionEnvelope.Result)
	}

	reviewItems := eventsemanticfixture.ReviewItems("pass")
	reviewRequest := eventsemanticapi.EventSemanticReviewRequest{
		ReviewerExecutionKey: "semantic-http-flow:reviewer",
		PromptHash:           strings.Repeat("b", 64), Model: "fixture-reviewer",
		Items: []eventsemanticapi.EventSemanticReviewItem{
			{CandidateType: reviewItems[0].CandidateType, CandidateKey: reviewItems[0].CandidateKey, Decision: reviewItems[0].Decision, ReasonCodes: reviewItems[0].ReasonCodes, EvidenceIDs: reviewItems[0].EvidenceIDs},
			{CandidateType: reviewItems[1].CandidateType, CandidateKey: reviewItems[1].CandidateKey, Decision: reviewItems[1].Decision, ReasonCodes: reviewItems[1].ReasonCodes, EvidenceIDs: reviewItems[1].EvidenceIDs},
		},
	}
	var reviewEnvelope struct {
		Result eventsemanticapi.EventSemanticSubmissionResult `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodPost,
		"/event-semantics/submissions/"+submissionEnvelope.Result.SubmissionID+"/reviews",
		reviewRequest, http.StatusOK, &reviewEnvelope)
	if reviewEnvelope.Result.Status != string(eventbiz.StatusAccepted) {
		t.Fatalf("review = %#v", reviewEnvelope.Result)
	}

	var resultEnvelope struct {
		Result eventsemanticapi.EventSemanticsResult `json:"result"`
	}
	semanticAPIRequest(t, handler, http.MethodGet, "/events/"+eventsemanticfixture.EventID+"/semantics", nil, http.StatusOK, &resultEnvelope)
	if len(resultEnvelope.Result.Submissions) != 1 || resultEnvelope.Result.Submissions[0].Status != string(eventbiz.StatusAccepted) {
		t.Fatalf("semantic result = %#v", resultEnvelope.Result)
	}
}

func eventSemanticSubmissionRequest(input eventbiz.Submission) eventsemanticapi.EventSemanticSubmissionRequest {
	request := eventsemanticapi.EventSemanticSubmissionRequest{
		ContextLeaseID: input.ContextLeaseID, EventID: input.EventID, AgentExecutionID: input.AgentExecutionID,
		AgentKey: input.AgentKey, AgentVersion: input.AgentVersion, SupersedesSubmissionID: input.SupersedesSubmissionID,
		GeneratorPromptHash: input.GeneratorPromptHash, GeneratorModel: input.GeneratorModel,
		ReviewerPromptHash: input.ReviewerPromptHash, ReviewerModel: input.ReviewerModel,
		AdjudicatorPromptHash: input.AdjudicatorPromptHash, AdjudicatorModel: input.AdjudicatorModel,
		OntologyVersion: input.OntologyVersion, AcceptancePolicyVersion: input.AcceptancePolicyVersion,
	}
	for _, link := range input.EntityLinks {
		request.EntityLinks = append(request.EntityLinks, eventsemanticapi.EventSemanticV3EntityLinkCandidate{
			CandidateKey: link.Key, Mention: link.Mention, EntityID: link.EntityID,
			ProjectedEntityType: link.ProjectedEntityType, EntityRole: link.EntityRole,
			EvidenceIDs: link.EvidenceIDs, ResolutionMethod: link.ResolutionMethod,
			ResolutionConfidence: link.ResolutionConfidence,
		})
	}
	for _, signal := range input.VariableSignals {
		item := eventsemanticapi.EventSemanticVariableSignalCandidate{
			CandidateKey: signal.Key, SubjectLinkKey: signal.SubjectLinkKey,
			VariableKey: signal.VariableKey, VariableVersion: signal.VariableVersion,
			Direction: signal.Direction, AssertionModality: signal.AssertionModality,
			EvidenceIDs: signal.EvidenceIDs, ExtractionConfidence: signal.ExtractionConfidence,
		}
		for _, measurement := range signal.Measurements {
			item.Measurements = append(item.Measurements, eventsemanticapi.EventSemanticMeasurement{
				MeasurementText: measurement.Text, EvidenceIDs: measurement.EvidenceIDs,
			})
		}
		request.VariableSignals = append(request.VariableSignals, item)
	}
	return request
}

func semanticAPIRequest(t *testing.T, handler http.Handler, method, path string, body any, wantStatus int, target any) {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, v1.APIPrefix+path, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer semantic-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", method, path, response.Code, wantStatus, response.Body)
	}
	if target != nil {
		if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
			t.Fatalf("decode %s %s: %v; body=%s", method, path, err, response.Body)
		}
	}
}

func newEventSemanticHTTPHandler(t *testing.T, application *Service) http.Handler {
	t.Helper()
	authenticator, err := serverpkg.NewAuthenticator([]serverpkg.Credential{{
		Secret: "semantic-token",
		Principal: v1.Principal{Identity: "semantic-publisher", Scopes: []string{
			serverpkg.ScopeEventSemanticsRead, serverpkg.ScopeEventSemanticsWrite,
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	httpServer, err := serverpkg.NewHTTPServer(
		conf.Config{App: conf.AppConfig{Env: conf.EnvLocal}, Server: conf.ServerConfig{Host: "127.0.0.1", Port: 18082, ReadTimeoutSeconds: 5, WriteTimeoutSeconds: 10}},
		parentservice.NewDataService(parentservice.Dependencies{}), semanticTestEventService{}, application,
		semanticTestEvidenceService{}, authenticator, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return httpServer.Server.Handler
}

type semanticTestEventService struct{}

func (semanticTestEventService) PublishReviewedEvents(context.Context, *eventapi.PublicationRequest) (*v1.Response[eventapi.PublicationResult], error) {
	return &v1.Response[eventapi.PublicationResult]{Status: http.StatusNoContent}, nil
}
func (semanticTestEventService) ListActiveEventTags(context.Context, *eventapi.TagCatalogRequest) (*v1.Response[eventapi.TagCatalog], error) {
	return &v1.Response[eventapi.TagCatalog]{Status: http.StatusNoContent}, nil
}
func (semanticTestEventService) ListEvents(context.Context, *eventapi.ListRequest) (*v1.Response[eventapi.Page], error) {
	return &v1.Response[eventapi.Page]{Status: http.StatusNoContent}, nil
}

type semanticTestEvidenceService struct{}

func (semanticTestEvidenceService) PublishRawEvidence(context.Context, *evidenceapi.RawEvidencePublicationRequest) (*v1.Response[evidenceapi.RawEvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.RawEvidencePublicationResult]{Status: http.StatusNoContent}, nil
}
func (semanticTestEvidenceService) PublishEvidence(context.Context, *evidenceapi.EvidencePublicationRequest) (*v1.Response[evidenceapi.EvidencePublicationResult], error) {
	return &v1.Response[evidenceapi.EvidencePublicationResult]{Status: http.StatusNoContent}, nil
}
