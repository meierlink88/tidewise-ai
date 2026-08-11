package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

type evidencePublicationHTTPStub struct {
	testDataHTTPServer
	rawRequest      *RawEvidencePublicationRequest
	evidenceRequest *EvidencePublicationRequest
}

func (s *evidencePublicationHTTPStub) PublishRawEvidence(_ context.Context, request *RawEvidencePublicationRequest) (*Response[RawEvidencePublicationResult], error) {
	s.rawRequest = request
	return &Response[RawEvidencePublicationResult]{Status: StatusCreated, Result: RawEvidencePublicationResult{
		ReceiptID: "11111111-1111-4111-8111-111111111111", ImportedAt: time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC),
		RawEvidence: RawEvidencePublicationItemResult{
			RawEvidenceID: request.RawEvidence.RawEvidenceID,
			ContentHash:   "1b46f625a140463536b92ffb1718d101bbcdfe09a76ef63089af6a0d99b8aa33",
			Keywords:      append([]string(nil), request.RawEvidence.Keywords...), Disposition: "created",
		},
	}}, nil
}

func (s *evidencePublicationHTTPStub) PublishEvidence(_ context.Context, request *EvidencePublicationRequest) (*Response[EvidencePublicationResult], error) {
	s.evidenceRequest = request
	return &Response[EvidencePublicationResult]{Status: StatusCreated, Result: EvidencePublicationResult{
		ReceiptID: "22222222-2222-4222-8222-222222222222", RawEvidenceID: request.RawEvidenceID,
		ImportedAt: time.Date(2026, 8, 11, 8, 1, 0, 0, time.UTC),
		Evidences: []EvidencePublicationItemResult{{
			EvidenceID: request.Evidences[0].EvidenceID, SplitOrder: 0, IsSplit: false, Disposition: "created",
		}}, Counts: EvidencePublicationCounts{EvidencesCreated: 1},
	}}, nil
}

func TestRawEvidencePublicationHTTPPreservesPublisherKeywords(t *testing.T) {
	stub := &evidencePublicationHTTPStub{}
	server := kratoshttp.NewServer()
	RegisterDataHTTPServer(server, stub)
	body := `{"raw_evidence":{"raw_evidence_id":"RAW_example_00000000000000000000","source_id":"SRC_example_00000000000000000000","source_name":"Example Wire","source_level":"L2_WIRE","source_url":"https://example.test/article/1","is_original":true,"raw_text":"Complete original article.","collected_at":"2026-08-11T01:05:00Z","keywords":[" AI芯片 ","供应链","AI芯片"]}}`
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, APIPrefix+"/raw-evidence-publications", strings.NewReader(body)))

	if response.Code != StatusCreated {
		t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
	}
	if stub.rawRequest == nil || !equalStrings(stub.rawRequest.RawEvidence.Keywords, []string{" AI芯片 ", "供应链", "AI芯片"}) {
		t.Fatalf("bound keywords = %#v", stub.rawRequest)
	}
	var result RawEvidencePublicationResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !equalStrings(result.RawEvidence.Keywords, []string{" AI芯片 ", "供应链", "AI芯片"}) {
		t.Fatalf("response keywords = %#v", result.RawEvidence.Keywords)
	}
}

func TestEvidencePublicationHTTPRejectsDuplicateAndUnknownFields(t *testing.T) {
	for _, body := range []string{
		`{"raw_evidence_id":"RAW_example_00000000000000000000","raw_evidence_id":"RAW_example_00000000000000000000","evidences":[]}`,
		`{"raw_evidence_id":"RAW_example_00000000000000000000","evidences":[],"group_id":"not-a-resource"}`,
		`{"raw_evidence_id":"RAW_example_00000000000000000000","evidences":[{"evidence_id":"EVD_example_00000000000000000000","split_order":null,"layer_type":"SINGLE","source_what":"fact","expression_fingerprint":"fact","expression_key":"fact-v1","fingerprint_version":"v1"}]}`,
	} {
		server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(func(response http.ResponseWriter, _ *http.Request, err error) {
			public, ok := err.(*PublicError)
			if !ok {
				t.Fatalf("error = %T %v", err, err)
			}
			response.WriteHeader(public.Status)
		}))
		RegisterDataHTTPServer(server, &evidencePublicationHTTPStub{})
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodPost, APIPrefix+"/evidence-publications", strings.NewReader(body)))
		if response.Code != StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", response.Code, response.Body.String())
		}
	}
}

func equalStrings(left, right []string) bool {
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
