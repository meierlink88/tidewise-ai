package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

const (
	reasoningThemeID       = "11111111-1111-4111-8111-111111111111"
	reasoningAnchorID      = "22222222-2222-4222-8222-222222222222"
	reasoningCenterID      = "33333333-3333-4333-8333-333333333333"
	reasoningOtherCenterID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
)

func TestPostgresResearchReasoningTreeListReadsAndVerifiesPublishedSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	expectReasoningThemeSummary(mock, now)
	expectReasoningPublication(mock, 1, 2)

	result, err := NewPostgresRepository(db).ListResearchThemeReasoningTrees(context.Background(), reasoningThemeID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Theme.ID != reasoningThemeID || len(result.ReasoningTrees) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if result.ReasoningTrees[0].AnchorID != reasoningAnchorID || result.ReasoningTrees[0].CenterChainNodeName != "光模块" {
		t.Fatalf("tree summary = %#v", result.ReasoningTrees[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresResearchReasoningTreeListRejectsReceiptCountDrift(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	expectReasoningThemeSummary(mock, now)
	expectReasoningPublication(mock, 0, 2)

	_, err = NewPostgresRepository(db).ListResearchThemeReasoningTrees(context.Background(), reasoningThemeID)
	if !errors.Is(err, ErrResearchReasoningTreeInvariant) {
		t.Fatalf("error = %v", err)
	}
}

func TestPostgresResearchReasoningTreeDetailKeepsEventAndPathOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	expectReasoningThemeSummary(mock, now)
	expectReasoningPublication(mock, 1, 2)
	events, _ := json.Marshal([]map[string]any{
		{"event_id": "44444444-4444-4444-8444-444444444444", "title": "资本开支上调", "summary": "投入增加", "event_time": "2026-07-20T01:00:00Z", "evidence_role": "driver", "evidence_summary": "直接驱动"},
	})
	pathNodes, _ := json.Marshal([]map[string]any{
		{"position": 1, "chain_node_id": "55555555-5555-4555-8555-555555555555", "name": "AI芯片", "change_direction": "increase", "change_summary": "采购增加", "impact_summary": "扩大部署", "incoming_transmission_mechanism": nil},
		{"position": 2, "chain_node_id": reasoningCenterID, "name": "光模块", "change_direction": "mixed", "change_summary": "需求上升", "impact_summary": "订单增加", "incoming_transmission_mechanism": "集群扩容提高互联需求"},
	})
	mock.ExpectQuery(regexp.QuoteMeta(getResearchReasoningTreeDetailQuery)).
		WithArgs(reasoningThemeID, reasoningAnchorID, "99999999-9999-4999-8999-999999999999").
		WillReturnRows(sqlmock.NewRows([]string{
			"anchor_id", "theme_id", "center_chain_node_id", "center_chain_node_name",
			"one_line_conclusion", "fact_summary", "net_direction_summary", "support_summary", "counter_summary", "trading_direction", "next_checkpoint",
			"events", "path_nodes", "invalid_theme_event_count",
		}).AddRow(
			reasoningAnchorID, reasoningThemeID, reasoningCenterID, "光模块",
			"需求偏强", "资本开支增加", "偏正", "当前支持", nil, "研究订单", "观察交付", events, pathNodes, 0,
		))

	result, err := NewPostgresRepository(db).GetResearchThemeReasoningTree(context.Background(), reasoningThemeID, reasoningAnchorID)
	if err != nil {
		t.Fatal(err)
	}
	if result.ReasoningTree.Events[0].EvidenceRole != "driver" || result.ReasoningTree.PathNodes[1].Position != 2 {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresResearchReasoningTreeDetailRejectsReceiptCenterDrift(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	expectReasoningThemeSummary(mock, now)
	expectReasoningPublication(mock, 1, 2)
	events, _ := json.Marshal([]map[string]any{{
		"event_id": "44444444-4444-4444-8444-444444444444", "title": "资本开支上调", "summary": "投入增加",
		"event_time": "2026-07-20T01:00:00Z", "evidence_role": "driver", "evidence_summary": "直接驱动",
	}})
	pathNodes, _ := json.Marshal([]map[string]any{
		{"position": 1, "chain_node_id": "55555555-5555-4555-8555-555555555555", "name": "AI芯片", "change_direction": "increase", "change_summary": "采购增加", "impact_summary": "扩大部署", "incoming_transmission_mechanism": nil},
		{"position": 2, "chain_node_id": "66666666-6666-4666-8666-666666666666", "name": "先进封装", "change_direction": "mixed", "change_summary": "需求上升", "impact_summary": "订单增加", "incoming_transmission_mechanism": "需求传导"},
	})
	mock.ExpectQuery(regexp.QuoteMeta(getResearchReasoningTreeDetailQuery)).
		WithArgs(reasoningThemeID, reasoningAnchorID, "99999999-9999-4999-8999-999999999999").
		WillReturnRows(sqlmock.NewRows([]string{
			"anchor_id", "theme_id", "center_chain_node_id", "center_chain_node_name",
			"one_line_conclusion", "fact_summary", "net_direction_summary", "support_summary", "counter_summary", "trading_direction", "next_checkpoint",
			"events", "path_nodes", "invalid_theme_event_count",
		}).AddRow(
			reasoningAnchorID, reasoningThemeID, "66666666-6666-4666-8666-666666666666", "先进封装",
			"需求偏强", "资本开支增加", "偏正", "当前支持", nil, "研究订单", "观察交付", events, pathNodes, 0,
		))

	_, err = NewPostgresRepository(db).GetResearchThemeReasoningTree(context.Background(), reasoningThemeID, reasoningAnchorID)
	if !errors.Is(err, ErrResearchReasoningTreeInvariant) {
		t.Fatalf("error = %v, want invariant violation", err)
	}
}

func TestPostgresResearchReasoningTreeDetailRejectsIncompleteThemeCenterCoverage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	expectReasoningThemeSummaryWithNodes(mock, now, []map[string]any{
		{"id": reasoningCenterID, "name": "光模块", "relation_role": "beneficiary", "impact_summary": "受益"},
		{"id": reasoningOtherCenterID, "name": "先进封装", "relation_role": "constraint", "impact_summary": "约束"},
	})
	expectReasoningPublication(mock, 1, 2)

	_, err = NewPostgresRepository(db).GetResearchThemeReasoningTree(context.Background(), reasoningThemeID, reasoningAnchorID)
	if !errors.Is(err, ErrResearchReasoningTreeInvariant) {
		t.Fatalf("error = %v, want invariant violation", err)
	}
}

func TestPostgresResearchReasoningTreeDetailRejectsEventOutsideTheme(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Date(2026, 7, 20, 8, 0, 0, 0, time.UTC)
	expectReasoningThemeSummary(mock, now)
	expectReasoningPublication(mock, 1, 2)
	events, _ := json.Marshal([]map[string]any{{
		"event_id": "44444444-4444-4444-8444-444444444444", "title": "资本开支上调", "summary": "投入增加",
		"event_time": "2026-07-20T01:00:00Z", "evidence_role": "driver", "evidence_summary": "直接驱动",
	}})
	pathNodes, _ := json.Marshal([]map[string]any{
		{"position": 1, "chain_node_id": "55555555-5555-4555-8555-555555555555", "name": "AI芯片", "change_direction": "increase", "change_summary": "采购增加", "impact_summary": "扩大部署", "incoming_transmission_mechanism": nil},
		{"position": 2, "chain_node_id": reasoningCenterID, "name": "光模块", "change_direction": "mixed", "change_summary": "需求上升", "impact_summary": "订单增加", "incoming_transmission_mechanism": "需求传导"},
	})
	mock.ExpectQuery(regexp.QuoteMeta(getResearchReasoningTreeDetailQuery)).
		WithArgs(reasoningThemeID, reasoningAnchorID, "99999999-9999-4999-8999-999999999999").
		WillReturnRows(sqlmock.NewRows([]string{
			"anchor_id", "theme_id", "center_chain_node_id", "center_chain_node_name",
			"one_line_conclusion", "fact_summary", "net_direction_summary", "support_summary", "counter_summary", "trading_direction", "next_checkpoint",
			"events", "path_nodes", "invalid_theme_event_count",
		}).AddRow(
			reasoningAnchorID, reasoningThemeID, reasoningCenterID, "光模块",
			"需求偏强", "资本开支增加", "偏正", "当前支持", nil, "研究订单", "观察交付", events, pathNodes, 1,
		))

	_, err = NewPostgresRepository(db).GetResearchThemeReasoningTree(context.Background(), reasoningThemeID, reasoningAnchorID)
	if !errors.Is(err, ErrResearchReasoningTreeInvariant) {
		t.Fatalf("error = %v, want invariant violation", err)
	}
}

func TestValidReasoningTreeRejectsIncompletePublishedRows(t *testing.T) {
	mechanism := "需求传导"
	base := ResearchReasoningTree{
		AnchorID: reasoningAnchorID, CenterChainNodeID: reasoningCenterID, CenterChainNodeName: "光模块",
		OneLineConclusion: "需求偏强", FactSummary: "资本开支增加", NetDirectionSummary: "偏正",
		SupportSummary: "当前支持", TradingDirection: "研究订单", NextCheckpoint: "观察交付",
		Events: []ResearchReasoningTreeEvent{{
			EventID: "44444444-4444-4444-8444-444444444444", Title: "资本开支上调", Summary: "投入增加",
			EvidenceRole: "driver", EvidenceSummary: "直接驱动",
		}},
		PathNodes: []ResearchReasoningTreePathNode{
			{Position: 1, ChainNodeID: "55555555-5555-4555-8555-555555555555", Name: "AI芯片", ChangeDirection: "increase", ChangeSummary: "采购增加", ImpactSummary: "扩大部署"},
			{Position: 2, ChainNodeID: reasoningCenterID, Name: "光模块", ChangeDirection: "mixed", ChangeSummary: "需求上升", ImpactSummary: "订单增加", IncomingTransmissionMechanism: &mechanism},
		},
	}
	if !validReasoningTree(base) {
		t.Fatal("valid fixture was rejected")
	}

	tests := []struct {
		name   string
		mutate func(*ResearchReasoningTree)
	}{
		{name: "blank fact summary", mutate: func(value *ResearchReasoningTree) { value.FactSummary = " " }},
		{name: "blank support summary", mutate: func(value *ResearchReasoningTree) { value.SupportSummary = " " }},
		{name: "counter summary without contradiction", mutate: func(value *ResearchReasoningTree) {
			counter := "不应存在"
			value.CounterSummary = &counter
		}},
		{name: "contradiction without counter summary", mutate: func(value *ResearchReasoningTree) {
			value.Events = append(value.Events, ResearchReasoningTreeEvent{
				EventID: "77777777-7777-4777-8777-777777777777", Title: "反证", Summary: "反证事实",
				EvidenceRole: "contradicting", EvidenceSummary: "反驳判断",
			})
		}},
		{name: "unsupported change direction", mutate: func(value *ResearchReasoningTree) { value.PathNodes[0].ChangeDirection = "up" }},
		{name: "blank path impact", mutate: func(value *ResearchReasoningTree) { value.PathNodes[1].ImpactSummary = " " }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := base
			value.Events = append([]ResearchReasoningTreeEvent(nil), base.Events...)
			value.PathNodes = append([]ResearchReasoningTreePathNode(nil), base.PathNodes...)
			test.mutate(&value)
			if validReasoningTree(value) {
				t.Fatal("invalid published row was accepted")
			}
		})
	}
}

func expectReasoningThemeSummary(mock sqlmock.Sqlmock, now time.Time) {
	expectReasoningThemeSummaryWithNodes(mock, now, []map[string]any{{"id": reasoningCenterID, "name": "光模块", "relation_role": "beneficiary", "impact_summary": "受益"}})
}

func expectReasoningThemeSummaryWithNodes(mock sqlmock.Sqlmock, now time.Time, themeNodes []map[string]any) {
	nodes, _ := json.Marshal(themeNodes)
	indices := []byte(`[]`)
	mock.ExpectQuery(regexp.QuoteMeta(getResearchReasoningTreeThemeQuery)).
		WithArgs(reasoningThemeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "one_line_conclusion", "impact_level", "transmission_path", "trading_direction",
			"transmission_stage", "next_checkpoint", "market_confirmation_summary", "published_at",
			"chain_nodes", "indices", "supporting_event_count", "contradicting_event_count",
		}).AddRow(
			reasoningThemeID, "AI算力扩产", "结论", "high", "资本开支 → 光模块", "研究订单",
			"diffusion", "观察交付", "订单证据偏强", now, nodes, indices, 1, 0,
		))
}

func expectReasoningPublication(mock sqlmock.Sqlmock, actualEventCount, actualPathNodeCount int) {
	mapping, _ := json.Marshal(map[string]string{reasoningCenterID: reasoningAnchorID})
	counts, _ := json.Marshal(ResearchAnchorImportCounts{Anchors: 1, EventAssociations: 1, PathNodes: 2, Receipts: 1})
	tabs, _ := json.Marshal([]map[string]any{{"anchor_id": reasoningAnchorID, "center_chain_node_id": reasoningCenterID, "center_chain_node_name": "光模块"}})
	mock.ExpectQuery(regexp.QuoteMeta(getResearchReasoningTreePublicationQuery)).
		WithArgs(reasoningThemeID).
		WillReturnRows(sqlmock.NewRows([]string{
			"receipt_id", "anchor_ids_by_center_chain_node_id", "write_counts", "reasoning_trees", "event_association_count", "path_node_count",
		}).AddRow("99999999-9999-4999-8999-999999999999", mapping, counts, tabs, actualEventCount, actualPathNodeCount))
}
