package main

import "testing"

func TestClassifySeparatesMentionRetrievalSelectorAndReviewFailures(t *testing.T) {
	china := entityIdentity{ID: "china", EntityType: "economy", CanonicalName: "中国", Status: "active"}
	brazil := entityIdentity{ID: "brazil", EntityType: "economy", CanonicalName: "巴西", Status: "active"}
	entities := []entityIdentity{china, brazil}
	tests := []struct {
		name string
		run  acceptanceRun
		text eventText
		ref  []string
		want string
	}{
		{
			name: "Stage A missed formal countries",
			run:  acceptanceRun{Status: "rejected"},
			text: eventText{Corpus: "巴西启动对中国无缝合金钢瓶的反倾销日落复审调查"},
			ref:  []string{"中国"},
			want: "mention_extraction_miss",
		},
		{
			name: "formal mention missing from retrieval",
			run: acceptanceRun{Status: "rejected", StageAudit: stageAudit{
				Mentions: []mentionAudit{{CandidateKey: "china", Mention: "中国"}},
			}},
			text: eventText{Corpus: "中国"}, want: "retrieval_miss",
		},
		{
			name: "formal candidate rejected by selector",
			run: acceptanceRun{Status: "rejected", StageAudit: stageAudit{
				Mentions:      []mentionAudit{{CandidateKey: "china", Mention: "中国"}},
				CandidateSets: []candidateSetAudit{{CandidateKey: "china", Candidates: []candidateAudit{{EntityID: "china"}}}},
				Selections:    []selectionAudit{{CandidateKey: "china", NoMatch: true}},
			}},
			text: eventText{Corpus: "中国"}, want: "selector_false_reject",
		},
		{
			name: "candidate rejected by review",
			run: acceptanceRun{Status: "rejected", EntityRejected: 1, StageAudit: stageAudit{
				Mentions:      []mentionAudit{{CandidateKey: "china", Mention: "中国"}},
				CandidateSets: []candidateSetAudit{{CandidateKey: "china", Candidates: []candidateAudit{{EntityID: "china"}}}},
				Selections:    []selectionAudit{{CandidateKey: "china", EntityID: "china"}},
			}},
			text: eventText{Corpus: "中国"}, want: "review_reject",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := classify(test.run, test.text, test.ref, entities)
			if got != test.want {
				t.Fatalf("category=%q want=%q", got, test.want)
			}
		})
	}
}

func TestExpectedFormalIdentitiesDoesNotTreatCompanySubstringAsEconomy(t *testing.T) {
	usa := entityIdentity{ID: "usa", EntityType: "economy", CanonicalName: "美国", Status: "active"}
	gabon := entityIdentity{ID: "gabon", EntityType: "economy", CanonicalName: "加蓬", Aliases: []string{"GA"}, Status: "active"}
	airline := entityIdentity{ID: "american-airlines", EntityType: "company", CanonicalName: "美国航空", Status: "active"}

	got := expectedFormalIdentities("美国航空公布业绩", []string{"美国航空"}, []entityIdentity{usa, gabon, airline})
	if len(got) != 1 || got[0].ID != airline.ID {
		t.Fatalf("identities=%+v, want only American Airlines", got)
	}

	got = expectedFormalIdentities("Israel withdrew from Zawtar al-Gharbiyeh", []string{"以色列"}, []entityIdentity{
		gabon,
		{ID: "israel", EntityType: "economy", CanonicalName: "以色列", Status: "active"},
	})
	if len(got) != 1 || got[0].ID != "israel" {
		t.Fatalf("identities=%+v, want only Israel", got)
	}
}
