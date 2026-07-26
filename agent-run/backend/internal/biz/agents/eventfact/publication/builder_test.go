package publication

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

func TestBuildBatchesStableReviewedEventsWithoutSemanticRelations(t *testing.T) {
	artifact := eventfact.Artifact{
		ArtifactID: "sha256:artifact", CollectorExecutionID: "11111111-1111-4111-8111-111111111111",
		DocumentID: "sha256:artifact", Title: "原始文档", SourceName: "来源",
		SourceType: "official", SourceURL: "https://example.com/1", ContentLevel: "full_text",
		CollectedAt:   time.Date(2026, 7, 26, 1, 0, 0, 0, time.UTC),
		ContentSHA256: strings.Repeat("a", 64), Language: "zh",
	}
	result := eventfact.Result{
		ExecutionID:          "22222222-2222-4222-8222-222222222222",
		PublicationArtifacts: []eventfact.Artifact{artifact},
	}
	for index := 10; index >= 0; index-- {
		result.Candidates = append(result.Candidates, eventfact.Candidate{
			CandidateID: fmt.Sprintf("candidate:%d", index), ArtifactID: artifact.ArtifactID,
			Title: fmt.Sprintf("事件%d", index), FactualSummary: "事实摘要",
			FactPayload: map[string]any{"action": "扩产"}, EvidenceExcerpt: "逐字证据",
			SupportsFields: []string{"factual_summary"}, SourceLevel: "primary",
			ActorMentions: []string{"不得发布的原始 Mention"}, DedupeKey: fmt.Sprintf("event:%02d", index),
			IdentityHash: strings.Repeat(fmt.Sprintf("%x", index%16), 64),
			Tags: []eventfact.AssignedTag{{
				ID: "33333333-3333-4333-8333-333333333333", Kind: "news_category",
				Code: "technology", Confidence: 0.9, AssignmentReason: "权威 Catalog 分类",
			}},
			Review: eventfact.Review{
				SemanticPass: true, Reasons: []string{"证据支持事实"}, Confidence: 0.9,
			},
			ReviewState: eventfact.ReviewAutoApproved,
		})
	}
	journals, err := Build("work-key", eventfact.AgentVersion, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(journals) != 2 || journals[0].BatchOrdinal != 1 || journals[1].BatchOrdinal != 2 {
		t.Fatalf("journals = %#v", journals)
	}
	if strings.Contains(string(journals[0].Payload), "不得发布的原始 Mention") ||
		strings.Contains(string(journals[0].Payload), "entity_id") {
		t.Fatalf("semantic relation leaked: %s", journals[0].Payload)
	}
	var first request
	if err := json.Unmarshal(journals[0].Payload, &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 10 || first.Events[0].DedupeKey != "event:00" ||
		len(first.RawDocuments) != 1 || first.Provenance.ExtractorExecutionID != result.ExecutionID {
		t.Fatalf("first batch = %#v", first)
	}
	replay, err := Build("work-key", eventfact.AgentVersion, result)
	if err != nil {
		t.Fatal(err)
	}
	if string(replay[0].Payload) != string(journals[0].Payload) ||
		replay[0].PayloadHash != journals[0].PayloadHash {
		t.Fatal("publication payload is not byte-stable")
	}
}

func TestBuildExcludesManualReviewCandidates(t *testing.T) {
	result := eventfact.Result{Candidates: []eventfact.Candidate{{ReviewState: eventfact.ReviewManual}}}
	if journals, err := Build("work", eventfact.AgentVersion, result); err != nil || len(journals) != 0 {
		t.Fatalf("manual review journals=%#v err=%v", journals, err)
	}
}
