package eventimport

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeAndValidateRequiredV1(t *testing.T) {
	pkg, err := DecodeStrict(strings.NewReader(validPackageJSON()))
	if err != nil {
		t.Fatalf("DecodeStrict() error = %v", err)
	}

	mapping, err := pkg.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if mapping.EventStatus != "confirmed" || mapping.FactStatus != "verified" {
		t.Fatalf("mapping = %#v", mapping)
	}
	if want := "2026-07-16T07:03:49Z"; mapping.FirstSeenAt.Format(time.RFC3339) != want {
		t.Fatalf("first_seen_at = %s, want %s", mapping.FirstSeenAt.Format(time.RFC3339), want)
	}
	if want := "2026-07-15T01:36:49Z"; mapping.KnowableAt.Format(time.RFC3339) != want {
		t.Fatalf("knowable_at = %s, want %s", mapping.KnowableAt.Format(time.RFC3339), want)
	}
}

func TestDecodeRejectsCurrentV0AndUnknownFields(t *testing.T) {
	v0 := strings.NewReplacer(
		`"package_id":"pkg-1",`, "",
		`"package_id":"pkg-1",`, "",
		`"event_tags":[{"tag_id":"b0fe1994-0db2-526c-a57f-97fa73c1b595","tag_kind":"news_category","tag_code":"geopolitics","confidence":0.98,"review_status":"approved","assignment_reason":"material fact","assign_source":"ai"}]`, `"event_tags":[]`,
	).Replace(validPackageJSON())
	if _, err := DecodeStrict(strings.NewReader(v0)); err == nil {
		t.Fatal("DecodeStrict() accepted current-v0 package")
	}

	unknown := strings.Replace(validPackageJSON(), `"package_id":"pkg-1"`, `"package_id":"pkg-1","surprise":true`, 1)
	if _, err := DecodeStrict(strings.NewReader(unknown)); err == nil {
		t.Fatal("DecodeStrict() accepted unknown field")
	}
}

func TestValidateRejectsReviewAndEvidenceMismatch(t *testing.T) {
	tests := []struct {
		name    string
		replace func(string) string
		want    string
	}{
		{
			name: "rejected",
			replace: func(value string) string {
				return strings.Replace(value, `"decision":"auto_approved"`, `"decision":"rejected"`, 1)
			},
			want: "rejected",
		},
		{
			name: "review package mismatch",
			replace: func(value string) string {
				return strings.Replace(value, `"review":{
    "review_id":"review-1",
    "package_id":"pkg-1"`, `"review":{
    "review_id":"review-1",
    "package_id":"other"`, 1)
			},
			want: "review.package_id",
		},
		{
			name: "missing supports fields",
			replace: func(value string) string {
				return strings.Replace(value, `"supports_fields":["title","factual_summary"]`, `"supports_fields":[]`, 1)
			},
			want: "supports_fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := DecodeStrict(strings.NewReader(tt.replace(validPackageJSON())))
			if err != nil {
				t.Fatalf("DecodeStrict() error = %v", err)
			}
			if _, err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestValidateRejectsUnknownAndMismatchedFrozenTagIdentity(t *testing.T) {
	for name, replacement := range map[string]string{
		"unknown code": `"tag_code":"not-a-frozen-code"`,
		"wrong uuid":   `"tag_id":"00000000-0000-0000-0000-000000000000"`,
	} {
		t.Run(name, func(t *testing.T) {
			value := validPackageJSON()
			if name == "unknown code" {
				value = strings.Replace(value, `"tag_code":"geopolitics"`, replacement, 1)
			} else {
				value = strings.Replace(value, `"tag_id":"b0fe1994-0db2-526c-a57f-97fa73c1b595"`, replacement, 1)
			}
			pkg, err := DecodeStrict(strings.NewReader(value))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "unknown event tag identity") {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestCanonicalHashUsesExactJSONSemantics(t *testing.T) {
	left, err := CanonicalHash([]byte(`{"b":1.00,"a":[1000,0.0010]}`))
	if err != nil {
		t.Fatalf("CanonicalHash(left) error = %v", err)
	}
	right, err := CanonicalHash([]byte("{\n  \"a\": [1e3, 1e-3], \"b\": 1\n}"))
	if err != nil {
		t.Fatalf("CanonicalHash(right) error = %v", err)
	}
	if left != right {
		t.Fatalf("hash mismatch: %s != %s", left, right)
	}

	different, err := CanonicalHash([]byte(`{"a":[0.001,1000],"b":1}`))
	if err != nil {
		t.Fatalf("CanonicalHash(different) error = %v", err)
	}
	if left == different {
		t.Fatal("array order did not affect canonical hash")
	}
}

func validPackageJSON() string {
	return `{
  "idempotency_key":"idem-1",
  "package_id":"pkg-1",
  "raw_documents":[{
    "document_id":"sha256:doc-1",
    "source_name":"Example",
    "source_url":"https://example.com/a",
    "title":"Document",
    "content_text":"Body",
    "content_level":"summary",
    "published_at":"2026-07-15T01:36:49Z",
    "collected_at":"2026-07-16T07:03:49Z",
    "content_hash":"0123456789abcdef"
  }],
  "event":{
    "dedupe_key":"event:v1:example",
    "title":"Event",
    "factual_summary":"A verifiable fact",
    "occurred_at":null,
    "fact_status":"verified",
    "event_status":"confirmed",
    "fact_payload":{"amount":1.00}
  },
  "event_sources":[{
    "document_id":"sha256:doc-1",
    "evidence_excerpt":"Evidence",
    "source_url":"https://example.com/a",
    "evidence_relation":"supports",
    "supports_fields":["title","factual_summary"],
    "source_level":"secondary",
    "content_level":"summary",
    "evidence_hash":"sha256:evidence-1"
  }],
  "event_tags":[{
    "tag_id":"b0fe1994-0db2-526c-a57f-97fa73c1b595",
    "tag_kind":"news_category",
    "tag_code":"geopolitics",
    "confidence":0.98,
    "review_status":"approved",
    "assignment_reason":"material fact",
    "assign_source":"ai"
  }],
  "review":{
    "review_id":"review-1",
    "package_id":"pkg-1",
    "decision":"auto_approved",
    "event_status":"confirmed",
    "fact_status":"verified",
    "evidence_grade":"single_source",
    "reasons":["evidence accepted"],
    "component_versions":{"review_policy":"v2","reviewer":"v2"}
  }
}`
}
