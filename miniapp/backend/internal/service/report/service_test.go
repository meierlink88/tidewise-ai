package report

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/report"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
)

const serviceTestReportID = "RPT11111111-1111-4111-8111-111111111111"

type repositoryStub struct {
	layerCalls    int
	chainCalls    int
	evidenceCalls int
	layerError    error
	evidence      biz.EvidenceCollection
}

func (*repositoryStub) ListReports(context.Context, biz.ListQuery) (biz.Page, error) {
	return biz.Page{Items: []biz.Summary{}}, nil
}

func (*repositoryStub) GetHome(context.Context, string) (biz.Home, error) {
	return biz.Home{}, nil
}

func (r *repositoryStub) GetLayer(context.Context, string, string) (biz.LayerDetail, error) {
	r.layerCalls++
	return biz.LayerDetail{}, r.layerError
}

func (r *repositoryStub) GetIndustryChain(context.Context, string, string) (biz.IndustryChainDetail, error) {
	r.chainCalls++
	return biz.IndustryChainDetail{}, nil
}

func (r *repositoryStub) ListEvidences(context.Context, string, biz.EvidenceScope) (biz.EvidenceCollection, error) {
	r.evidenceCalls++
	return r.evidence, nil
}

func TestServiceRejectsInvalidDetailAndEvidenceInputsBeforeData(t *testing.T) {
	repository := &repositoryStub{}
	service, err := NewService(biz.NewUseCase(repository))
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*api.LayerRequest{
		{ReportID: "RPT-invalid", LayerKey: biz.LayerGeopolitics},
		{ReportID: serviceTestReportID, LayerKey: "company"},
	} {
		if _, callErr := service.GetLayer(context.Background(), request); !errors.Is(callErr, v1.ErrInvalidRequest) {
			t.Fatalf("layer error = %v", callErr)
		}
	}
	if _, callErr := service.GetIndustryChain(context.Background(), &api.IndustryChainRequest{
		ReportID: serviceTestReportID, ChainKey: "Chain-21",
	}); !errors.Is(callErr, v1.ErrInvalidRequest) {
		t.Fatalf("chain error = %v", callErr)
	}
	for _, request := range []*api.EvidenceRequest{
		{ReportID: serviceTestReportID, ScopeType: biz.ScopeAnchor, ScopeKey: "layer/anchor"},
		{ReportID: serviceTestReportID, ScopeType: "event", ScopeKey: "event-1"},
		{ReportID: serviceTestReportID, ScopeType: biz.ScopeReportCard, ScopeKey: "geo-card", HasUnknownQuery: true},
	} {
		if _, callErr := service.ListEvidences(context.Background(), request); !errors.Is(callErr, v1.ErrInvalidRequest) {
			t.Fatalf("evidence error = %v", callErr)
		}
	}
	if repository.layerCalls != 0 || repository.chainCalls != 0 || repository.evidenceCalls != 0 {
		t.Fatalf("repository calls = layer:%d chain:%d evidence:%d", repository.layerCalls, repository.chainCalls, repository.evidenceCalls)
	}
}

func TestServiceMapsStableErrorsAndEvidenceSlimDTO(t *testing.T) {
	repository := &repositoryStub{layerError: biz.ErrLayerNotFound}
	service, err := NewService(biz.NewUseCase(repository))
	if err != nil {
		t.Fatal(err)
	}
	if _, callErr := service.GetLayer(context.Background(), &api.LayerRequest{
		ReportID: serviceTestReportID, LayerKey: biz.LayerGeopolitics,
	}); !errors.Is(callErr, v1.ErrReportLayerNotFound) {
		t.Fatalf("layer error = %v", callErr)
	}

	publishedAt := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	repository.evidence = biz.EvidenceCollection{ReportID: serviceTestReportID,
		Scope: biz.EvidenceScope{Type: biz.ScopeReportCard, Key: "geo-card"}, Items: []biz.EvidenceItem{{
			PublishedAt: &publishedAt, Summary: "显式关联证据", Keywords: []string{"供应链"},
		}}}
	result, callErr := service.ListEvidences(context.Background(), &api.EvidenceRequest{
		ReportID: serviceTestReportID, ScopeType: biz.ScopeReportCard, ScopeKey: "geo-card",
	})
	if callErr != nil {
		t.Fatal(callErr)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"evidence_id", "event_id", "display_order", `"role"`, "source_type", "evidence_count"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("Evidence DTO leaked %q: %s", forbidden, payload)
		}
	}
	if len(result.Items) != 1 || result.Items[0].PublishedAt == nil || *result.Items[0].PublishedAt != "2026-09-01T03:00:00Z" {
		t.Fatalf("result = %#v", result)
	}
}
