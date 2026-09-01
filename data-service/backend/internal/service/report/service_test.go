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

func TestPublishReportMapsOffsetInputAndReturnsCanonicalUTCPublication(t *testing.T) {
	content := reportfixture.Content()
	publishedAt := time.Date(2026, 9, 1, 1, 2, 3, 456789000, time.FixedZone("CST", 8*60*60))
	useCase := &fakeUseCase{publishResult: reportbiz.PublicationResult{
		Record:      reportbiz.Record{ID: reportfixture.ReportOne, ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PublishedAt: publishedAt},
		ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}}
	service, err := NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.PublishReport(context.Background(), &reportapi.PublicationRequest{
		ContractVersion: reportapi.ContractVersion, SourceReportID: "agentos-report-2026-09-01-a",
		Content: apiContent(t, content),
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != v1.StatusCreated || response.Result.ContentHash != useCase.publishResult.ContentHash ||
		response.Result.PublishedAt != "2026-08-31T17:02:03.456789Z" {
		t.Fatalf("PublishReport() = %#v", response)
	}
	if useCase.publishContent.GeneratedAt.Format(time.RFC3339Nano) != "2026-09-01T08:30:00+08:00" ||
		useCase.publishSourceID != "agentos-report-2026-09-01-a" {
		t.Fatalf("mapped publication source=%q generated_at=%s", useCase.publishSourceID, useCase.publishContent.GeneratedAt.Format(time.RFC3339Nano))
	}

	useCase.publishResult.Replayed = true
	response, err = service.PublishReport(context.Background(), &reportapi.PublicationRequest{
		ContractVersion: reportapi.ContractVersion, SourceReportID: "agentos-report-2026-09-01-a", Content: apiContent(t, content),
	})
	if err != nil || response.Status != v1.StatusOK || !response.Result.Replayed {
		t.Fatalf("replay response=%#v error=%v", response, err)
	}
}

func TestReadModelsCanonicalizeAllReportTimestampsAndExposeExplicitImpactCounts(t *testing.T) {
	content := reportfixture.Content()
	publishedAt := time.Date(2026, 9, 1, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	summary := reportbiz.Summary{
		ID: reportfixture.ReportOne, SourceReportID: "agentos-report-2026-09-01-a",
		ReportType: content.ReportType, Title: content.Title, Status: content.Status, Simulation: content.Simulation,
		GeneratedAt: content.GeneratedAt, Timezone: content.Timezone, PublishedLayers: content.PublishedLayers,
		Statistics: content.Statistics, PublishedAt: publishedAt,
	}
	impactRef := content.ReportCards[0].ImpactItems[0].Ref
	useCase := &fakeUseCase{
		page: reportbiz.Page{Items: []reportbiz.Summary{summary}},
		home: reportbiz.Home{Report: summary, ReportCards: content.ReportCards, Company: content.Company,
			EvidenceCounts: map[reportbiz.TargetReference]int{impactRef: 2}},
	}
	service, err := NewService(useCase)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := service.ListReports(context.Background(), &reportapi.ListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	item := listed.Result.Items[0]
	if item.GeneratedAt != "2026-09-01T00:30:00Z" || item.PublishedAt != "2026-09-01T00:00:00Z" {
		t.Fatalf("canonical timestamps = generated %q published %q", item.GeneratedAt, item.PublishedAt)
	}
	home, err := service.GetReportHome(context.Background(), &reportapi.ReportRequest{ReportID: reportfixture.ReportOne})
	if err != nil {
		t.Fatal(err)
	}
	if home.Result.Report.GeneratedAt != "2026-09-01T00:30:00Z" ||
		home.Result.ReportCards[0].EvidenceCount != 1 || home.Result.ReportCards[0].ImpactItems[0].EvidenceCount != 2 {
		t.Fatalf("home read model = %#v", home.Result)
	}
}

func TestPublicationErrorsHaveStablePublicCodes(t *testing.T) {
	for _, test := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "conflict", err: reportbiz.ErrPublicationConflict, wantStatus: v1.StatusConflict, wantCode: reportapi.ErrorReportPublicationConflict},
		{name: "invalid", err: &reportbiz.ValidationError{Path: "content.title", Message: "invalid"}, wantStatus: v1.StatusUnprocessableEntity, wantCode: reportapi.ErrorInvalidRequest},
		{name: "evidence", err: &reportbiz.ReferenceError{Path: "content.evidence_refs", Reference: reportfixture.EvidenceOne, Message: "missing"}, wantStatus: v1.StatusUnprocessableEntity, wantCode: reportapi.ErrorReportEvidenceReferenceInvalid},
		{name: "repository", err: errors.New("database unavailable"), wantStatus: v1.StatusInternalServerError, wantCode: reportapi.ErrorReportRepositoryFailure},
	} {
		t.Run(test.name, func(t *testing.T) {
			useCase := &fakeUseCase{publishErr: test.err}
			service, err := NewService(useCase)
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.PublishReport(context.Background(), &reportapi.PublicationRequest{
				ContractVersion: reportapi.ContractVersion, SourceReportID: "agentos-report-2026-09-01-a",
				Content: apiContent(t, reportfixture.Content()),
			})
			var public *v1.PublicError
			if !errors.As(err, &public) || public.Status != test.wantStatus || public.Code != test.wantCode {
				t.Fatalf("PublishReport() error=%#v, want status=%d code=%s", err, test.wantStatus, test.wantCode)
			}
		})
	}
}

func TestEvidenceScopeNotFoundHasStableNotFoundCode(t *testing.T) {
	err := readError(reportbiz.ErrEvidenceScopeNotFound)
	var public *v1.PublicError
	if !errors.As(err, &public) || public.Status != v1.StatusNotFound || public.Code != reportapi.ErrorReportEvidenceScopeNotFound {
		t.Fatalf("readError()=%#v", err)
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

type fakeUseCase struct {
	publishResult   reportbiz.PublicationResult
	publishErr      error
	publishSourceID string
	publishContent  reportbiz.Content
	page            reportbiz.Page
	home            reportbiz.Home
}

func (f *fakeUseCase) Publish(_ context.Context, _ string, sourceID string, content reportbiz.Content) (reportbiz.PublicationResult, error) {
	f.publishSourceID, f.publishContent = sourceID, content
	return f.publishResult, f.publishErr
}

func (f *fakeUseCase) List(context.Context, reportbiz.ListRequest) (reportbiz.Page, error) {
	return f.page, nil
}

func (f *fakeUseCase) GetHome(context.Context, string) (reportbiz.Home, error) {
	return f.home, nil
}

func (*fakeUseCase) GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error) {
	return reportbiz.Summary{}, reportbiz.Layer{}, []reportbiz.IndustryChainSummary{}, nil
}

func (*fakeUseCase) GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChain, error) {
	return reportbiz.Summary{}, reportbiz.IndustryChain{}, nil
}

func (*fakeUseCase) ListEvidence(context.Context, string, reportbiz.ScopeType, string) ([]reportbiz.Evidence, error) {
	return []reportbiz.Evidence{}, nil
}

var _ UseCase = (*fakeUseCase)(nil)
