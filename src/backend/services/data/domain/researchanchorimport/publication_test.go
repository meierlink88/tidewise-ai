package researchanchorimport

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedMultiAnchorFixtureHasFrozenIdentityAndPayloadHash(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "01-multi-anchor-import-request.json")
	fixture, err := os.Open(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()

	publication, err := DecodeStrict(fixture)
	if err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}
	if err := publication.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	hash, err := CanonicalHash(publication)
	if err != nil {
		t.Fatalf("CanonicalHash() error = %v", err)
	}
	if want := "e006ca80db77df2b07e0028d3b499b664956fd9ff0d5b57e2d00ccc6c19741a2"; hash != want {
		t.Fatalf("payload hash = %q, want %q", hash, want)
	}

	wantIDs := map[string]string{
		"22222222-2222-4222-8222-222222222222": "534d83be-774b-51d9-ad00-cdee4ba91799",
		"33333333-3333-4333-8333-333333333333": "5c18fc57-6bd8-5612-9a24-01a4e928b761",
	}
	for _, anchor := range publication.Anchors {
		if got := AnchorID(publication.ThemeID, anchor.CenterChainNodeID); got != wantIDs[anchor.CenterChainNodeID] {
			t.Fatalf("AnchorID(%q) = %q, want %q", anchor.CenterChainNodeID, got, wantIDs[anchor.CenterChainNodeID])
		}
	}
}

func TestDecodeStrictRejectsTheFirstWrongJSONType(t *testing.T) {
	payload := `{"theme_id":null,"anchors":[{"center_chain_node_id":false}]}`
	_, err := DecodeStrict(strings.NewReader(payload))
	var decodeError *DecodeError
	if !errors.As(err, &decodeError) {
		t.Fatalf("DecodeStrict() error = %T %v, want DecodeError", err, err)
	}
	if decodeError.Path != "theme_id" {
		t.Fatalf("first error path = %q, want theme_id", decodeError.Path)
	}
}

func TestDecodeStrictRejectsTheFirstMissingRequiredField(t *testing.T) {
	completeAnchor := `{"theme_id":"11111111-1111-4111-8111-111111111111","anchors":[{` +
		`"center_chain_node_id":"22222222-2222-4222-8222-222222222222",` +
		`"one_line_conclusion":"结论","fact_summary":"事实","net_direction_summary":"方向",` +
		`"support_summary":"支持","counter_summary":null,` +
		`"trading_direction":"交易","next_checkpoint":"检查","events":[],"path_nodes":[]}]}`
	tests := []struct {
		name    string
		payload string
		path    string
	}{
		{name: "top level", payload: `{"anchors":[]}`, path: "theme_id"},
		{
			name:    "support summary",
			payload: strings.Replace(completeAnchor, `"support_summary":"支持",`, "", 1),
			path:    "anchors[0].support_summary",
		},
		{
			name:    "explicit nullable counter summary",
			payload: strings.Replace(completeAnchor, `"counter_summary":null,`, "", 1),
			path:    "anchors[0].counter_summary",
		},
		{
			name: "explicit nullable path field",
			payload: `{"theme_id":"11111111-1111-4111-8111-111111111111","anchors":[{` +
				`"center_chain_node_id":"22222222-2222-4222-8222-222222222222",` +
				`"one_line_conclusion":"结论","fact_summary":"事实","net_direction_summary":"方向",` +
				`"support_summary":"支持","counter_summary":null,` +
				`"trading_direction":"交易","next_checkpoint":"检查","events":[],` +
				`"path_nodes":[{"chain_node_id":"33333333-3333-4333-8333-333333333333",` +
				`"change_direction":"increase","change_summary":"变化","impact_summary":"影响"}]}]}`,
			path: "anchors[0].path_nodes[0].incoming_transmission_mechanism",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeStrict(strings.NewReader(test.payload))
			var decodeError *DecodeError
			if !errors.As(err, &decodeError) {
				t.Fatalf("DecodeStrict() error = %T %v, want DecodeError", err, err)
			}
			if decodeError.Path != test.path {
				t.Fatalf("first error path = %q, want %q", decodeError.Path, test.path)
			}
		})
	}
}

func TestDecodeStrictRejectsNonCanonicalUUIDAsInvalidRequest(t *testing.T) {
	payload := `{"theme_id":"AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA","anchors":[]}`
	_, err := DecodeStrict(strings.NewReader(payload))
	var decodeError *DecodeError
	if !errors.As(err, &decodeError) {
		t.Fatalf("DecodeStrict() error = %T %v, want DecodeError", err, err)
	}
	if decodeError.Path != "theme_id" {
		t.Fatalf("first error path = %q, want theme_id", decodeError.Path)
	}
}

func TestPublicationRejectsFrozenAnchorInvariantsAtDeterministicPaths(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Publication)
		path   string
	}{
		{"anchor order", func(p *Publication) { p.Anchors[0], p.Anchors[1] = p.Anchors[1], p.Anchors[0] }, "anchors[1].center_chain_node_id"},
		{"required text", func(p *Publication) { p.Anchors[0].FactSummary = "   " }, "anchors[0].fact_summary"},
		{"required support summary", func(p *Publication) { p.Anchors[0].SupportSummary = "   " }, "anchors[0].support_summary"},
		{"contradiction requires counter summary", func(p *Publication) { p.Anchors[0].CounterSummary = nil }, "anchors[0].counter_summary"},
		{"contradiction rejects blank counter summary", func(p *Publication) {
			value := "   "
			p.Anchors[0].CounterSummary = &value
		}, "anchors[0].counter_summary"},
		{"counter summary forbidden without contradiction", func(p *Publication) {
			value := "不应存在的反证汇总"
			p.Anchors[1].CounterSummary = &value
		}, "anchors[1].counter_summary"},
		{"driver event", func(p *Publication) {
			for index := range p.Anchors[0].Events {
				p.Anchors[0].Events[index].EvidenceRole = "context"
			}
		}, "anchors[0].events"},
		{"duplicate event", func(p *Publication) { p.Anchors[0].Events[1].EventID = p.Anchors[0].Events[0].EventID }, "anchors[0].events[1].event_id"},
		{"short path", func(p *Publication) { p.Anchors[0].PathNodes = p.Anchors[0].PathNodes[:1] }, "anchors[0].path_nodes"},
		{"duplicate path node", func(p *Publication) { p.Anchors[0].PathNodes[1].ChainNodeID = p.Anchors[0].PathNodes[0].ChainNodeID }, "anchors[0].path_nodes[1].chain_node_id"},
		{"missing center", func(p *Publication) { p.Anchors[0].PathNodes[1].ChainNodeID = "88888888-8888-4888-8888-888888888888" }, "anchors[0].path_nodes"},
		{"first incoming mechanism", func(p *Publication) {
			value := "must be null"
			p.Anchors[0].PathNodes[0].IncomingTransmissionMechanism = &value
		}, "anchors[0].path_nodes[0].incoming_transmission_mechanism"},
		{"later missing mechanism", func(p *Publication) { p.Anchors[0].PathNodes[1].IncomingTransmissionMechanism = nil }, "anchors[0].path_nodes[1].incoming_transmission_mechanism"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			publication := readFixture(t)
			test.mutate(&publication)
			err := publication.Validate()
			var validationError *ValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("Validate() error = %T %v, want ValidationError", err, err)
			}
			if validationError.Path != test.path {
				t.Fatalf("error path = %q, want %q", validationError.Path, test.path)
			}
		})
	}
}

func readFixture(t *testing.T) Publication {
	t.Helper()
	fixturePath := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "01-multi-anchor-import-request.json")
	fixture, err := os.Open(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	publication, err := DecodeStrict(fixture)
	if err != nil {
		t.Fatal(err)
	}
	return publication
}
