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

func TestPublishReportMapsContractAndReturnsUTC(t *testing.T) {
	useCase := &fakeUseCase{publishResult: reportbiz.PublicationResult{Record: reportbiz.Record{
		ID: reportfixture.ReportOne, PublishedAt: time.Date(2026, 9, 2, 8, 0, 0, 0, time.FixedZone("CST", 8*3600)),
	}}}
	service, _ := NewService(useCase)
	response, err := service.PublishReport(context.Background(), &reportapi.PublicationRequest{
		PublisherReportID: "publisher-report", Report: apiReport(t, reportfixture.IndustryOnlyReport()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != v1.StatusCreated || response.Result.PublishedAt != "2026-09-02T00:00:00Z" || useCase.publisherID != "publisher-report" {
		t.Fatalf("response=%#v publisher=%s", response, useCase.publisherID)
	}
	if useCase.report.Geopolitics != nil || len(useCase.report.IndustryChains) != 1 {
		t.Fatalf("mapped Report=%#v", useCase.report)
	}
}

func TestListIndustryChainsReturnsCursorPage(t *testing.T) {
	useCase := &fakeUseCase{chainPage: reportbiz.IndustryChainPage{
		Items:      []reportbiz.IndustryChainSummary{{LocalKey: "chain-01", Ordinal: 1, Name: "产业链 01", ImpactItems: []reportbiz.IndustryChainImpactSummary{}}},
		NextCursor: stringPointer("next"),
	}}
	service, _ := NewService(useCase)
	response, err := service.ListReportIndustryChains(context.Background(), &reportapi.ChainListRequest{ReportID: reportfixture.ReportOne, Limit: "20"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Items) != 1 || response.Result.NextCursor == nil || *response.Result.NextCursor != "next" {
		t.Fatalf("response=%#v", response)
	}
}

func TestEvidenceReadUsesOpaqueScopeToken(t *testing.T) {
	useCase := &fakeUseCase{evidence: []reportbiz.Evidence{{Summary: "证据摘要", Keywords: []string{"关键词"}}}}
	service, _ := NewService(useCase)
	response, err := service.ListReportEvidence(context.Background(), &reportapi.EvidenceRequest{
		ReportID: reportfixture.ReportOne, ScopeToken: "RPE11111111-1111-4111-8111-111111111111",
	})
	if err != nil {
		t.Fatal(err)
	}
	if useCase.scopeToken != "RPE11111111-1111-4111-8111-111111111111" || response.Result.Items[0].Summary != "证据摘要" {
		t.Fatalf("response=%#v token=%s", response, useCase.scopeToken)
	}
}

func TestPublicationErrorsHaveStableCodes(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{reportbiz.ErrPublicationConflict, v1.StatusConflict, reportapi.ErrorReportPublicationConflict},
		{&reportbiz.ValidationError{Path: "report.generated_at", Message: "invalid"}, v1.StatusUnprocessableEntity, reportapi.ErrorInvalidRequest},
		{&reportbiz.ReferenceError{Path: "report.evidence_ids", Reference: reportfixture.EvidenceOne, Message: "missing"}, v1.StatusUnprocessableEntity, reportapi.ErrorReportEvidenceReferenceInvalid},
	} {
		useCase := &fakeUseCase{publishErr: test.err}
		service, _ := NewService(useCase)
		_, err := service.PublishReport(context.Background(), &reportapi.PublicationRequest{PublisherReportID: "publisher", Report: apiReport(t, reportfixture.Report())})
		var public *v1.PublicError
		if !errors.As(err, &public) || public.Status != test.status || public.Code != test.code {
			t.Fatalf("error=%#v", err)
		}
	}
}

func apiReport(t *testing.T, report reportbiz.Report) reportapi.Report {
	t.Helper()
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var result reportapi.Report
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
	report        reportbiz.Report
	page          reportbiz.Page
	home          reportbiz.Home
	chainPage     reportbiz.IndustryChainPage
	evidence      []reportbiz.Evidence
	scopeToken    string
}

func (f *fakeUseCase) Publish(_ context.Context, publisherID string, report reportbiz.Report) (reportbiz.PublicationResult, error) {
	f.publisherID, f.report = publisherID, report
	return f.publishResult, f.publishErr
}
func (f *fakeUseCase) List(context.Context, reportbiz.ListRequest) (reportbiz.Page, error) {
	return f.page, nil
}
func (f *fakeUseCase) GetHome(context.Context, string) (reportbiz.Home, error) { return f.home, nil }
func (*fakeUseCase) GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.LayerProjection, error) {
	return reportbiz.Summary{}, reportbiz.LayerProjection{}, nil
}
func (f *fakeUseCase) ListIndustryChains(context.Context, reportbiz.IndustryChainListRequest) (reportbiz.IndustryChainPage, error) {
	return f.chainPage, nil
}
func (*fakeUseCase) GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChainProjection, error) {
	return reportbiz.Summary{}, reportbiz.IndustryChainProjection{}, nil
}
func (f *fakeUseCase) ListEvidence(_ context.Context, _ string, scopeToken string) ([]reportbiz.Evidence, error) {
	f.scopeToken = scopeToken
	return f.evidence, nil
}

var _ UseCase = (*fakeUseCase)(nil)
