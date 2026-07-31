package postgres

import (
	"testing"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
)

func TestSemanticReviewIdentityAndEvidenceAreBoundToFrozenRun(t *testing.T) {
	identity := semanticReviewIdentity{
		AgentExecutionID:   "execution",
		ReviewerPromptHash: "reviewer-hash", ReviewerModel: "reviewer-model",
		AdjudicatorPromptHash: "adjudicator-hash", AdjudicatorModel: "adjudicator-model",
	}
	if !identity.matches(eventsemantics.ReviewSubmission{
		ReviewerExecutionKey: "execution:reviewer",
		PromptHash:           "reviewer-hash", Model: "reviewer-model",
	}) {
		t.Fatal("expected frozen reviewer identity to match")
	}
	if identity.matches(eventsemantics.ReviewSubmission{
		ReviewerExecutionKey: "other-execution:reviewer",
		PromptHash:           "reviewer-hash", Model: "reviewer-model",
	}) {
		t.Fatal("review lineage from another execution was accepted")
	}
	if identity.matches(eventsemantics.ReviewSubmission{
		ReviewerExecutionKey: "execution:reviewer",
		PromptHash:           "adjudicator-hash", Model: "reviewer-model",
	}) {
		t.Fatal("mismatched prompt hash was accepted")
	}
	if !reviewEvidenceMatchesCandidate([]string{"evidence-1"}, []string{"evidence-1", "evidence-2"}) {
		t.Fatal("candidate Evidence citation should match")
	}
	if reviewEvidenceMatchesCandidate([]string{"invented"}, []string{"evidence-1"}) {
		t.Fatal("invented review Evidence was accepted")
	}
}

func TestNoSemanticCandidatesProduceARealRejectedSubmissionOutcome(t *testing.T) {
	status := summarizeSemanticSubmission(eventsemantics.PrecheckResult{})
	if status != eventsemantics.StatusRejected {
		t.Fatalf("status = %q, want rejected", status)
	}
}
