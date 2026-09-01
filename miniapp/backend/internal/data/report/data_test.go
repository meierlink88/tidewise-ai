package report

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/report"
	dataapi "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
)

const (
	testReportID   = "RPT11111111-1111-4111-8111-111111111111"
	testEvidenceID = "EVD11111111-1111-4111-8111-111111111111"
)

func TestRepositoryListReportsUsesVersionedRangePaginationAndStrictSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != reportsPath {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("published_from") != "2026-08-31T16:00:00Z" || query.Get("published_to") != "2026-09-01T16:00:00Z" ||
			query.Get("limit") != "100" || query.Get("cursor") != "next-page" {
			t.Fatalf("query = %v", query)
		}
		if request.Header.Get("Authorization") != "Bearer miniapp-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		next := "last-page"
		writeDataResult(t, writer, wirePage{Items: []wireSummary{summaryFixture()}, NextCursor: &next})
	}))
	defer server.Close()
	repository := newTestRepository(t, server)
	from := time.Date(2026, 8, 31, 16, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 1, 16, 0, 0, 0, time.UTC)

	page, err := repository.ListReports(context.Background(), biz.ListQuery{
		PublishedFrom: &from, PublishedTo: &to, Limit: 100, Cursor: "next-page",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != testReportID || page.Items[0].GeneratedAt.Location() != time.UTC ||
		page.NextCursor == nil || *page.NextCursor != "last-page" {
		t.Fatalf("page = %#v", page)
	}
}

func TestRepositoryMapsPersistedCardsAndRejectsContractDrift(t *testing.T) {
	t.Run("maps evidence counts without exposing IDs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writeDataResult(t, writer, homeFixture())
		}))
		defer server.Close()
		repository := newTestRepository(t, server)
		home, err := repository.GetHome(context.Background(), testReportID)
		if err != nil {
			t.Fatal(err)
		}
		if home.IndustryChainCount != 54 || len(home.Cards) != 2 || !home.Cards[0].HasEvidence || len(home.Cards[0].ImpactItems) != 1 ||
			home.Cards[0].ImpactItems[0].HasEvidence || home.Cards[0].DetailRef.Key != biz.LayerGeopolitics {
			t.Fatalf("home = %#v", home)
		}
	})

	t.Run("industry chain count must match the persisted statistics projection", func(t *testing.T) {
		for _, test := range []struct {
			name  string
			count *int
		}{
			{name: "missing", count: nil},
			{name: "mismatched", count: func() *int { count := 53; return &count }()},
		} {
			t.Run(test.name, func(t *testing.T) {
				fixture := homeFixture()
				fixture.IndustryChainCount = test.count
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					writeDataResult(t, writer, fixture)
				}))
				defer server.Close()
				if _, err := newTestRepository(t, server).GetHome(context.Background(), testReportID); !errors.Is(err, biz.ErrDataUnavailable) {
					t.Fatalf("invalid industry chain count error = %v", err)
				}
			})
		}
	})

	t.Run("missing fixed layer card fails closed", func(t *testing.T) {
		fixture := homeFixture()
		fixture.ReportCards = fixture.ReportCards[:1]
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writeDataResult(t, writer, fixture)
		}))
		defer server.Close()
		if _, err := newTestRepository(t, server).GetHome(context.Background(), testReportID); !errors.Is(err, biz.ErrDataUnavailable) {
			t.Fatalf("missing macro card error = %v", err)
		}
	})

	t.Run("impact target must match card kind", func(t *testing.T) {
		fixture := homeFixture()
		fixture.ReportCards[0].ImpactItems[0].Ref = wireReference{Type: biz.ScopeIndustryChainNode, Key: "node-1"}
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writeDataResult(t, writer, fixture)
		}))
		defer server.Close()
		if _, err := newTestRepository(t, server).GetHome(context.Background(), testReportID); !errors.Is(err, biz.ErrDataUnavailable) {
			t.Fatalf("mismatched impact reference error = %v", err)
		}
	})

	t.Run("unknown field fails closed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"request_id":"data-request","result":{"items":[],"next_cursor":null,"event_count":113}}`))
		}))
		defer server.Close()
		repository := newTestRepository(t, server)
		if _, err := repository.ListReports(context.Background(), biz.ListQuery{Limit: 100}); !errors.Is(err, biz.ErrDataUnavailable) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("missing zero-valued field fails closed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"request_id":"data-request","result":{"items":[{"id":"RPT11111111-1111-4111-8111-111111111111","source_report_id":"source","report_type":"reasoning","title":"报告","status":"published","generated_at":"2026-09-01T04:00:00Z","timezone":"Asia/Shanghai","published_layers":[],"statistics":{"event_count":0,"ordinary_fact_count":0,"signal_fact_count":0,"transmission_hypothesis_count":0,"remaining_topology_pending_count":0,"adaptive_inclusion_threshold":0,"adaptive_continuation_threshold":0,"adaptive_hard_max_hops":0,"adaptive_observed_max_hops":0,"adaptive_stopped_by_confidence":0,"adaptive_stopped_by_no_unvisited_neighbor":0,"adaptive_rejected_below_inclusion":0,"geopolitic_anchor_count":0,"macroeconomic_anchor_count":0,"signaled_chain_node_count":0,"industry_chain_count":0,"unmapped_chain_node_count":0},"published_at":"2026-09-01T04:01:00Z"}],"next_cursor":null}}`))
		}))
		defer server.Close()
		repository := newTestRepository(t, server)
		if _, err := repository.ListReports(context.Background(), biz.ListQuery{Limit: 100}); !errors.Is(err, biz.ErrDataUnavailable) {
			t.Fatalf("missing simulation error = %v", err)
		}
	})
}

func TestRepositoryMapsLayerTargetsAndPreservesReportChainOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != reportsPath+"/"+testReportID+"/layers/geopolitics" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		writeDataResult(t, writer, layerFixture())
	}))
	defer server.Close()
	repository := newTestRepository(t, server)

	detail, err := repository.GetLayer(context.Background(), testReportID, biz.LayerGeopolitics)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Layer.DownwardTransmission.PublishedPaths) != 1 ||
		detail.Layer.DownwardTransmission.PublishedPaths[0].TargetRefs[0].Ref != (biz.Reference{Type: biz.ScopeIndustryChain, Key: "chain-21"}) {
		t.Fatalf("paths = %#v", detail.Layer.DownwardTransmission.PublishedPaths)
	}
	if len(detail.RelatedIndustryChains) != 1 || detail.RelatedIndustryChains[0].DisplayOrder != 21 ||
		detail.Layer.Anchors[0].Scope != (biz.EvidenceScope{Type: biz.ScopeAnchor, Key: "anchor-g1"}) {
		t.Fatalf("detail = %#v", detail)
	}
}

func TestRepositoryValidatesExplicitGraphEndpointClosure(t *testing.T) {
	valid := chainFixture()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeDataResult(t, writer, valid)
	}))
	defer server.Close()
	repository := newTestRepository(t, server)
	if _, err := repository.GetIndustryChain(context.Background(), testReportID, "chain-21"); err != nil {
		t.Fatal(err)
	}

	invalid := chainFixture()
	invalid.IndustryChain.Edges[0].ToNodeKey = "missing-node"
	invalidServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeDataResult(t, writer, invalid)
	}))
	defer invalidServer.Close()
	if _, err := newTestRepository(t, invalidServer).GetIndustryChain(context.Background(), testReportID, "chain-21"); !errors.Is(err, biz.ErrDataUnavailable) {
		t.Fatalf("invalid graph error = %v", err)
	}
}

func TestRepositoryEvidencePreservesPublicationOrderAndDropsInternalIdentity(t *testing.T) {
	newer := "2026-09-01T03:00:00Z"
	older := "2026-08-31T03:00:00+08:00"
	collection := wireEvidenceCollection{ReportID: testReportID, ScopeType: biz.ScopeReportCard,
		ScopeKey: "geo-card", Items: []wireEvidenceItem{
			{EvidenceID: testEvidenceID, Role: "supports", DisplayOrder: 1, PublishedAt: &older,
				Summary: "第一条由发布顺序决定。", Keywords: []string{"第一"}},
			{EvidenceID: "EVD22222222-2222-4222-8222-222222222222", Role: "supports", DisplayOrder: 2,
				PublishedAt: &newer, Summary: "时间更新但仍居第二。", Keywords: []string{"第二"}},
		}}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("scope_type") != biz.ScopeReportCard || request.URL.Query().Get("scope_key") != "geo-card" {
			t.Fatalf("query = %v", request.URL.Query())
		}
		writeDataResult(t, writer, collection)
	}))
	defer server.Close()
	repository := newTestRepository(t, server)

	result, err := repository.ListEvidences(context.Background(), testReportID,
		biz.EvidenceScope{Type: biz.ScopeReportCard, Key: "geo-card"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || result.Items[0].Summary != "第一条由发布顺序决定。" ||
		result.Items[0].PublishedAt == nil || result.Items[0].PublishedAt.Location() != time.UTC {
		t.Fatalf("result = %#v", result)
	}

	collection.Items[1].DisplayOrder = 3
	invalidServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writeDataResult(t, writer, collection)
	}))
	defer invalidServer.Close()
	if _, err := newTestRepository(t, invalidServer).ListEvidences(context.Background(), testReportID,
		biz.EvidenceScope{Type: biz.ScopeReportCard, Key: "geo-card"}); !errors.Is(err, biz.ErrDataUnavailable) {
		t.Fatalf("discontinuous order error = %v", err)
	}
}

func TestRepositoryMapsDataNotFoundWithoutLeakingBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set(dataapi.RequestIDHeader, "data-not-found")
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte(`{"request_id":"data-not-found","error":{"code":"REPORT_NOT_FOUND","message":"postgres secret"}}`))
	}))
	defer server.Close()
	repository := newTestRepository(t, server)
	_, err := repository.GetHome(context.Background(), testReportID)
	if !errors.Is(err, biz.ErrReportNotFound) || strings.Contains(err.Error(), "postgres secret") {
		t.Fatalf("error = %v", err)
	}
}

func TestRepositoryMapsEvidenceScopeNotFound(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		code   string
		wanted error
	}{
		{name: "stable scope code", status: http.StatusNotFound, code: "REPORT_EVIDENCE_SCOPE_NOT_FOUND", wanted: biz.ErrEvidenceScopeNotFound},
		{name: "invalid request fails closed", status: http.StatusBadRequest, code: "INVALID_REQUEST", wanted: biz.ErrDataUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set(dataapi.RequestIDHeader, "data-evidence-error")
				writer.WriteHeader(test.status)
				_ = json.NewEncoder(writer).Encode(map[string]any{
					"request_id": "data-evidence-error",
					"error":      map[string]any{"code": test.code, "message": "downstream detail"},
				})
			}))
			defer server.Close()
			repository := newTestRepository(t, server)
			_, err := repository.ListEvidences(context.Background(), testReportID,
				biz.EvidenceScope{Type: biz.ScopeReportCard, Key: "missing-card"})
			if !errors.Is(err, test.wanted) {
				t.Fatalf("error = %v, want %v", err, test.wanted)
			}
		})
	}
}

func summaryFixture() wireSummary {
	return wireSummary{ID: testReportID, SourceReportID: "source-report-1", ReportType: "investment_reasoning",
		Title: "传导推理报告", Status: "published", GeneratedAt: "2026-09-01T12:00:00+08:00",
		Timezone: "Asia/Shanghai", PublishedLayers: []string{"geopolitics", "macroeconomics", "industry_chain"},
		Statistics: wireStatistics{AdaptiveInclusionThreshold: 0.6, AdaptiveContinuationThreshold: 0.5,
			IndustryChainCount: 54},
		PublishedAt: "2026-09-01T04:01:00Z"}
}

func homeFixture() wireHome {
	industryChainCount := 54
	return wireHome{Report: summaryFixture(), IndustryChainCount: &industryChainCount, ReportCards: []wireCard{
		{Key: "geo-card", Kind: biz.LayerGeopolitics,
			DisplayOrder: 1, DetailRef: wireReference{Type: biz.ScopeLayer, Key: biz.LayerGeopolitics},
			Title: "地缘政治", Subtitle: "风险主线", Conclusion: "海湾安全风险升温。",
			Result: resultFixture(), Confidence: confidenceFixture(), TimeWindow: "短期",
			ImpactItems: []wireCardImpactItem{{Ref: wireReference{Type: biz.ScopeAnchor, Key: "anchor-g1"},
				Name: "海湾安全", Result: resultFixture(), Confidence: confidenceFixture(), TimeWindow: "短期", EvidenceCount: 0}},
			EvidenceCount: 2},
		{Key: "macro-card", Kind: biz.LayerMacroeconomics,
			DisplayOrder: 2, DetailRef: wireReference{Type: biz.ScopeLayer, Key: biz.LayerMacroeconomics},
			Title: "宏观经济", Subtitle: "需求主线", Conclusion: "宏观路径分化。",
			Result: wireResult{Code: "diverging", Label: "分化"}, Confidence: confidenceFixture(), TimeWindow: "中期",
			ImpactItems: []wireCardImpactItem{{Ref: wireReference{Type: biz.ScopeAnchor, Key: "anchor-m1"},
				Name: "需求路径", Result: wireResult{Code: "diverging", Label: "分化"},
				Confidence: confidenceFixture(), TimeWindow: "中期", EvidenceCount: 1}},
			EvidenceCount: 1},
	}, Company: wireCompanyBoundary{Key: "company", DisplayOrder: 4, Title: "企业",
		Published: false, Boundary: "本次推理尚未进入企业层。"}}
}

func layerFixture() wireLayerDetail {
	return wireLayerDetail{Report: summaryFixture(), Layer: wireLayer{Key: biz.LayerGeopolitics,
		DisplayOrder: 1, Title: "地缘政治", Conclusion: "海湾风险升温。", Result: resultFixture(),
		Confidence: confidenceFixture(), TimeWindow: "短期", Anchors: []wireAnchor{{Key: "anchor-g1", DisplayOrder: 1,
			Name: "海湾安全", CurrentState: "风险上升", Result: resultFixture(),
			Nature: natureFixture(), Reasoning: "冲突风险影响运输。", TimeWindow: "短期",
			Confidence: confidenceFixture(), EvidenceCount: 1}}, ReasoningSteps: []wireReasoningStep{},
		RelatedAnchorKeys: []string{}, RelatedChainKeys: []string{"chain-21"},
		DownwardTransmission: wireDownwardTransmission{Summary: "风险向产业链传导。",
			PublishedPaths: []wireTransmissionPath{{Key: "path-g1", DisplayOrder: 1,
				SourceConclusion: "风险升温", TargetRefs: []wireTransmissionTarget{{
					Ref: wireReference{Type: biz.ScopeIndustryChain, Key: "chain-21"}, Label: "运输链", Result: resultFixture()}},
				Logic: "运价上升。", RelationNature: "推理假设", EvidenceRole: "路径证据",
				Confidence: confidenceFixture(), Status: "已发布", EvidenceCount: 0}},
			CandidateMechanisms: []wireCandidateMechanism{}, BoundaryNotes: []string{"不表达正式因果。"}},
		Uncertainty: wireLayerUncertainty{Checkpoints: []wireCheckpoint{}}, EvidenceCount: 1},
		RelatedIndustryChains: []wireIndustryChainSummary{{Key: "chain-21", DisplayOrder: 21, Name: "运输链",
			Conclusion: "成本升温。", Status: "已发布", Result: resultFixture(),
			Confidence: confidenceFixture(), TimeWindow: "中期", EvidenceCount: 1}}}
}

func chainFixture() wireIndustryChainDetail {
	return wireIndustryChainDetail{Report: summaryFixture(), IndustryChain: wireIndustryChain{
		Key: "chain-21", ClaimKey: "claim-21", DisplayOrder: 21, Name: "运输链", Conclusion: "成本升温。",
		Status: "已发布", Result: resultFixture(), Confidence: confidenceFixture(), TimeWindow: "中期",
		Nodes: []wireIndustryChainNode{{Key: "node-1", DisplayOrder: 1, Name: "运输", Impact: "成本上升",
			Result: resultFixture(), Nature: natureFixture(), Reasoning: "运价上升。", TimeWindow: "短期",
			Confidence: confidenceFixture(), EvidenceCount: 1}, {Key: "node-2", DisplayOrder: 2, Name: "下游",
			Impact: "利润承压", Result: wireResult{Code: "cooling", Label: "降温"}, Nature: natureFixture(),
			Reasoning: "成本向下游传导。", TimeWindow: "中期", Confidence: confidenceFixture(), EvidenceCount: 0}},
		Edges:       []wireIndustryChainEdge{{Key: "edge-1", DisplayOrder: 1, FromNodeKey: "node-1", ToNodeKey: "node-2", RelationLabel: "传导"}},
		Uncertainty: wireChainUncertainty{Checkpoints: []wireCheckpoint{}}, EvidenceCount: 1}}
}

func resultFixture() wireResult { return wireResult{Code: "warming", Label: "升温"} }

func natureFixture() wireNature { return wireNature{Code: "direct_evidence", Label: "直接证据"} }

func confidenceFixture() wireConfidence { return wireConfidence{Label: "中", Score: nil} }

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
