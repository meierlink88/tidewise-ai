package dbmigration

import (
	"strings"
	"testing"
)

func TestAllianceEconomyFoundationSchemaIsMinimalAndCleanupGated(t *testing.T) {
	sql := strings.ToLower(readMigration(t, "000018_reinitialize_alliance_economy_foundation.sql"))
	for _, required := range []string{
		"current_setting('tidewise.alliance_economy_schema_write_authorized', true)",
		"reviewed_local_cleanup_verified",
		"if exists (select 1 from alliance_org_profiles)",
		"alter table alliance_org_profiles",
		"drop column org_code",
		"drop column org_type",
		"drop column primary_domain",
		"drop column scope_region",
		"drop column official_url",
		"add column abbreviation text not null default ''",
		"add column leadership_summary text not null",
		"add column influence_scope_summary text not null",
		"char_length(btrim(abbreviation)) <= 32",
		"btrim(abbreviation) <> '—'",
		"char_length(btrim(leadership_summary)) <= 500",
		"char_length(btrim(influence_scope_summary)) <= 1000",
		"migration 000018 is irreversible",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("alliance/economy migration missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"entity_key",
		"economy_profiles_v18",
		"alliance_org_profiles_v18",
		"identity_kind",
		"create unique index",
		"insert into ",
		"update ",
		"delete from ",
		"truncate ",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("alliance/economy migration contains forbidden %q", forbidden)
		}
	}
}
