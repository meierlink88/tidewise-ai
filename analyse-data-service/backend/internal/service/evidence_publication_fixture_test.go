package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/evidencepublication"
)

func TestEvidencePublicationNeutralFixturesExerciseTwoPhaseRetryAndSafeFailure(t *testing.T) {
	store := newFixtureEvidenceStore()
	publication, err := evidencepublication.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	handler := dataServiceTestHandler(Dependencies{EvidencePublications: publication}, map[string]v1.Principal{
		"publisher-token": {
			Identity: "neutral-publisher",
			Scopes:   []string{ScopeRawEvidenceImport, ScopeEvidenceImport},
		},
	}, "request-evidence-fixture")
	rawPayload := readEvidenceFixture(t, "raw-evidence-publication.json")
	evidencePayload := readEvidenceFixture(t, "evidence-publication.json")

	firstRaw := postEvidenceFixture(t, handler, Namespace+"/raw-evidence-publications", rawPayload)
	secondRaw := postEvidenceFixture(t, handler, Namespace+"/raw-evidence-publications", rawPayload)
	if firstRaw.Status != http.StatusCreated || secondRaw.Status != http.StatusCreated ||
		firstRaw.RequestID != "request-evidence-fixture" || secondRaw.RequestID != "request-evidence-fixture" {
		t.Fatalf("Raw fixture responses first=%#v second=%#v", firstRaw, secondRaw)
	}
	var createdRaw, reusedRaw v1.RawEvidencePublicationResult
	decodeFixtureResult(t, firstRaw, &createdRaw)
	decodeFixtureResult(t, secondRaw, &reusedRaw)
	if createdRaw.RawEvidence.Disposition != "created" || reusedRaw.RawEvidence.Disposition != "reused" ||
		createdRaw.ReceiptID == reusedRaw.ReceiptID {
		t.Fatalf("Raw fixture dispositions created=%#v reused=%#v", createdRaw, reusedRaw)
	}

	firstEvidence := postEvidenceFixture(t, handler, Namespace+"/evidence-publications", evidencePayload)
	secondEvidence := postEvidenceFixture(t, handler, Namespace+"/evidence-publications", evidencePayload)
	var createdEvidence, reusedEvidence v1.EvidencePublicationResult
	decodeFixtureResult(t, firstEvidence, &createdEvidence)
	decodeFixtureResult(t, secondEvidence, &reusedEvidence)
	if firstEvidence.Status != http.StatusCreated || secondEvidence.Status != http.StatusCreated ||
		createdEvidence.Counts.EvidencesCreated != 2 || reusedEvidence.Counts.EvidencesReused != 2 ||
		createdEvidence.ReceiptID == reusedEvidence.ReceiptID {
		t.Fatalf("Evidence fixture results created=%#v reused=%#v", createdEvidence, reusedEvidence)
	}
	if len(createdEvidence.Evidences) != 2 || !createdEvidence.Evidences[0].IsSplit || !createdEvidence.Evidences[1].IsSplit {
		t.Fatalf("Evidence fixture split result = %#v", createdEvidence.Evidences)
	}

	driftPayload := bytes.Replace(evidencePayload, []byte("Example Corp expanded production."), []byte("Drifted fact."), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, Namespace+"/evidence-publications", driftPayload), http.StatusConflict, "EVIDENCE_PUBLICATION_CONFLICT")
	invalidLayer := bytes.Replace(evidencePayload, []byte(`"layer_type": "SINGLE"`), []byte(`"layer_type": "INVALID"`), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, Namespace+"/evidence-publications", invalidLayer), http.StatusBadRequest, "INVALID_REQUEST")
	invalidRaw := bytes.Replace(rawPayload, []byte(`"source_level": "L2_WIRE"`), []byte(`"source_level": "INVALID"`), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, Namespace+"/raw-evidence-publications", invalidRaw), http.StatusBadRequest, "INVALID_REQUEST")
	emptySet := []byte(`{"raw_evidence_id":"RAW_fixture_00000000000000000000","evidences":[]}`)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, Namespace+"/evidence-publications", emptySet), http.StatusUnprocessableEntity, "EVIDENCE_PUBLICATION_INVALID")
	missingRaw := bytes.Replace(evidencePayload, []byte("RAW_fixture_00000000000000000000"), []byte("RAW_missing_00000000000000000000"), 1)
	assertEvidenceFixtureError(t, postEvidenceFixture(t, handler, Namespace+"/evidence-publications", missingRaw), http.StatusUnprocessableEntity, "EVIDENCE_PUBLICATION_REFERENCE_INVALID")

	if len(store.raw) != 1 || len(store.evidences) != 2 || len(store.rawReceipts) != 2 || len(store.evidenceReceipts) != 2 {
		t.Fatalf("failed calls produced partial facts: raw=%d Evidence=%d Raw receipts=%d Evidence receipts=%d",
			len(store.raw), len(store.evidences), len(store.rawReceipts), len(store.evidenceReceipts))
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
	payload, err := os.ReadFile("../../api/data/v1/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

type fixtureEvidenceStore struct {
	raw              map[string]evidencepublication.StoredRawEvidence
	evidences        map[string]evidencepublication.StoredEvidence
	rawReceipts      []evidencepublication.RawEvidencePublicationReceipt
	evidenceReceipts []evidencepublication.EvidencePublicationReceipt
}

func newFixtureEvidenceStore() *fixtureEvidenceStore {
	return &fixtureEvidenceStore{
		raw:       make(map[string]evidencepublication.StoredRawEvidence),
		evidences: make(map[string]evidencepublication.StoredEvidence),
	}
}

func (s *fixtureEvidenceStore) InTransaction(ctx context.Context, fn func(evidencepublication.Transaction) error) error {
	copyStore := &fixtureEvidenceStore{
		raw:              copyRawEvidenceMap(s.raw),
		evidences:        copyEvidenceMap(s.evidences),
		rawReceipts:      append([]evidencepublication.RawEvidencePublicationReceipt(nil), s.rawReceipts...),
		evidenceReceipts: append([]evidencepublication.EvidencePublicationReceipt(nil), s.evidenceReceipts...),
	}
	if err := fn((*fixtureEvidenceTransaction)(copyStore)); err != nil {
		return err
	}
	s.raw, s.evidences = copyStore.raw, copyStore.evidences
	s.rawReceipts, s.evidenceReceipts = copyStore.rawReceipts, copyStore.evidenceReceipts
	return nil
}

type fixtureEvidenceTransaction fixtureEvidenceStore

func (*fixtureEvidenceTransaction) LockIdentities(context.Context, []string) error { return nil }
func (t *fixtureEvidenceTransaction) RawEvidence(_ context.Context, id string) (*evidencepublication.StoredRawEvidence, error) {
	record, ok := t.raw[id]
	if !ok {
		return nil, nil
	}
	return &record, nil
}
func (t *fixtureEvidenceTransaction) InsertRawEvidence(_ context.Context, record evidencepublication.StoredRawEvidence) error {
	t.raw[record.RawEvidenceID] = record
	return nil
}
func (t *fixtureEvidenceTransaction) EvidencesByRawEvidence(_ context.Context, rawID string) ([]evidencepublication.StoredEvidence, error) {
	result := make([]evidencepublication.StoredEvidence, 0)
	for _, record := range t.evidences {
		if record.RawEvidenceID == rawID {
			result = append(result, record)
		}
	}
	return result, nil
}
func (t *fixtureEvidenceTransaction) EvidencesByIDs(_ context.Context, ids []string) ([]evidencepublication.StoredEvidence, error) {
	result := make([]evidencepublication.StoredEvidence, 0, len(ids))
	for _, id := range ids {
		if record, ok := t.evidences[id]; ok {
			result = append(result, record)
		}
	}
	return result, nil
}
func (t *fixtureEvidenceTransaction) InsertEvidence(_ context.Context, record evidencepublication.StoredEvidence) error {
	t.evidences[record.EvidenceID] = record
	return nil
}
func (t *fixtureEvidenceTransaction) InsertRawEvidenceReceipt(_ context.Context, receipt evidencepublication.RawEvidencePublicationReceipt) error {
	t.rawReceipts = append(t.rawReceipts, receipt)
	return nil
}
func (t *fixtureEvidenceTransaction) InsertEvidenceReceipt(_ context.Context, receipt evidencepublication.EvidencePublicationReceipt) error {
	t.evidenceReceipts = append(t.evidenceReceipts, receipt)
	return nil
}

func copyRawEvidenceMap(source map[string]evidencepublication.StoredRawEvidence) map[string]evidencepublication.StoredRawEvidence {
	result := make(map[string]evidencepublication.StoredRawEvidence, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func copyEvidenceMap(source map[string]evidencepublication.StoredEvidence) map[string]evidencepublication.StoredEvidence {
	result := make(map[string]evidencepublication.StoredEvidence, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

var _ evidencepublication.Store = (*fixtureEvidenceStore)(nil)
