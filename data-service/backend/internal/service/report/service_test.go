package report

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/report"
	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	reportfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/report"
)

func TestPublishReportMapsV2AndReturnsUTC(t *testing.T) {
	useCase := &fakeUseCase{publishResult: reportbiz.PublicationResult{Record: reportbiz.Record{ID: reportfixture.ReportOne, PublishedAt: time.Date(2026, 9, 2, 8, 0, 0, 0, time.FixedZone("CST", 8*3600))}, ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	service, _ := NewService(useCase)
	response, err := service.PublishReport(context.Background(), &reportapi.PublicationRequest{ContractVersion: reportapi.ContractVersion, PublisherReportID: "publisher-report", Content: apiContent(t, reportfixture.IndustryOnlyContent())})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != v1.StatusCreated || response.Result.PublishedAt != "2026-09-02T00:00:00Z" || useCase.publisherID != "publisher-report" {
		t.Fatalf("response=%#v publisher=%s", response, useCase.publisherID)
	}
	if useCase.content.Geopolitics != nil || len(useCase.content.IndustryChains) != 1 {
		t.Fatalf("mapped content=%#v", useCase.content)
	}
}

func TestListIndustryChainsReturnsCursorPage(t *testing.T) {
	useCase := &fakeUseCase{chainPage: reportbiz.IndustryChainPage{Items: []reportbiz.IndustryChainSummary{{Key: "chain-01", DisplayOrder: 1}}, NextCursor: stringPointer("next")}}
	service, _ := NewService(useCase)
	response, err := service.ListReportIndustryChains(context.Background(), &reportapi.ChainListRequest{ReportID: reportfixture.ReportOne, Limit: "20"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Items) != 1 || response.Result.NextCursor == nil || *response.Result.NextCursor != "next" {
		t.Fatalf("response=%#v", response)
	}
}

func TestPublicationErrorsHaveStableCodes(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{{reportbiz.ErrPublicationConflict, v1.StatusConflict, reportapi.ErrorReportPublicationConflict}, {&reportbiz.ValidationError{Path: "content.title", Message: "invalid"}, v1.StatusUnprocessableEntity, reportapi.ErrorInvalidRequest}, {&reportbiz.ReferenceError{Path: "content.evidence_refs", Reference: reportfixture.EvidenceOne, Message: "missing"}, v1.StatusUnprocessableEntity, reportapi.ErrorReportEvidenceReferenceInvalid}} {
		useCase := &fakeUseCase{publishErr: test.err}
		service, _ := NewService(useCase)
		_, err := service.PublishReport(context.Background(), &reportapi.PublicationRequest{ContractVersion: reportapi.ContractVersion, PublisherReportID: "publisher", Content: apiContent(t, reportfixture.Content())})
		var public *v1.PublicError
		if !errors.As(err, &public) || public.Status != test.status || public.Code != test.code {
			t.Fatalf("error=%#v", err)
		}
	}
}

func apiContent(t *testing.T, content reportbiz.Content) reportapi.Content {
	t.Helper()
	payload, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	var result reportapi.Content
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func stringPointer(value string) *string { return &value }

type fakeUseCase struct {
	publishResult reportbiz.PublicationResult
	publishErr    error
	publisherID   string
	content       reportbiz.Content
	page          reportbiz.Page
	home          reportbiz.Home
	chainPage     reportbiz.IndustryChainPage
}

func (f *fakeUseCase) Publish(_ context.Context, _ string, publisherID string, content reportbiz.Content) (reportbiz.PublicationResult, error) {
	f.publisherID, f.content = publisherID, content
	return f.publishResult, f.publishErr
}
func (f *fakeUseCase) List(context.Context, reportbiz.ListRequest) (reportbiz.Page, error) {
	return f.page, nil
}
func (f *fakeUseCase) GetHome(context.Context, string) (reportbiz.Home, error) { return f.home, nil }
func (*fakeUseCase) GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error) {
	return reportbiz.Summary{}, reportbiz.Layer{}, nil, nil
}
func (f *fakeUseCase) ListIndustryChains(context.Context, reportbiz.IndustryChainListRequest) (reportbiz.IndustryChainPage, error) {
	return f.chainPage, nil
}
func (*fakeUseCase) GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChain, error) {
	return reportbiz.Summary{}, reportbiz.IndustryChain{}, nil
}
func (*fakeUseCase) ListEvidence(context.Context, string, reportbiz.ScopeType, string) ([]reportbiz.Evidence, error) {
	return []reportbiz.Evidence{}, nil
}

var _ UseCase = (*fakeUseCase)(nil)
