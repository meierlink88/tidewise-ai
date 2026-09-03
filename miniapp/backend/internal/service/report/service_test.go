package report

import (
	"context"
	"testing"
	"time"

	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/report"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
)

const testReportID = "RPT11111111-1111-4111-8111-111111111111"
const testScopeToken = "RPE11111111-1111-4111-8111-111111111111"

func TestServicePassesCodeLabelAndCursorToAPI(t *testing.T) {
	summary := biz.Summary{ID: testReportID, PublisherReportID: "publisher", GeneratedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), PublishedAt: time.Date(2026, 9, 2, 1, 0, 0, 0, time.UTC), IndustryChainCount: 54}
	repository := &repositoryStub{listPage: biz.Page{Items: []biz.Summary{summary}}, home: biz.HomeSnapshot{Report: summary}, chainPage: biz.IndustryChainPage{Items: []biz.IndustryChainSummary{{LocalKey: "chain-01", Name: "产业链", Conclusion: "结论", Result: biz.CodedLabel{Code: "future", Label: "未来结果"}, Confidence: biz.Confidence{Code: "future", Label: "未来置信"}, TimeWindow: biz.TimeWindow{Code: "future", Label: "未来窗口"}, ImpactItems: []biz.IndustryChainImpactSummary{}}}}}
	service, _ := NewService(biz.NewUseCaseWithClock(repository, func() time.Time { return time.Date(2026, 9, 2, 9, 0, 0, 0, time.FixedZone("CST", 8*3600)) }))
	response, err := service.GetHome(context.Background(), &api.HomeRequest{})
	if err != nil || response.Reports[0].Report.IndustryChainCount != 54 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	page, err := service.ListIndustryChains(context.Background(), &api.IndustryChainListRequest{ReportID: testReportID, Limit: "20", Cursor: "cursor"})
	if err != nil || page.Items[0].Result.Code != "future" || repository.lastChainQuery.Cursor != "cursor" {
		t.Fatalf("page=%#v err=%v query=%#v", page, err, repository.lastChainQuery)
	}
}

func TestServiceEvidenceUsesScopeToken(t *testing.T) {
	repository := &repositoryStub{evidence: biz.EvidenceCollection{ReportID: testReportID, ScopeToken: testScopeToken, Items: []biz.EvidenceItem{{Summary: "摘要", Keywords: []string{"关键词"}}}}}
	service, _ := NewService(biz.NewUseCase(repository))
	response, err := service.ListEvidences(context.Background(), &api.EvidenceRequest{ReportID: testReportID, ScopeToken: testScopeToken})
	if err != nil || response.ScopeToken != testScopeToken {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestServicePreservesIndustryChainTopologyNodesSeparatelyFromAssessments(t *testing.T) {
	summary := biz.Summary{ID: testReportID, PublisherReportID: "publisher", GeneratedAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC), PublishedAt: time.Date(2026, 9, 2, 1, 0, 0, 0, time.UTC), IndustryChainCount: 1}
	repository := &repositoryStub{chain: biz.IndustryChainDetail{Report: summary, IndustryChain: biz.IndustryChain{
		LocalKey: "chain-01", TopologyNodes: []biz.IndustryChainTopologyNode{{LocalKey: "node-01", Name: "节点一"}, {LocalKey: "node-02", Name: "结构上下文节点"}},
		Nodes: []biz.IndustryChainNode{{LocalKey: "node-01", Name: "节点一"}},
	}}}
	service, _ := NewService(biz.NewUseCase(repository))
	response, err := service.GetIndustryChain(context.Background(), &api.IndustryChainRequest{ReportID: testReportID, ChainKey: "chain-01"})
	if err != nil || len(response.IndustryChain.TopologyNodes) != 2 || len(response.IndustryChain.Nodes) != 1 {
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
