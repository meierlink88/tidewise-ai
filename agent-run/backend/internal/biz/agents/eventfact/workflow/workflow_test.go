package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

type fakeModel struct {
	responses []string
	calls     int
}

func (f *fakeModel) Generate(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
	if f.calls >= len(f.responses) {
		return nil, errors.New("unexpected model call")
	}
	response := f.responses[f.calls]
	f.calls++
	return schema.AssistantMessage(response, nil), nil
}

func (*fakeModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected stream call")
}

type fakeReader struct {
	artifacts []eventfact.Artifact
}

type fakeCanonicalReader struct {
	events []eventfact.CanonicalEvent
}

func (f fakeCanonicalReader) FindCanonicalEvents(context.Context, []string) ([]eventfact.CanonicalEvent, error) {
	return append([]eventfact.CanonicalEvent(nil), f.events...), nil
}

func (f fakeReader) Read(context.Context, []string) ([]eventfact.Artifact, error) {
	return append([]eventfact.Artifact(nil), f.artifacts...), nil
}

func TestWorkflowAutoApprovesSingleVerifiedSource(t *testing.T) {
	extractor := &fakeModel{responses: []string{`{
		"documents":[{
			"artifact_id":"sha256:artifact",
			"no_event_reason":"",
			"events":[{
				"title":"某公司宣布新增产线",
				"factual_summary":"某公司于2026年7月26日宣布新增一条产线。",
				"occurred_at":"2026-07-26T00:00:00Z",
				"fact_payload":{"lifecycle_status":"announced","action":"新增产线"},
				"evidence_excerpt":"某公司于2026年7月26日宣布新增一条产线。",
				"supports_fields":["title","factual_summary","occurred_at","fact_payload"],
				"source_level":"primary",
				"actor_mentions":["某公司"],
				"action":"新增",
				"object_mentions":["产线"],
				"lifecycle_status":"announced",
				"time_precision":"day",
				"location_mentions":[],
				"reference_period":"",
				"quantities":["一条"],
				"tag_codes":["technology"]
			}]
		}]
	}`}}
	reviewer := &fakeModel{responses: []string{`{
		"reviews":[{
			"candidate_id":"candidate:1",
			"semantic_pass":true,
			"conflict":false,
			"reasons":["事实表达与逐字证据一致"],
			"confidence":0.98
		}]
	}`}}
	artifact := eventfact.Artifact{
		ArtifactID: "sha256:artifact", CollectorExecutionID: "11111111-1111-4111-8111-111111111111",
		DocumentID: "sha256:artifact", Title: "某公司宣布新增产线",
		SourceName: "公司公告", SourceType: "official", SourceURL: "https://example.com/1",
		ContentLevel: "full_text", CollectedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		ContentSHA256: strings.Repeat("a", 64),
		Body:          "# 某公司宣布新增产线\n\n某公司于2026年7月26日宣布新增一条产线。",
	}
	runnable, err := New(context.Background(), fakeReader{artifacts: []eventfact.Artifact{artifact}}, fakeCanonicalReader{}, extractor, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: eventfact.ExecutionAttempt{
			ID: "22222222-2222-4222-8222-222222222222",
			WorkItem: eventfact.WorkItem{
				CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
			},
		},
		Catalog: eventfact.TagCatalog{
			Revision: "event-tags:r", Hash: strings.Repeat("b", 64),
			Tags: []eventfact.Tag{{
				ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
				Code: "technology", Name: "科技", IsActive: true,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 || reviewer.calls != 1 ||
		result.ExtractionModelCalls != 1 || result.ReviewModelCalls != 1 {
		t.Fatalf("model calls extraction=%d review=%d result=%#v", extractor.calls, reviewer.calls, result)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].ReviewState != eventfact.ReviewAutoApproved ||
		result.Candidates[0].Tags[0].ID != "33333333-3333-4333-8333-333333333333" ||
		result.Candidates[0].DedupeKey == "" || result.Candidates[0].IdentityHash == "" {
		t.Fatalf("candidate = %#v", result.Candidates)
	}
}

func TestWorkflowResumesPersistedFactsAtCatalogClassificationBoundary(t *testing.T) {
	extractor := &fakeModel{responses: []string{
		`{"assignments":[{"candidate_id":"candidate:1","tag_codes":["technology"]}]}`,
	}}
	reviewer := &fakeModel{responses: []string{
		`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["事实表达与逐字证据一致"],"confidence":0.98}]}`,
	}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	partialCandidate := eventfact.Candidate{
		CandidateID: "candidate:1", ArtifactID: "sha256:artifact",
		Title: "某公司宣布新增产线", FactualSummary: "某公司于2026年7月26日宣布新增一条产线。",
		FactPayload:     map[string]any{"action": "新增产线"},
		EvidenceExcerpt: "某公司于2026年7月26日宣布新增一条产线。",
		SupportsFields:  []string{"title", "factual_summary", "fact_payload"},
		SourceLevel:     "primary", ActorMentions: []string{"某公司"}, Action: "新增",
		ObjectMentions: []string{"产线"}, LifecycleStatus: "announced",
		TimePrecision: "unknown",
	}
	result, err := runnable.Invoke(context.Background(), &Input{
		Attempt: testInput().Attempt,
		Catalog: testInput().Catalog,
		ResumeResult: &eventfact.Result{
			Candidates:           []eventfact.Candidate{partialCandidate},
			ExtractionModelCalls: 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if extractor.calls != 1 || reviewer.calls != 1 ||
		result.ExtractionModelCalls != 1 || result.ReviewModelCalls != 1 ||
		result.Candidates[0].ReviewState != eventfact.ReviewAutoApproved {
		t.Fatalf(
			"resumed workflow extractor=%d reviewer=%d result=%#v",
			extractor.calls, reviewer.calls, result,
		)
	}
}

func TestWorkflowMapsSemanticConflictToManualReview(t *testing.T) {
	extractor := &fakeModel{responses: []string{validExtractionJSON()}}
	reviewer := &fakeModel{responses: []string{`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":false,"conflict":true,"reasons":["证据存在语义冲突"],"confidence":0.7}]}`}}
	runnable, err := New(context.Background(), fakeReader{artifacts: []eventfact.Artifact{testArtifact()}}, fakeCanonicalReader{}, extractor, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates[0].ReviewState != eventfact.ReviewManual {
		t.Fatalf("review state = %q", result.Candidates[0].ReviewState)
	}
}

func TestWorkflowReusesAgentRunCanonicalCoreFacts(t *testing.T) {
	identityCandidate := []eventfact.Candidate{{
		ActorMentions: []string{"某公司"}, Action: "新增", ObjectMentions: []string{"产线"},
		LifecycleStatus: "announced", TimePrecision: "day", LocationMentions: []string{},
	}}
	occurredAt := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	identityCandidate[0].OccurredAt = &occurredAt
	applyDeterministicIdentities(identityCandidate)
	canonicalCore := []byte(`{"title":"已发布规范标题","factual_summary":"已发布且不可变的事实摘要。","occurred_at":"2026-07-26T00:00:00Z","fact_payload":{"action":"既有核心事实"}}`)
	extractor := &fakeModel{responses: []string{validExtractionJSON()}}
	reviewer := &fakeModel{responses: []string{`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["新增证据支持既有核心事实"],"confidence":0.95}]}`}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{events: []eventfact.CanonicalEvent{{
			DedupeKey: "canonical:event", IdentityHash: identityCandidate[0].IdentityHash,
			CoreFacts: canonicalCore,
		}}},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	candidate := result.Candidates[0]
	if candidate.DedupeKey != "canonical:event" ||
		candidate.Title != "已发布规范标题" ||
		candidate.FactPayload["action"] != "既有核心事实" {
		t.Fatalf("canonical Candidate = %#v", candidate)
	}
}

func TestWorkflowLetsModelJudgeProgrammaticallyRecalledSemanticDuplicate(t *testing.T) {
	extractor := &fakeModel{responses: []string{validExtractionJSON()}}
	reviewer := &fakeModel{responses: []string{
		`{"judgments":[{"candidate_id":"candidate:1","dedupe_key":"canonical:semantic","same_event":true}]}`,
		`{"reviews":[{"candidate_id":"candidate:1","semantic_pass":true,"conflict":false,"reasons":["新增证据支持既有核心事实"],"confidence":0.95}]}`,
	}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{events: []eventfact.CanonicalEvent{{
			DedupeKey:    "canonical:semantic",
			IdentityHash: strings.Repeat("f", 64),
			CoreFacts: []byte(
				`{"title":"某公司扩建产线","factual_summary":"某公司宣布新增产线。","occurred_at":"2026-07-26T00:00:00Z","fact_payload":{"action":"扩建产线","lifecycle_status":"announced"}}`,
			),
		}}},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates[0].DedupeKey != "canonical:semantic" ||
		result.Candidates[0].Title != "某公司扩建产线" ||
		reviewer.calls != 2 || result.ReviewModelCalls != 2 {
		t.Fatalf("semantic duplicate result = %#v reviewerCalls=%d", result, reviewer.calls)
	}
}

func TestWorkflowPreservesRejectedNonVerbatimCandidateWithoutReviewCall(t *testing.T) {
	extractor := &fakeModel{responses: []string{strings.Replace(
		validExtractionJSON(),
		`"evidence_excerpt":"某公司于2026年7月26日宣布新增一条产线。"`,
		`"evidence_excerpt":"正文中不存在的句子"`,
		1,
	)}}
	reviewer := &fakeModel{responses: []string{`{}`}}
	runnable, err := New(context.Background(), fakeReader{artifacts: []eventfact.Artifact{testArtifact()}}, fakeCanonicalReader{}, extractor, reviewer)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 ||
		result.Candidates[0].ReviewState != eventfact.ReviewRejected ||
		len(result.Candidates[0].Review.Reasons) != 1 ||
		!strings.Contains(result.Candidates[0].Review.Reasons[0], "verbatim") {
		t.Fatalf("rejected Candidate = %#v", result.Candidates)
	}
	if extractor.calls != 1 || reviewer.calls != 0 ||
		result.ExtractionModelCalls != 1 || result.ReviewModelCalls != 0 {
		t.Fatalf("model calls extraction=%d review=%d result=%#v", extractor.calls, reviewer.calls, result)
	}
}

func TestWorkflowRejectsCamelCaseSemanticRelationInFactPayload(t *testing.T) {
	extractor := &fakeModel{responses: []string{strings.Replace(
		validExtractionJSON(),
		`"fact_payload":{"lifecycle_status":"announced","action":"新增产线"}`,
		`"fact_payload":{"action":"新增产线","eventToEntity":{"entityId":"entity-1"}}`,
		1,
	)}}
	reviewer := &fakeModel{responses: []string{`{}`}}
	runnable, err := New(
		context.Background(),
		fakeReader{artifacts: []eventfact.Artifact{testArtifact()}},
		fakeCanonicalReader{},
		extractor,
		reviewer,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := runnable.Invoke(context.Background(), testInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Candidates[0].ReviewState != eventfact.ReviewRejected ||
		!strings.Contains(result.Candidates[0].Review.Reasons[0], "forbidden semantic") {
		t.Fatalf("semantic relation Candidate = %#v", result.Candidates[0])
	}
	if reviewer.calls != 0 {
		t.Fatalf("review model calls = %d, want 0", reviewer.calls)
	}
}

func validExtractionJSON() string {
	return `{"documents":[{"artifact_id":"sha256:artifact","no_event_reason":"","events":[{"title":"某公司宣布新增产线","factual_summary":"某公司于2026年7月26日宣布新增一条产线。","occurred_at":"2026-07-26T00:00:00Z","fact_payload":{"lifecycle_status":"announced","action":"新增产线"},"evidence_excerpt":"某公司于2026年7月26日宣布新增一条产线。","supports_fields":["title","factual_summary","occurred_at","fact_payload"],"source_level":"primary","actor_mentions":["某公司"],"action":"新增","object_mentions":["产线"],"lifecycle_status":"announced","time_precision":"day","location_mentions":[],"reference_period":"","quantities":["一条"],"tag_codes":["technology"]}]}]}`
}

func testArtifact() eventfact.Artifact {
	return eventfact.Artifact{
		ArtifactID: "sha256:artifact", CollectorExecutionID: "11111111-1111-4111-8111-111111111111",
		DocumentID: "sha256:artifact", Title: "某公司宣布新增产线",
		SourceName: "公司公告", SourceType: "official", SourceURL: "https://example.com/1",
		ContentLevel: "full_text", CollectedAt: time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		ContentSHA256: strings.Repeat("a", 64),
		Body:          "# 某公司宣布新增产线\n\n某公司于2026年7月26日宣布新增一条产线。",
	}
}

func testInput() *Input {
	return &Input{
		Attempt: eventfact.ExecutionAttempt{
			ID: "22222222-2222-4222-8222-222222222222",
			WorkItem: eventfact.WorkItem{
				CollectorExecutionIDs: []string{"11111111-1111-4111-8111-111111111111"},
			},
		},
		Catalog: eventfact.TagCatalog{
			Revision: "event-tags:r", Hash: strings.Repeat("b", 64),
			Tags: []eventfact.Tag{{
				ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
				Code: "technology", Name: "科技", IsActive: true,
			}},
		},
	}
}
