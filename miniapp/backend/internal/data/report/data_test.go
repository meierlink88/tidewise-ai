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

const (
	testReportID   = "RPT11111111-1111-4111-8111-111111111111"
	testEvidenceID = "EVD11111111-1111-4111-8111-111111111111"
)

func TestRepositoryConsumesReportPublicationV2(t *testing.T) {
	t.Run("lists strict v2 summaries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != reportsPath || request.URL.Query().Get("limit") != "100" || request.URL.Query().Get("cursor") != "next" {
				t.Fatalf("request = %s?%s", request.URL.Path, request.URL.RawQuery)
			}
			writeDataResult(t, writer, wireV2Page{Items: []wireV2Summary{v2SummaryFixture()}})
		}))
		defer server.Close()
		page, err := newTestRepository(t, server).ListReports(context.Background(), biz.ListQuery{Limit: 100, Cursor: "next"})
		if err != nil || len(page.Items) != 1 || page.Items[0].ID != testReportID {
			t.Fatalf("page=%#v err=%v", page, err)
		}
	})

	t.Run("builds cards from optional sections and paged chains", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case reportsPath + "/" + testReportID + "/home":
				writeDataResult(t, writer, v2HomeFixture())
			case reportsPath + "/" + testReportID + "/layers/geopolitics":
				writeDataResult(t, writer, v2LayerFixture())
			case reportsPath + "/" + testReportID + "/industry-chains":
				writeDataResult(t, writer, wireV2ChainPage{Items: []wireV2ChainSummary{v2ChainSummaryFixture()}})
			default:
				t.Fatalf("unexpected path %s", request.URL.Path)
			}
		}))
		defer server.Close()
		home, err := newTestRepository(t, server).GetHome(context.Background(), testReportID)
		if err != nil {
			t.Fatal(err)
		}
		if home.IndustryChainCount != 1 || len(home.Cards) != 2 || home.Cards[0].Kind != biz.LayerGeopolitics ||
			home.Cards[1].Kind != biz.ScopeIndustryChain || !home.Cards[1].ImpactItems[0].HasEvidence {
			t.Fatalf("home=%#v", home)
		}
	})

	t.Run("keeps textual transmission targets without inventing a ref", func(t *testing.T) {
		fixture := v2LayerFixture()
		fixture.Layer.Summary.Transmissions[0].Targets[0].Ref = nil
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writeDataResult(t, writer, fixture)
		}))
		defer server.Close()
		detail, err := newTestRepository(t, server).GetLayer(context.Background(), testReportID, biz.LayerGeopolitics)
		if err != nil || detail.Layer.DownwardTransmission.PublishedPaths[0].TargetRefs[0].Ref != nil {
			t.Fatalf("detail=%#v err=%v", detail, err)
		}
	})

	t.Run("maps direct and hypothesis nodes without fake Evidence", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writeDataResult(t, writer, v2ChainFixture())
		}))
		defer server.Close()
		detail, err := newTestRepository(t, server).GetIndustryChain(context.Background(), testReportID, "chain-01")
		if err != nil || len(detail.IndustryChain.Nodes) != 2 || !detail.IndustryChain.Nodes[0].HasEvidence || detail.IndustryChain.Nodes[1].HasEvidence {
			t.Fatalf("detail=%#v err=%v", detail, err)
		}
	})
}

func TestRepositoryV2FailsClosed(t *testing.T) {
	t.Run("unknown response field", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(`{"request_id":"data-request","result":{"items":[],"next_cursor":null,"unknown":true}}`))
		}))
		defer server.Close()
		_, err := newTestRepository(t, server).ListReports(context.Background(), biz.ListQuery{Limit: 100})
		if !errors.Is(err, biz.ErrDataUnavailable) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("hypothesis preview cannot expose direct Evidence", func(t *testing.T) {
		fixture := v2ChainSummaryFixture()
		fixture.ImpactItems[0].Nature = wireV2Nature{Code: "reasoning_hypothesis", Label: "推理假设"}
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case reportsPath + "/" + testReportID + "/home":
				writeDataResult(t, writer, v2HomeFixture())
			case reportsPath + "/" + testReportID + "/layers/geopolitics":
				writeDataResult(t, writer, v2LayerFixture())
			default:
				writeDataResult(t, writer, wireV2ChainPage{Items: []wireV2ChainSummary{fixture}})
			}
		}))
		defer server.Close()
		_, err := newTestRepository(t, server).GetHome(context.Background(), testReportID)
		if !errors.Is(err, biz.ErrDataUnavailable) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestRepositoryEvidencePreservesPublicationOrder(t *testing.T) {
	publishedAt := "2026-09-01T03:00:00Z"
	collection := wireEvidenceCollection{ReportID: testReportID, ScopeType: "section_summary",
		ScopeKey: "geopolitics", Items: []wireEvidenceItem{{EvidenceID: testEvidenceID,
			Role: "supports_claim", DisplayOrder: 1, PublishedAt: &publishedAt,
			Summary: "海湾安全风险升温。", Keywords: []string{"海湾", "安全"}}}}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("scope_type") != "section_summary" {
			t.Fatalf("query=%v", request.URL.Query())
		}
		writeDataResult(t, writer, collection)
	}))
	defer server.Close()
	result, err := newTestRepository(t, server).ListEvidences(context.Background(), testReportID,
		biz.EvidenceScope{Type: "section_summary", Key: "geopolitics"})
	if err != nil || len(result.Items) != 1 || result.Items[0].PublishedAt == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func v2SummaryFixture() wireV2Summary {
	return wireV2Summary{ID: testReportID, PublisherReportID: "publisher-report-1",
		ReportType: "investment_reasoning", Title: "传导推理报告", GenerationStatus: "complete",
		GeneratedAt: "2026-09-01T04:00:00Z", Timezone: "Asia/Shanghai", HasGeopolitics: true,
		Statistics:  wireV2Statistics{GeopoliticAnchorCount: 1, SignaledChainNodeCount: 2, IndustryChainCount: 1},
		PublishedAt: "2026-09-01T04:01:00Z"}
}

func v2HomeFixture() wireV2Home {
	layer := v2LayerFixture().Layer
	return wireV2Home{Report: v2SummaryFixture(), Geopolitics: &wireV2LayerSnapshot{
		Key: layer.Key, Title: layer.Title, Summary: layer.Summary}}
}

func v2LayerFixture() wireV2LayerDetail {
	ref := &wireV2Reference{Type: "industry_chain", Key: "chain-01"}
	counterevidence := "相反事实可能削弱该结论。"
	boundary := "不扩展到没有 Signal 的对象。"
	reversal := "若 Signal 反转，则下调结论。"
	layer := wireV2Layer{Key: "geopolitics", Title: "地缘政治", Summary: wireV2LayerSummary{
		Claim: wireV2Claim{Key: "geo-claim", Text: "海湾安全风险升温。"},
		Transmissions: []wireV2Transmission{{Key: "geo-path", DisplayOrder: 1, SourceClaimKey: "geo-claim",
			SourceConclusion: "风险升温", Targets: []wireV2TransmissionTarget{{Ref: ref, Label: "运输链",
				Results: []wireV2NamedResult{{Name: "成本", Result: v2Result("warming", "升温")}}}},
			Logic: "运价上升。", RelationNature: "跨层推理", Confidence: v2Confidence(), Status: "published",
			EvidenceRefs: []wireV2EvidenceReference{}}},
		Uncertainty: wireV2LayerUncertainty{Counterevidence: &counterevidence, Boundary: &boundary,
			ReversalCondition: &reversal, Checkpoints: []wireCheckpoint{}},
		EvidenceRefs: []wireV2EvidenceReference{{EvidenceID: testEvidenceID, Role: "supports_claim", DisplayOrder: 1}}},
		Detail: wireV2LayerAnalysis{Anchors: []wireV2Anchor{{Key: "anchor-01", DisplayOrder: 1, Name: "海湾安全",
			Effects: []wireV2Effect{{DisplayOrder: 1, Dimension: "地缘风险", Direction: "up", Confidence: "high"}},
			Result:  v2Result("warming", "升温"), Nature: wireV2Nature{Code: "direct_evidence", Label: "直接证据"},
			Reasoning: "冲突风险影响运输。", TimeWindow: v2Window(), Confidence: v2Confidence(),
			EvidenceRefs: []wireV2EvidenceReference{{EvidenceID: testEvidenceID, Role: "direct_target", DisplayOrder: 1}}}},
			ReasoningSteps: []wireV2ReasoningStep{}, RelatedChainKeys: []string{"chain-01"}}}
	chain := v2ChainSummaryFixture()
	return wireV2LayerDetail{Report: v2SummaryFixture(), Layer: layer,
		RelatedIndustryChains: []wireV2ChainSummary{chain}}
}

func v2ChainSummaryFixture() wireV2ChainSummary {
	return wireV2ChainSummary{Key: "chain-01", DisplayOrder: 1, Name: "运输产业链",
		Claim: wireV2Claim{Key: "chain-claim", Text: "运输成本升温。"}, Status: "published",
		Result: v2Result("warming", "升温"), Confidence: v2Confidence(), TimeWindow: v2Window(),
		ImpactItems: []wireV2ChainImpactSummary{{Key: "impact-01", DisplayOrder: 1, NodeKey: "node-01",
			Name: "运输", Result: v2Result("warming", "升温"), Nature: wireV2Nature{Code: "direct_evidence", Label: "直接证据"},
			Confidence: v2Confidence(), TimeWindow: v2Window(), EvidenceCount: 1}}, EvidenceCount: 1}
}

func v2ChainFixture() wireV2IndustryChainDetail {
	direct := wireV2EvidenceReference{EvidenceID: testEvidenceID, Role: "direct_target", DisplayOrder: 1}
	chain := wireV2IndustryChain{Key: "chain-01", DisplayOrder: 1, Name: "运输产业链",
		Summary: wireV2ChainSummaryDetail{Claim: wireV2Claim{Key: "chain-claim", Text: "运输成本升温。"},
			Status: "published", Result: v2Result("warming", "升温"), Confidence: v2Confidence(),
			TimeWindow: v2Window(), Path: "安全风险 -> 运价", EvidenceRefs: []wireV2EvidenceReference{},
			Graph: wireV2IndustryChainGraph{Nodes: []wireV2TopologyNode{{Key: "node-01", DisplayOrder: 1, Name: "运输"},
				{Key: "node-02", DisplayOrder: 2, Name: "下游"}},
				Edges: []wireV2IndustryChainEdge{{Key: "edge-01", DisplayOrder: 1, FromNodeKey: "node-01", ToNodeKey: "node-02", RelationLabel: "组成"}}},
			Uncertainty: wireV2ChainUncertainty{CounterevidenceAndGap: "需经营数据验证。", StopCondition: "Signal 失效时停止。"}},
		Detail: wireV2ChainAnalysis{
			NodeImpacts: []wireV2ChainNode{{Key: "impact-01", DisplayOrder: 1, NodeKey: "node-01",
				Effects: []wireV2Effect{{DisplayOrder: 1, Dimension: "成本", Direction: "up", Confidence: "high"}},
				Result:  v2Result("warming", "升温"), Nature: wireV2Nature{Code: "direct_evidence", Label: "直接证据"},
				Reasoning: "运价上升。", TimeWindow: v2Window(), Confidence: v2Confidence(), EvidenceRefs: []wireV2EvidenceReference{direct}},
				{Key: "impact-02", DisplayOrder: 2, NodeKey: "node-02",
					Effects: []wireV2Effect{{DisplayOrder: 1, Dimension: "利润", Direction: "down", Confidence: "medium"}},
					Result:  v2Result("cooling", "降温"), Nature: wireV2Nature{Code: "reasoning_hypothesis", Label: "推理假设"},
					Reasoning: "成本向下游传导。", TimeWindow: v2Window(), Confidence: v2Confidence(), EvidenceRefs: []wireV2EvidenceReference{}}}}}
	return wireV2IndustryChainDetail{Report: v2SummaryFixture(), IndustryChain: chain}
}

func v2Result(code, label string) wireV2Result { return wireV2Result{Code: code, Label: label} }
func v2Confidence() wireV2Confidence           { return wireV2Confidence{Code: "medium", Label: "中"} }
func v2Window() wireV2TimeWindow {
	return wireV2TimeWindow{Horizons: []string{"short", "medium"}, Label: "短期-中期"}
}

func newTestRepository(t *testing.T, server *httptest.Server) *Repository {
	t.Helper()
	client, err := dataapi.NewHTTPClient(dataapi.HTTPConfig{BaseURL: server.URL, ServiceToken: "miniapp-token",
		Timeout: time.Second, MaxReadAttempts: 1, HTTPClient: server.Client()})
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
