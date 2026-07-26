package entityseed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ApprovedAllianceEconomyManifestSHA256 = approvedAllianceEconomyManifestSHA256

type AllianceEconomyDependencyCount struct {
	Scope        string `json:"scope"`
	RelationType string `json:"relation_type,omitempty"`
	FromType     string `json:"from_type,omitempty"`
	ToType       string `json:"to_type,omitempty"`
	RowCount     int    `json:"row_count"`
}

type AllianceEconomyForeignKey struct {
	TableName       string `json:"table_name"`
	ColumnName      string `json:"column_name"`
	ReferencedTable string `json:"referenced_table"`
	DeleteRule      string `json:"delete_rule"`
}

type AllianceEconomyDependencyReport struct {
	Counts            []AllianceEconomyDependencyCount `json:"counts"`
	ForeignKeys       []AllianceEconomyForeignKey      `json:"foreign_keys"`
	Fingerprints      []string                         `json:"fingerprints"`
	CrossDomainEdges  []AllianceEconomyDependencyCount `json:"cross_domain_edges"`
	Blocked           bool                             `json:"blocked"`
	Checksum          string                           `json:"checksum"`
	ProtectedChecksum string                           `json:"protected_checksum"`
}

type AllianceEconomyCleanupResult struct {
	DeletedMemberOf           int `json:"deleted_member_of"`
	DeletedAllianceProfiles   int `json:"deleted_alliance_profiles"`
	DeletedAlliances          int `json:"deleted_alliances"`
	RemainingAlliances        int `json:"remaining_alliances"`
	RemainingAllianceProfiles int `json:"remaining_alliance_profiles"`
	RemainingMemberOf         int `json:"remaining_member_of"`
	RemainingEconomies        int `json:"remaining_economies"`
	RemainingEconomyProfiles  int `json:"remaining_economy_profiles"`
}

type AllianceEconomyRebuildResult struct {
	ManifestChecksum         string `json:"manifest_checksum"`
	Alliances                int    `json:"alliances"`
	AllianceProfiles         int    `json:"alliance_profiles"`
	Economies                int    `json:"economies"`
	EconomyProfiles          int    `json:"economy_profiles"`
	MemberOf                 int    `json:"member_of"`
	NonTargetEconomies       int    `json:"non_target_economies"`
	NonTargetEconomyProfiles int    `json:"non_target_economy_profiles"`
	Orphans                  int    `json:"orphans"`
	DuplicateTuples          int    `json:"duplicate_tuples"`
	Mismatches               int    `json:"mismatches"`
	EntityWrites             int    `json:"entity_writes"`
	ProfileWrites            int    `json:"profile_writes"`
	MemberWrites             int    `json:"member_writes"`
}

type AllianceEconomyRebuildPreflight struct {
	SchemaReady              bool
	IDConflicts              int
	KeyConflicts             int
	UnexpectedAllianceNodes  int
	UnexpectedAllianceEdges  int
	Alliances                int
	AllianceProfiles         int
	Economies                int
	EconomyProfiles          int
	NonTargetEconomies       int
	NonTargetEconomyProfiles int
	MemberOf                 int
}

func BuildAllianceEconomyDependencyReport(counts []AllianceEconomyDependencyCount, foreignKeys []AllianceEconomyForeignKey, fingerprints, protected []string) (AllianceEconomyDependencyReport, error) {
	report := AllianceEconomyDependencyReport{Counts: counts, ForeignKeys: foreignKeys, Fingerprints: fingerprints}
	for _, item := range counts {
		if item.Scope == "entity_edges" && (item.RelationType != "member_of" || item.FromType != "economy" || item.ToType != "alliance_org") && item.RowCount > 0 {
			report.CrossDomainEdges = append(report.CrossDomainEdges, item)
			if item.FromType == "alliance_org" || item.ToType == "alliance_org" {
				report.Blocked = true
			}
		}
	}
	sort.Slice(report.Counts, func(i, j int) bool {
		a, b := report.Counts[i], report.Counts[j]
		return a.Scope+"\x00"+a.RelationType+"\x00"+a.FromType+"\x00"+a.ToType < b.Scope+"\x00"+b.RelationType+"\x00"+b.FromType+"\x00"+b.ToType
	})
	sort.Slice(report.ForeignKeys, func(i, j int) bool {
		return report.ForeignKeys[i].TableName+"\x00"+report.ForeignKeys[i].ColumnName < report.ForeignKeys[j].TableName+"\x00"+report.ForeignKeys[j].ColumnName
	})
	sort.Strings(report.Fingerprints)
	sort.Strings(protected)
	payload, err := json.Marshal(struct {
		Counts       []AllianceEconomyDependencyCount `json:"counts"`
		ForeignKeys  []AllianceEconomyForeignKey      `json:"foreign_keys"`
		Fingerprints []string                         `json:"fingerprints"`
	}{report.Counts, report.ForeignKeys, report.Fingerprints})
	if err != nil {
		return AllianceEconomyDependencyReport{}, err
	}
	sum := sha256.Sum256(payload)
	report.Checksum = hex.EncodeToString(sum[:])
	report.ProtectedChecksum = AllianceEconomyFingerprintChecksum(protected)
	return report, nil
}

func ValidateAllianceEconomyCleanupAuthorization(report AllianceEconomyDependencyReport, reviewedChecksum string) error {
	if report.Blocked {
		return fmt.Errorf("cross-domain alliance/economy dependencies require an explicit Review decision")
	}
	if strings.TrimSpace(reviewedChecksum) == "" || reviewedChecksum != report.Checksum {
		return fmt.Errorf("alliance/economy dependency snapshot differs from reviewed checksum")
	}
	return nil
}

func ValidateAllianceEconomyCleanupResult(result AllianceEconomyCleanupResult) error {
	if result.RemainingAlliances != 0 || result.RemainingAllianceProfiles != 0 || result.RemainingMemberOf != 0 || result.RemainingEconomies != 50 || result.RemainingEconomyProfiles != 50 {
		return fmt.Errorf("alliance/economy cleanup zero assertion failed: alliance=%d alliance_profile=%d member_of=%d economy=%d economy_profile=%d", result.RemainingAlliances, result.RemainingAllianceProfiles, result.RemainingMemberOf, result.RemainingEconomies, result.RemainingEconomyProfiles)
	}
	return nil
}

func ValidateAllianceEconomyProtectedFacts(before, after AllianceEconomyDependencyReport) error {
	if after.Blocked || after.ProtectedChecksum != before.ProtectedChecksum {
		return fmt.Errorf("alliance/economy cleanup changed protected cross-domain facts")
	}
	return nil
}

func ClassifyAllianceEconomyRebuild(preflight AllianceEconomyRebuildPreflight) (bool, bool, error) {
	if !preflight.SchemaReady {
		return false, false, fmt.Errorf("migration 000018 must be applied in the separately authorized local rebuild package")
	}
	if preflight.IDConflicts != 0 || preflight.KeyConflicts != 0 || preflight.UnexpectedAllianceNodes != 0 || preflight.UnexpectedAllianceEdges != 0 {
		return false, false, fmt.Errorf("alliance/economy rebuild identity or scope collision: %+v", preflight)
	}
	cleanupReady := preflight.Alliances == 0 && preflight.AllianceProfiles == 0 && preflight.Economies == 35 && preflight.EconomyProfiles == 35 && preflight.NonTargetEconomies == 15 && preflight.NonTargetEconomyProfiles == 15 && preflight.MemberOf == 0
	exact := preflight.Alliances == 45 && preflight.AllianceProfiles == 45 && preflight.Economies == 79 && preflight.EconomyProfiles == 79 && preflight.NonTargetEconomies == 15 && preflight.NonTargetEconomyProfiles == 15 && preflight.MemberOf == 133
	if !cleanupReady && !exact {
		return false, false, fmt.Errorf("alliance/economy rebuild requires scoped cleanup or an exact idempotent target: %+v", preflight)
	}
	return cleanupReady, exact, nil
}

func ValidateAllianceEconomyExactResult(result AllianceEconomyRebuildResult) error {
	expected := AllianceEconomyRebuildResult{
		Alliances: 45, AllianceProfiles: 45, Economies: 79, EconomyProfiles: 79, MemberOf: 133,
		NonTargetEconomies: 15, NonTargetEconomyProfiles: 15,
	}
	comparable := result
	comparable.ManifestChecksum = ""
	comparable.EntityWrites, comparable.ProfileWrites, comparable.MemberWrites = 0, 0, 0
	if comparable != expected {
		return fmt.Errorf("existing alliance/economy target is not an exact idempotent match: %+v", result)
	}
	return nil
}

func ValidateAllianceEconomyRebuildResult(result AllianceEconomyRebuildResult) error {
	if result.Alliances != 45 || result.AllianceProfiles != 45 || result.Economies != 79 || result.EconomyProfiles != 79 || result.MemberOf != 133 || result.NonTargetEconomies != 15 || result.NonTargetEconomyProfiles != 15 || result.Orphans != 0 || result.DuplicateTuples != 0 || result.Mismatches != 0 {
		return fmt.Errorf("alliance/economy rebuild exact assertion failed: %+v", result)
	}
	return nil
}

func AllianceEconomyRebuildPayloads(manifest AllianceEconomyManifest) ([]byte, []byte, []byte, error) {
	type allianceRow struct {
		ID, EntityKey, Name                                    string
		Aliases                                                []string
		Abbreviation, LeadershipSummary, InfluenceScopeSummary string
	}
	type economyRow struct {
		ID, EntityKey, Name               string
		Aliases                           []string
		CountryCode, CurrencyCode, Region string
	}
	type edgeRow struct {
		ID, EdgeKey, FromID, ToID, SourceName, SourceURL, VerifiedAt string
	}
	alliances := make([]allianceRow, 0, len(manifest.Alliances))
	for _, item := range manifest.Alliances {
		alliances = append(alliances, allianceRow{entitySeedUUID(item.EntityKey), item.EntityKey, item.Name, item.Aliases, item.Profile.Abbreviation, item.Profile.LeadershipSummary, item.Profile.InfluenceScopeSummary})
	}
	economies := make([]economyRow, 0, len(manifest.Economies))
	for _, item := range manifest.Economies {
		economies = append(economies, economyRow{entitySeedUUID(item.EntityKey), item.EntityKey, item.Name, item.Aliases, item.CountryCode, item.CurrencyCode, item.Region})
	}
	edges := make([]edgeRow, 0, len(manifest.MemberOf))
	for _, item := range manifest.MemberOf {
		edges = append(edges, edgeRow{entitySeedUUID(item.EdgeKey), item.EdgeKey, entitySeedUUID(item.FromKey), entitySeedUUID(item.ToKey), item.SourceName, item.SourceURL, item.VerifiedAt})
	}
	alliancesPayload, err := json.Marshal(alliances)
	if err != nil {
		return nil, nil, nil, err
	}
	economiesPayload, err := json.Marshal(economies)
	if err != nil {
		return nil, nil, nil, err
	}
	memberOfPayload, err := json.Marshal(edges)
	if err != nil {
		return nil, nil, nil, err
	}
	return alliancesPayload, economiesPayload, memberOfPayload, nil
}

func AllianceEconomyFingerprintChecksum(fingerprints []string) string {
	sorted := append([]string(nil), fingerprints...)
	sort.Strings(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\n")))
	return hex.EncodeToString(sum[:])
}
