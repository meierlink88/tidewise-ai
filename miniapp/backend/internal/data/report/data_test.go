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
	if err != nil || home.Geopolitics == nil || home.Macroeconomics == nil || home.Geopolitics.Summary.Result.Code != "future_result" || len(home.Geopolitics.Summary.Transmissions) != 2 || home.Geopolitics.Summary.Transmissions[0].LocalKey != "geo-macro" || home.Geopolitics.Summary.Transmissions[1].LocalKey != "geo-chain" || len(home.Macroeconomics.Summary.Transmissions) != 1 || home.Macroeconomics.Summary.Transmissions[0].LocalKey != "macro-chain" {
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
	return wireLayerSummary{Conclusion: "结论", Result: coded("future_result", "未来结果"), Confidence: confidence(), TimeWindow: window(), DownwardTransmission: wireDownwardTransmission{ToMacroeconomics: &wireTransmissionGroup{Summary: "宏观传导", Paths: []wireTransmission{}}, ToIndustryChains: &wireTransmissionGroup{Summary: "产业链传导", Paths: []wireTransmission{}}}, Uncertainty: wireUncertainty{}, EvidenceScopeToken: stringPointer(testScopeToken)}
}
func homeFixture() wireHome {
	report := summaryFixture()
	report.HasMacroeconomics = true
	geopolitics := layerSummaryFixture()
	geopolitics.DownwardTransmission.ToMacroeconomics.Paths = []wireTransmission{transmissionFixture("geo-macro", "macro_anchor")}
	geopolitics.DownwardTransmission.ToIndustryChains.Paths = []wireTransmission{transmissionFixture("geo-chain", "industry_chain")}
	macroeconomics := layerSummaryFixture()
	macroeconomics.DownwardTransmission.ToMacroeconomics = nil
	macroeconomics.DownwardTransmission.ToIndustryChains.Paths = []wireTransmission{transmissionFixture("macro-chain", "industry_chain")}
	return wireHome{Report: report, Geopolitics: &wireLayerSnapshot{Key: "geopolitics", Title: "地缘政治", Summary: geopolitics}, Macroeconomics: &wireLayerSnapshot{Key: "macroeconomics", Title: "宏观经济", Summary: macroeconomics}}
}
func transmissionFixture(localKey, targetType string) wireTransmission {
	return wireTransmission{LocalKey: localKey, SourceConclusion: "源结论", Targets: []wireTransmissionTarget{{TargetType: coded(targetType, "目标类型"), TargetLocalKey: "target-01", TargetName: "目标", Result: coded("future_result", "未来结果")}}, TransmissionLogic: "传导逻辑", TransmissionKind: coded("cross_layer_reasoning", "跨层推理"), Confidence: confidence(), Status: coded("established", "已形成传导")}
}
func layerFixture() wireLayerDetail {
	return wireLayerDetail{Report: summaryFixture(), Layer: wireLayer{Key: "geopolitics", Title: "地缘政治", Summary: layerSummaryFixture(), AffectedAnchors: []wireAnchor{{LocalKey: "anchor-01", Name: "锚点", CurrentState: "UP", Result: coded("future_result", "未来结果"), ConclusionBasis: coded("direct_evidence", "直接证据"), ValidationStatus: coded("confirmed", "已确认"), Reasoning: "逻辑", TimeWindow: window(), Confidence: confidence(), EvidenceScopeToken: stringPointer(testScopeToken)}}, ReasoningSteps: []wireReasoningStep{}}}
}
func chainSummaryFixture() wireChainSummary {
	return wireChainSummary{LocalKey: "chain-01", Name: "产业链", Conclusion: "结论", Result: coded("future_result", "未来结果"), Confidence: confidence(), TimeWindow: window(), ImpactItems: []wireImpactSummary{}, EvidenceScopeToken: stringPointer(testScopeToken)}
}
func chainFixture() wireIndustryChainDetail {
	return wireIndustryChainDetail{Report: summaryFixture(), IndustryChain: wireIndustryChain{LocalKey: "chain-01", Name: "产业链", Conclusion: "结论", Result: coded("warming", "升温"), Confidence: confidence(), TimeWindow: window(), PathSummary: stringPointer("路径"), Graph: wireGraph{Nodes: []wireGraphNode{{LocalKey: "node-01", Name: "节点一"}, {LocalKey: "node-02", Name: "节点二"}, {LocalKey: "node-03", Name: "结构上下文节点"}}, Edges: []wireGraphEdge{{FromNodeLocalKey: "node-01", ToNodeLocalKey: "node-03", RelationLabel: "组成"}}}, AffectedNodes: []wireChainNode{{LocalKey: "node-01", Name: "节点一", Impact: "UP", Result: coded("warming", "升温"), ConclusionBasis: coded("direct_evidence", "直接证据"), ValidationStatus: coded("confirmed", "已确认"), Reasoning: "逻辑", TimeWindow: window(), Confidence: confidence(), EvidenceScopeToken: stringPointer(testScopeToken)}, {LocalKey: "node-02", Name: "节点二", Impact: "UP", Result: coded("pending", "待验证"), ConclusionBasis: coded("reasoning_hypothesis", "推理假设"), ValidationStatus: coded("pending_validation", "待验证"), Reasoning: "逻辑", TimeWindow: window(), Confidence: confidence()}}, CounterevidenceAndGap: stringPointer("反证与缺口"), StopCondition: stringPointer("停止条件"), EvidenceScopeToken: stringPointer(testScopeToken)}}
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
