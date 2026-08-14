package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	evidenceservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/evidence"
	researchfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/research"
)

func TestEvidencePublicationNeutralFixturesExerciseTwoPhaseRetryAndSafeFailure(t *testing.T) {
	store := newFixtureEvidenceStore()
	publication, err := evidencebiz.NewUseCase(store)
	if err != nil {
		t.Fatal(err)
	}
	evidenceApplication, err := evidenceservice.NewService(publication)
	if err != nil {
		t.Fatal(err)
	}
	authenticator, err := NewAuthenticator([]Credential{{
		Secret: "publisher-token",
		Principal: v1.Principal{
			Identity: "neutral-publisher",
			Scopes:   []string{ScopeRawEvidenceImport, ScopeEvidenceImport},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	httpServer, err := NewHTTPServer(testConfig(), serverTestDataService{}, researchfixture.Service{}, serverTestEventService{}, serverTestEventSemanticService{}, evidenceApplication, serverTestRawDocumentService{}, authenticator, nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := httpServer.Server.Handler
	rawPayload := readEvidenceFixture(t, "raw-evidence-publication.json")
	evidencePayload := readEvidenceFixture(t, "evidence-publication.json")

	firstRaw := postEvidenceFixture(t, handler, v1.APIPrefix+"/raw-evidence-publications", rawPayload)
	secondRaw := postEvidenceFixture(t, handler, v1.APIPrefix+"/raw-evidence-publications", rawPayload)
	if firstRaw.Status != http.StatusCreated || secondRaw.Status != http.StatusCreated ||
		firstRaw.RequestID != "request-evidence-fixture" || secondRaw.RequestID != "request-evidence-fixture" {
		t.Fatalf("Raw fixture responses first=%#v second=%#v", firstRaw, secondRaw)
	}
	var firstRawResult, secondRawResult evidenceapi.RawEvidencePublicationResult
	decodeFixtureResult(t, firstRaw, &firstRawResult)
	decodeFixtureResult(t, secondRaw, &secondRawResult)
	if firstRawResult.RawEvidenceID != "RAW_fixture_00000000000000000000" || secondRawResult != firstRawResult {
		t.Fatalf("Raw fixture identities first=%#v second=%#v", firstRawResult, secondRawResult)
	}

	firstEvidence := postEvidenceFixture(t, handler, v1.APIPrefix+"/evidence-publications", evidencePayload)
	secondEvidence := postEvidenceFixture(t, handler, v1.APIPrefix+"/evidence-publications", evidencePayload)
	var firstEvidenceResult, secondEvidenceResult evidenceapi.EvidencePublicationResult
	decodeFixtureResult(t, firstEvidence, &firstEvidenceResult)
	decodeFixtureResult(t, secondEvidence, &secondEvidenceResult)
	if firstEvidence.Status != http.StatusCreated || secondEvidence.Status != http.StatusCreated ||
		firstEvidenceResult.RawEvidenceID != firstRawResult.RawEvidenceID ||
		!equalEvidenceFixtureIDs(firstEvidenceResult.EvidenceIDs, secondEvidenceResult.EvidenceIDs) {
		t.Fatalf("Evidence fixture results first=%#v second=%#v", firstEvidenceResult, secondEvidenceResult)
	}
	if len(firstEvidenceResult.EvidenceIDs) != 2 {
		t.Fatalf("Evidence fixture IDs = %#v", firstEvidenceResult.EvidenceIDs)
	}

	driftPayload := bytes.Replace(evidencePayload, []byte("Example Corp expanded production."), []byte("Drifted fact."), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, v1.APIPrefix+"/evidence-publications", driftPayload), http.StatusConflict, evidenceapi.ErrorEvidencePublicationConflict)
	invalidLayer := bytes.Replace(evidencePayload, []byte(`"layer_type": "SINGLE"`), []byte(`"layer_type": "INVALID"`), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, v1.APIPrefix+"/evidence-publications", invalidLayer), http.StatusBadRequest, evidenceapi.ErrorInvalidRequest)
	invalidRaw := bytes.Replace(rawPayload, []byte(`"source_level": "L2_WIRE"`), []byte(`"source_level": "INVALID"`), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, v1.APIPrefix+"/raw-evidence-publications", invalidRaw), http.StatusBadRequest, evidenceapi.ErrorInvalidRequest)
	emptySet := []byte(`{"raw_evidence_id":"RAW_fixture_00000000000000000000","evidences":[]}`)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, v1.APIPrefix+"/evidence-publications", emptySet), http.StatusUnprocessableEntity, evidenceapi.ErrorEvidencePublicationInvalid)
	missingRaw := bytes.Replace(evidencePayload, []byte("RAW_fixture_00000000000000000000"), []byte("RAW_missing_00000000000000000000"), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, v1.APIPrefix+"/evidence-publications", missingRaw), http.StatusUnprocessableEntity, evidenceapi.ErrorEvidencePublicationReferenceInvalid)

	if len(store.raw) != 1 || len(store.evidences) != 2 {
		t.Fatalf("failed calls produced partial facts: raw=%d Evidence=%d", len(store.raw), len(store.evidences))
	}
}

type evidenceFixtureHTTPResult struct {
	Status    int
	RequestID string
	Result    json.RawMessage
	ErrorCode string
}

func postEvidenceFixture(t *testing.T, handler http.Handler, path string, payload []byte) evidenceFixtureHTTPResult {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer publisher-token")
	request.Header.Set("X-Request-ID", "request-evidence-fixture")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var envelope struct {
		RequestID string          `json:"request_id"`
		Result    json.RawMessage `json:"result"`
		Error     struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode fixture response status %d: %v: %s", response.Code, err, response.Body.String())
	}
	return evidenceFixtureHTTPResult{Status: response.Code, RequestID: envelope.RequestID, Result: envelope.Result, ErrorCode: envelope.Error.Code}
}

func decodeFixtureResult(t *testing.T, response evidenceFixtureHTTPResult, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Result, target); err != nil {
		t.Fatalf("decode fixture result: %v", err)
	}
}

func assertEvidenceFixtureError(t *testing.T, response evidenceFixtureHTTPResult, status int, code string) {
	t.Helper()
	if response.Status != status || response.ErrorCode != code || response.RequestID != "request-evidence-fixture" {
		t.Fatalf("error response = %#v, want status=%d code=%s", response, status, code)
	}
}

func readEvidenceFixture(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := os.ReadFile("../../api/data/v1/evidence/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

type fixtureEvidenceStore struct {
	raw       map[string]evidencebiz.StoredRawEvidence
	evidences map[string]evidencebiz.StoredEvidence
}

func newFixtureEvidenceStore() *fixtureEvidenceStore {
	return &fixtureEvidenceStore{
		raw:       make(map[string]evidencebiz.StoredRawEvidence),
		evidences: make(map[string]evidencebiz.StoredEvidence),
	}
}

func (s *fixtureEvidenceStore) InTransaction(ctx context.Context, fn func(evidencebiz.Transaction) error) error {
	copyStore := &fixtureEvidenceStore{
		raw:       copyRawEvidenceMap(s.raw),
		evidences: copyEvidenceMap(s.evidences),
	}
	if err := fn((*fixtureEvidenceTransaction)(copyStore)); err != nil {
		return err
	}
	s.raw, s.evidences = copyStore.raw, copyStore.evidences
	return nil
}

type fixtureEvidenceTransaction fixtureEvidenceStore

func (*fixtureEvidenceTransaction) LockIdentities(context.Context, []string) error { return nil }
func (t *fixtureEvidenceTransaction) RawEvidence(_ context.Context, id string) (*evidencebiz.StoredRawEvidence, error) {
	record, ok := t.raw[id]
	if !ok {
		return nil, nil
	}
	return &record, nil
}
func (t *fixtureEvidenceTransaction) InsertRawEvidence(_ context.Context, record evidencebiz.StoredRawEvidence) error {
	t.raw[record.RawEvidenceID] = record
	return nil
}
func (t *fixtureEvidenceTransaction) EvidencesByRawEvidence(_ context.Context, rawID string) ([]evidencebiz.StoredEvidence, error) {
	result := make([]evidencebiz.StoredEvidence, 0)
	for _, record := range t.evidences {
		if record.RawEvidenceID == rawID {
			result = append(result, record)
		}
	}
	return result, nil
}
func (t *fixtureEvidenceTransaction) EvidencesByIDs(_ context.Context, ids []string) ([]evidencebiz.StoredEvidence, error) {
	result := make([]evidencebiz.StoredEvidence, 0, len(ids))
	for _, id := range ids {
		if record, ok := t.evidences[id]; ok {
			result = append(result, record)
		}
	}
	return result, nil
}
func (t *fixtureEvidenceTransaction) InsertEvidence(_ context.Context, record evidencebiz.StoredEvidence) error {
	t.evidences[record.EvidenceID] = record
	return nil
}

func equalEvidenceFixtureIDs(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func copyRawEvidenceMap(source map[string]evidencebiz.StoredRawEvidence) map[string]evidencebiz.StoredRawEvidence {
	result := make(map[string]evidencebiz.StoredRawEvidence, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func copyEvidenceMap(source map[string]evidencebiz.StoredEvidence) map[string]evidencebiz.StoredEvidence {
	result := make(map[string]evidencebiz.StoredEvidence, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

var _ evidencebiz.Store = (*fixtureEvidenceStore)(nil)
