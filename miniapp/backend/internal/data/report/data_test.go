package report

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
	dataapi "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
)

const testReportID = "RPT11111111-1111-4111-8111-111111111111"
const testScopeToken = "RPE11111111-1111-4111-8111-111111111111"

func TestRepositoryPassesPublishedCodesLabelsAndCursorWithoutTranslation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case reportsPath:
			if request.URL.Query().Get("cursor") != "report-cursor" {
				t.Fatalf("query=%v", request.URL.Query())
			}
			writeDataResult(t, writer, wirePage{Items: []wireSummary{summaryFixture()}})
		case reportsPath + "/" + testReportID + "/home":
			writeDataResult(t, writer, homeFixture())
		case reportsPath + "/" + testReportID + "/layers/geopolitics":
			writeDataResult(t, writer, layerFixture())
		case reportsPath + "/" + testReportID + "/industry-chains":
			if request.URL.Query().Get("cursor") != "chain-cursor" {
				t.Fatalf("query=%v", request.URL.Query())
			}
			writeDataResult(t, writer, wireChainPage{Items: []wireChainSummary{chainSummaryFixture()}})
		default:
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
	}))
	defer server.Close()
	repository := newTestRepository(t, server)
	if _, err := repository.ListReports(context.Background(), biz.ListQuery{Limit: 100, Cursor: "report-cursor"}); err != nil {
		t.Fatal(err)
	}
	home, err := repository.GetHome(context.Background(), testReportID)
	if err != nil || home.Geopolitics == nil || home.Geopolitics.Summary.Result.Code != "future_result" {
		t.Fatalf("home=%#v err=%v", home, err)
	}
	layer, err := repository.GetLayer(context.Background(), testReportID, "geopolitics")
	if err != nil || layer.Layer.Anchors[0].ConclusionBasis.Code != "direct_evidence" || layer.Layer.Confidence.Code != "future_confidence" {
		t.Fatalf("layer=%#v err=%v", layer, err)
	}
	page, err := repository.ListIndustryChains(context.Background(), biz.ChainListQuery{ReportID: testReportID, Limit: 20, Cursor: "chain-cursor"})
	if err != nil || page.Items[0].TimeWindow.Code != "future_window" {
		t.Fatalf("page=%#v err=%v", page, err)
	}
}

func TestRepositoryReadsIndustryChainAndOpaqueEvidenceScope(t *testing.T) {
	publishedAt := "2026-09-02T01:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case reportsPath + "/" + testReportID + "/industry-chains/chain-01":
			writeDataResult(t, writer, chainFixture())
		case reportsPath + "/" + testReportID + "/evidences":
			if request.URL.Query().Get("scope_token") != testScopeToken {
				t.Fatalf("query=%v", request.URL.Query())
			}
			writeDataResult(t, writer, wireEvidenceCollection{ReportID: testReportID, ScopeToken: testScopeToken, Items: []wireEvidenceItem{{PublishedAt: &publishedAt, Summary: "证据摘要", Keywords: []string{"关键词"}}}})
		default:
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
	}))
	defer server.Close()
	repository := newTestRepository(t, server)
	chain, err := repository.GetIndustryChain(context.Background(), testReportID, "chain-01")
	if err != nil || len(chain.IndustryChain.TopologyNodes) != 3 || chain.IndustryChain.Nodes[1].EvidenceScopeToken != nil || chain.IndustryChain.Nodes[1].ConclusionBasis.Code != "reasoning_hypothesis" {
		t.Fatalf("chain=%#v err=%v", chain, err)
	}
	evidence, err := repository.ListEvidences(context.Background(), testReportID, testScopeToken)
	if err != nil || evidence.Items[0].PublishedAt == nil {
		t.Fatalf("evidence=%#v err=%v", evidence, err)
	}
}

func TestRepositoryFailsClosedOnUnknownResponseField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"request_id":"data-request","result":{"items":[],"next_cursor":null,"unknown":true}}`))
	}))
	defer server.Close()
	_, err := newTestRepository(t, server).ListReports(context.Background(), biz.ListQuery{Limit: 100})
	if !errors.Is(err, biz.ErrDataUnavailable) {
		t.Fatalf("err=%v", err)
	}
}

func summaryFixture() wireSummary {
	return wireSummary{ID: testReportID, PublisherReportID: "publisher", GeneratedAt: "2026-09-02T00:00:00Z", HasGeopolitics: true, HasMacroeconomics: false, IndustryChainCount: 54, PublishedAt: "2026-09-02T01:00:00Z"}
}
func coded(code, label string) wireCodedLabel { return wireCodedLabel{Code: code, Label: label} }
func confidence() wireConfidence {
	return wireConfidence{Code: "future_confidence", Label: "未来置信度"}
}
func window() wireTimeWindow {
	return wireTimeWindow{Code: "future_window", Label: "未来时间窗口"}
}
func layerSummaryFixture() wireLayerSummary {
	return wireLayerSummary{Conclusion: "结论", Result: coded("future_result", "未来结果"), Confidence: confidence(), TimeWindow: window(), DownwardTransmission: []wireTransmission{}, Uncertainty: wireUncertainty{}, EvidenceScopeToken: stringPointer(testScopeToken)}
}
func homeFixture() wireHome {
	return wireHome{Report: summaryFixture(), Geopolitics: &wireLayerSnapshot{Key: "geopolitics", Title: "地缘政治", Summary: layerSummaryFixture()}}
}
func layerFixture() wireLayerDetail {
	return wireLayerDetail{Report: summaryFixture(), Layer: wireLayer{Key: "geopolitics", Title: "地缘政治", Summary: layerSummaryFixture(), AffectedAnchors: []wireAnchor{{LocalKey: "anchor-01", Name: "锚点", CurrentState: "UP", Result: coded("future_result", "未来结果"), ConclusionBasis: &wireCodedLabel{Code: "direct_evidence", Label: "直接证据"}, TransmissionLogic: "逻辑", TimeWindow: window(), Confidence: confidence(), EvidenceScopeToken: stringPointer(testScopeToken)}}, ReasoningSteps: []wireReasoningStep{}}}
}
func chainSummaryFixture() wireChainSummary {
	return wireChainSummary{LocalKey: "chain-01", Name: "产业链", Conclusion: "结论", Status: "已发布", Result: coded("future_result", "未来结果"), Confidence: confidence(), TimeWindow: window(), ImpactItems: []wireImpactSummary{}, EvidenceScopeToken: stringPointer(testScopeToken)}
}
func chainFixture() wireIndustryChainDetail {
	return wireIndustryChainDetail{Report: summaryFixture(), IndustryChain: wireIndustryChain{LocalKey: "chain-01", Name: "产业链", Conclusion: "结论", Status: "已发布", Result: coded("warming", "升温"), Confidence: confidence(), TimeWindow: window(), Path: "路径", Graph: wireGraph{Nodes: []wireGraphNode{{LocalKey: "node-01", Name: "节点一"}, {LocalKey: "node-02", Name: "节点二"}, {LocalKey: "node-03", Name: "结构上下文节点"}}, Edges: []wireGraphEdge{{FromNodeKey: "node-01", ToNodeKey: "node-03", Relation: coded("component", "组成")}}}, AffectedNodes: []wireChainNode{{LocalKey: "impact-01", NodeLocalKey: "node-01", Name: "节点一", Impact: "UP", Result: coded("warming", "升温"), ConclusionBasis: &wireCodedLabel{Code: "direct_evidence", Label: "直接证据"}, TransmissionLogic: "逻辑", TimeWindow: window(), Confidence: confidence(), EvidenceScopeToken: stringPointer(testScopeToken)}, {LocalKey: "impact-02", NodeLocalKey: "node-02", Name: "节点二", Impact: "UP", Result: coded("pending", "待验证"), ConclusionBasis: &wireCodedLabel{Code: "reasoning_hypothesis", Label: "推理假设"}, ValidationStatus: &wireCodedLabel{Code: "pending_validation", Label: "待验证"}, TransmissionLogic: "逻辑", TimeWindow: window(), Confidence: confidence()}}, CounterevidenceAndGap: "反证与缺口", StopCondition: "停止条件", EvidenceScopeToken: stringPointer(testScopeToken)}}
}
func stringPointer(value string) *string { return &value }

func newTestRepository(t *testing.T, server *httptest.Server) *Repository {
	t.Helper()
	client, err := dataapi.NewHTTPClient(dataapi.HTTPConfig{BaseURL: server.URL, ServiceToken: "miniapp-token", Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	repository, err := NewRepository(client)
	if err != nil {
		t.Fatal(err)
	}
	return repository
}
func writeDataResult(t *testing.T, writer http.ResponseWriter, result any) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(map[string]any{"request_id": "data-request", "result": result}); err != nil {
		t.Fatal(err)
	}
}
