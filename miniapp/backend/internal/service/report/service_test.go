package report

import (
	"context"
	"testing"

	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/report"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
)

const testReportID = "RPT11111111-1111-4111-8111-111111111111"
const testScopeToken = "RPE11111111-1111-4111-8111-111111111111"

func TestServiceEvidenceUsesScopeToken(t *testing.T) {
	repository := &repositoryStub{evidence: biz.EvidenceCollection{ReportID: testReportID, ScopeToken: testScopeToken, Items: []biz.EvidenceItem{{Summary: "摘要", Keywords: []string{"关键词"}}}}}
	service, _ := NewService(biz.NewUseCase(repository))
	response, err := service.ListEvidences(context.Background(), &api.EvidenceRequest{ReportID: testReportID, ScopeToken: testScopeToken})
	if err != nil || response.ScopeToken != testScopeToken {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

type repositoryStub struct {
	listPage       biz.Page
	home           biz.HomeSnapshot
	chainPage      biz.IndustryChainPage
	layer          biz.LayerDetail
	chain          biz.IndustryChainDetail
	evidence       biz.EvidenceCollection
	lastChainQuery biz.ChainListQuery
}

func (r *repositoryStub) ListReports(context.Context, biz.ListQuery) (biz.Page, error) {
	return r.listPage, nil
}
func (r *repositoryStub) GetHome(context.Context, string) (biz.HomeSnapshot, error) {
	return r.home, nil
}
func (r *repositoryStub) ListIndustryChains(_ context.Context, query biz.ChainListQuery) (biz.IndustryChainPage, error) {
	r.lastChainQuery = query
	return r.chainPage, nil
}
func (r *repositoryStub) GetLayer(context.Context, string, string) (biz.LayerDetail, error) {
	return r.layer, nil
}
func (r *repositoryStub) GetIndustryChain(context.Context, string, string) (biz.IndustryChainDetail, error) {
	return r.chain, nil
}
func (r *repositoryStub) ListEvidences(context.Context, string, string) (biz.EvidenceCollection, error) {
	return r.evidence, nil
}

var _ biz.Repository = (*repositoryStub)(nil)

func (*repositoryStub) ListAnalyses(context.Context, biz.AnalysisQuery) (biz.AnalysisPage, error) {
	return biz.AnalysisPage{}, biz.ErrDataUnavailable
}

func (*repositoryStub) GetAnalysis(context.Context, biz.AnalysisQuery) (biz.NormalizedDetailProjection, error) {
	return biz.NormalizedDetailProjection{}, biz.ErrDataUnavailable
}

func (*repositoryStub) GetAnalysisChain(context.Context, biz.AnalysisQuery) (biz.NormalizedChain, error) {
	return biz.NormalizedChain{}, biz.ErrDataUnavailable
}
