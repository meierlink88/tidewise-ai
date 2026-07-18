package domain

import (
	"strings"
	"testing"
)

func TestAllianceOrgProfileValidate(t *testing.T) {
	valid := AllianceOrgProfile{
		EntityID:              "alliance-1",
		Abbreviation:          "G7",
		LeadershipSummary:     "成员轮值协调",
		InfluenceScopeSummary: "全球宏观政策协调",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*AllianceOrgProfile)
	}{
		{name: "missing entity", mutate: func(p *AllianceOrgProfile) { p.EntityID = " " }},
		{name: "abbreviation too long", mutate: func(p *AllianceOrgProfile) { p.Abbreviation = strings.Repeat("A", 33) }},
		{name: "placeholder abbreviation", mutate: func(p *AllianceOrgProfile) { p.Abbreviation = "—" }},
		{name: "missing leadership", mutate: func(p *AllianceOrgProfile) { p.LeadershipSummary = " " }},
		{name: "leadership too long", mutate: func(p *AllianceOrgProfile) { p.LeadershipSummary = strings.Repeat("领", 501) }},
		{name: "missing influence", mutate: func(p *AllianceOrgProfile) { p.InfluenceScopeSummary = " " }},
		{name: "influence too long", mutate: func(p *AllianceOrgProfile) { p.InfluenceScopeSummary = strings.Repeat("影", 1001) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := valid
			tt.mutate(&profile)
			if err := profile.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want validation error")
			}
		})
	}
}
