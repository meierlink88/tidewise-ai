package researchthemeimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBatchValidateAcceptsFrozenV1Contract(t *testing.T) {
	batch := validBatch()

	window, err := batch.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !window.Start.Before(window.End) {
		t.Fatalf("validated window = %s..%s, want a positive interval", window.Start, window.End)
	}
}

func TestDecodeStrictRejectsUnknownThemeFields(t *testing.T) {
	payload := strings.Replace(validBatchJSON(), `"theme_key"`, `"confidence":0.92,"theme_key"`, 1)

	if _, err := DecodeStrict(bytes.NewBufferString(payload)); err == nil {
		t.Fatal("DecodeStrict() accepted analyst-only confidence")
	}
}

func TestDecodeStrictRejectsDuplicateAndCaseVariantFields(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		wantPath string
		wantKey  string
	}{
		{
			name:     "duplicate batch field",
			payload:  strings.Replace(validBatchJSON(), `"analysis_batch_id":`, `"analysis_batch_id":"duplicate","analysis_batch_id":`, 1),
			wantPath: "analysis_batch_id",
		},
		{
			name:     "case variant theme field",
			payload:  strings.Replace(validBatchJSON(), `"theme_key":`, `"Theme_Key":"shadow","theme_key":`, 1),
			wantPath: "themes[0].Theme_Key",
			wantKey:  "theme:ai-semiconductor-expansion",
		},
		{
			name:     "duplicate event field",
			payload:  strings.Replace(validBatchJSON(), `"event_id":`, `"event_id":"33333333-3333-4333-8333-333333333333","event_id":`, 1),
			wantPath: "themes[0].events[0].event_id",
			wantKey:  "theme:ai-semiconductor-expansion",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeStrict(bytes.NewBufferString(test.payload))
			if err == nil {
				t.Fatal("DecodeStrict() accepted an ambiguous JSON object")
			}
			var decodeError *DecodeError
			if !errors.As(err, &decodeError) {
				t.Fatalf("DecodeStrict() error = %T %v, want DecodeError", err, err)
			}
			if decodeError.Path != test.wantPath || decodeError.ThemeKey != test.wantKey {
				t.Fatalf("DecodeStrict() details = path %q theme_key %q, want %q %q", decodeError.Path, decodeError.ThemeKey, test.wantPath, test.wantKey)
			}
		})
	}
}

func TestBatchValidateRejectsInvalidFrozenContract(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Batch)
		path   string
	}{
		{name: "blank batch id", mutate: func(b *Batch) { b.AnalysisBatchID = " " }, path: "analysis_batch_id"},
		{name: "non UTC window", mutate: func(b *Batch) { b.WindowStart = "2026-07-15T08:00:00+08:00" }, path: "window_start"},
		{name: "empty batch", mutate: func(b *Batch) { b.Themes = nil }, path: "themes"},
		{name: "invalid theme key", mutate: func(b *Batch) { b.Themes[0].ThemeKey = "Theme AI" }, path: "themes[0].theme_key"},
		{name: "unsorted themes", mutate: func(b *Batch) {
			second := b.Themes[0]
			second.ThemeKey = "theme:a"
			b.Themes = append(b.Themes, second)
		}, path: "themes[1].theme_key"},
		{name: "blank market summary", mutate: func(b *Batch) { b.Themes[0].MarketConfirmationSummary = "" }, path: "themes[0].market_confirmation_summary"},
		{name: "invalid impact", mutate: func(b *Batch) { b.Themes[0].ImpactLevel = "critical" }, path: "themes[0].impact_level"},
		{name: "invalid stage", mutate: func(b *Batch) { b.Themes[0].TransmissionStage = "midstream" }, path: "themes[0].transmission_stage"},
		{name: "no chain nodes", mutate: func(b *Batch) { b.Themes[0].ChainNodes = nil }, path: "themes[0].chain_nodes"},
		{name: "uppercase node uuid", mutate: func(b *Batch) { b.Themes[0].ChainNodes[0].ChainNodeID = "AAAAAAAA-1111-4111-8111-111111111111" }, path: "themes[0].chain_nodes[0].chain_node_id"},
		{name: "invalid node role", mutate: func(b *Batch) { b.Themes[0].ChainNodes[0].RelationRole = "upstream" }, path: "themes[0].chain_nodes[0].relation_role"},
		{name: "duplicate node", mutate: func(b *Batch) { b.Themes[0].ChainNodes = append(b.Themes[0].ChainNodes, b.Themes[0].ChainNodes[0]) }, path: "themes[0].chain_nodes[1].chain_node_id"},
		{name: "no events", mutate: func(b *Batch) { b.Themes[0].Events = nil }, path: "themes[0].events"},
		{name: "invalid event role", mutate: func(b *Batch) { b.Themes[0].Events[0].EvidenceRole = "primary" }, path: "themes[0].events[0].evidence_role"},
		{name: "no driver event", mutate: func(b *Batch) { b.Themes[0].Events[0].EvidenceRole = "supporting" }, path: "themes[0].events"},
		{name: "duplicate event", mutate: func(b *Batch) { b.Themes[0].Events = append(b.Themes[0].Events, b.Themes[0].Events[0]) }, path: "themes[0].events[1].event_id"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			batch := cloneValidBatch(t)
			test.mutate(&batch)
			_, err := batch.Validate()
			if err == nil {
				t.Fatal("Validate() accepted invalid contract")
			}
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Validate() error = %T %v, want ValidationError", err, err)
			}
			if validation.Path != test.path {
				t.Fatalf("validation path = %q, want %q", validation.Path, test.path)
			}
		})
	}
}

func cloneValidBatch(t *testing.T) Batch {
	t.Helper()
	payload, err := json.Marshal(validBatch())
	if err != nil {
		t.Fatal(err)
	}
	var cloned Batch
	if err := json.Unmarshal(payload, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

func validBatchJSON() string {
	return `{
  "analysis_batch_id":"20260718T-v6-72h-validation",
  "window_start":"2026-07-15T00:00:00Z",
  "window_end":"2026-07-18T00:00:00Z",
  "themes":[{
    "theme_key":"theme:ai-semiconductor-expansion",
    "name":"AI算力扩产与半导体",
    "one_line_conclusion":"晶圆扩产增强但卡点与价格背离",
    "impact_level":"high",
    "transmission_path":"AI芯片采购 → 晶圆扩产",
    "trading_direction":"优先研究设备和材料",
    "transmission_stage":"validation",
    "next_checkpoint":"重点跟踪订单和交期",
    "market_confirmation_summary":"当前没有可归属的正式市场观测",
    "chain_nodes":[{"chain_node_id":"11111111-1111-4111-8111-111111111111","relation_role":"driver","impact_summary":"需求驱动"}],
    "events":[{"event_id":"22222222-2222-4222-8222-222222222222","evidence_role":"driver","supported_claim":"支持扩产判断"}]
  }]
}`
}

func validBatch() Batch {
	return Batch{
		AnalysisBatchID: "20260718T-v6-72h-validation",
		WindowStart:     "2026-07-15T00:00:00Z",
		WindowEnd:       "2026-07-18T00:00:00Z",
		Themes: []Theme{{
			ThemeKey:                  "theme:ai-semiconductor-expansion",
			Name:                      "AI算力扩产与半导体",
			OneLineConclusion:         "晶圆扩产增强但卡点与价格背离",
			ImpactLevel:               "high",
			TransmissionPath:          "AI芯片采购与晶圆供需缺口 → 晶圆资本开支上调 → 设备、材料与封装需求",
			TradingDirection:          "优先研究设备、材料、存储和基板等扩产约束环节",
			TransmissionStage:         "validation",
			NextCheckpoint:            "重点跟踪设备订单、交期、利用率及关键材料价格是否同步走强",
			MarketConfirmationSummary: "当前批次没有可归属的价格或正式市场观测",
			ChainNodes: []ChainNode{{
				ChainNodeID:   "11111111-1111-4111-8111-111111111111",
				RelationRole:  "driver",
				ImpactSummary: "算力扩产直接推升该节点需求与订单预期",
			}},
			Events: []Event{{
				EventID:        "22222222-2222-4222-8222-222222222222",
				EvidenceRole:   "driver",
				SupportedClaim: "该事件直接支持晶圆资本开支上调的判断",
			}},
		}},
	}
}
