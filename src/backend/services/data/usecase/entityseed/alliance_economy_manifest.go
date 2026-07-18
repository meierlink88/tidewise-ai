package seed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const (
	approvedAllianceEconomyManifestSHA256 = "118573006c341830f3c977a994d12a537fe2d1ac8174a818d8007fe98e87a55d"
	approvedAllianceChecksum              = "4e5be67e7c87871de0958862b62c453e08d8fbb5b6ce138904d053a58864ef5a"
	approvedAllianceDetailsChecksum       = "07a3362347088e264f4ff977acbb3e6d6010f40d6ceb05c3f390a6fa7d9d4043"
	approvedEconomyChecksum               = "66c1e13af00ca1c898132fec113812e971aa52adbf2da17400e331a4c5ae3db1"
	approvedEconomyDetailsChecksum        = "71086becf32081414f99c6fb9a4a139df80a793c610aa89a5e0b1f797c8e2f52"
	approvedMemberOfChecksum              = "c3d652571fa93307088633cbfe06dce18b1257d8ac2b3ee2ae88c5a27e69fcf7"
	approvedMemberOfDetailsChecksum       = "12e8f23e4219a0aa26484b516e14a624197cf64366c9a09e066b7ff7f4fce910"
)

type AllianceEconomyManifest struct {
	Version   int                      `json:"version"`
	Checksums AllianceEconomyChecksums `json:"checksums"`
	Alliances []ApprovedAlliance       `json:"alliances"`
	Economies []ApprovedEconomy        `json:"economies"`
	MemberOf  []ApprovedMemberOf       `json:"member_of"`
}

type AllianceEconomyChecksums struct {
	Alliances       string `json:"alliances"`
	AllianceDetails string `json:"alliance_details"`
	Economies       string `json:"economies"`
	EconomyDetails  string `json:"economy_details"`
	MemberOf        string `json:"member_of"`
	MemberOfDetails string `json:"member_of_details"`
}

type ApprovedAlliance struct {
	SheetRow  int                     `json:"sheet_row"`
	EntityKey string                  `json:"entity_key"`
	Name      string                  `json:"name"`
	Aliases   []string                `json:"aliases"`
	Profile   ApprovedAllianceProfile `json:"profile"`
	Action    string                  `json:"action"`
	Decision  string                  `json:"decision"`
}

type ApprovedAllianceProfile struct {
	Abbreviation          string `json:"abbreviation"`
	LeadershipSummary     string `json:"leadership_summary"`
	InfluenceScopeSummary string `json:"influence_scope_summary"`
}

type ApprovedEconomy struct {
	EntityKey    string   `json:"entity_key"`
	Name         string   `json:"name"`
	EnglishName  string   `json:"english_name"`
	Aliases      []string `json:"aliases"`
	CountryCode  string   `json:"country_code"`
	CurrencyCode string   `json:"currency_code"`
	Region       string   `json:"region"`
	Action       string   `json:"action"`
}

type ApprovedMemberOf struct {
	EdgeKey          string `json:"edge_key"`
	FromKey          string `json:"from_key"`
	RelationType     string `json:"relation_type"`
	ToKey            string `json:"to_key"`
	MembershipStatus string `json:"membership_status"`
	SourceName       string `json:"source_name"`
	SourceURL        string `json:"source_url"`
	VerifiedAt       string `json:"verified_at"`
	Action           string `json:"action"`
}

func LoadApprovedAllianceEconomyManifest(path string) (AllianceEconomyManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AllianceEconomyManifest{}, fmt.Errorf("read approved alliance economy manifest: %w", err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != approvedAllianceEconomyManifestSHA256 {
		return AllianceEconomyManifest{}, fmt.Errorf("approved alliance economy manifest frozen file checksum mismatch")
	}
	return decodeApprovedAllianceEconomyManifest(data)
}

func decodeApprovedAllianceEconomyManifest(data []byte) (AllianceEconomyManifest, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var manifest AllianceEconomyManifest
	if err := decoder.Decode(&manifest); err != nil {
		return AllianceEconomyManifest{}, fmt.Errorf("decode approved alliance economy manifest: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return AllianceEconomyManifest{}, fmt.Errorf("approved alliance economy manifest must contain a single JSON document")
		}
		return AllianceEconomyManifest{}, fmt.Errorf("approved alliance economy manifest must contain a single JSON document: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return AllianceEconomyManifest{}, err
	}
	return manifest, nil
}

func (m AllianceEconomyManifest) Validate() error {
	if m.Version != 1 {
		return fmt.Errorf("unsupported approved alliance economy manifest version %d", m.Version)
	}
	if m.Checksums != (AllianceEconomyChecksums{Alliances: approvedAllianceChecksum, AllianceDetails: approvedAllianceDetailsChecksum, Economies: approvedEconomyChecksum, EconomyDetails: approvedEconomyDetailsChecksum, MemberOf: approvedMemberOfChecksum, MemberOfDetails: approvedMemberOfDetailsChecksum}) {
		return fmt.Errorf("approved alliance economy manifest declared checksum mismatch")
	}
	if got := canonicalAllianceChecksum(m.Alliances); got != m.Checksums.Alliances {
		return fmt.Errorf("alliance canonical checksum mismatch: got %s", got)
	}
	if got := canonicalAllianceDetailsChecksum(m.Alliances); got != m.Checksums.AllianceDetails {
		return fmt.Errorf("alliance details canonical checksum mismatch: got %s", got)
	}
	if got := canonicalEconomyChecksum(m.Economies); got != m.Checksums.Economies {
		return fmt.Errorf("economy canonical checksum mismatch: got %s", got)
	}
	if got := canonicalEconomyDetailsChecksum(m.Economies); got != m.Checksums.EconomyDetails {
		return fmt.Errorf("economy details canonical checksum mismatch: got %s", got)
	}
	if got := canonicalMemberOfChecksum(m.MemberOf); got != m.Checksums.MemberOf {
		return fmt.Errorf("member_of canonical checksum mismatch: got %s", got)
	}
	if got := canonicalMemberOfDetailsChecksum(m.MemberOf); got != m.Checksums.MemberOfDetails {
		return fmt.Errorf("member_of details canonical checksum mismatch: got %s", got)
	}
	allianceKeys, err := validateApprovedAlliances(m.Alliances)
	if err != nil {
		return err
	}
	economyKeys, err := validateApprovedEconomies(m.Economies)
	if err != nil {
		return err
	}
	_, err = validateApprovedMemberOf(m.MemberOf, allianceKeys, economyKeys)
	if err != nil {
		return err
	}
	return nil
}

func validateApprovedAlliances(items []ApprovedAlliance) (map[string]struct{}, error) {
	if len(items) != 45 {
		return nil, fmt.Errorf("approved alliances = %d, want 45", len(items))
	}
	keys := make(map[string]struct{}, len(items))
	actions := map[string]int{}
	rows := map[int]struct{}{}
	for _, item := range items {
		if item.SheetRow <= 0 || strings.TrimSpace(item.Name) == "" || !strings.HasPrefix(item.EntityKey, "alliance_org:") || item.Decision != "approve" {
			return nil, fmt.Errorf("invalid approved alliance %q", item.EntityKey)
		}
		if _, exists := keys[item.EntityKey]; exists {
			return nil, fmt.Errorf("duplicate alliance key %q", item.EntityKey)
		}
		if _, exists := rows[item.SheetRow]; exists {
			return nil, fmt.Errorf("duplicate alliance sheet row %d", item.SheetRow)
		}
		if err := validateAliases(item.Aliases); err != nil {
			return nil, fmt.Errorf("alliance %q aliases: %w", item.EntityKey, err)
		}
		profile := domain.AllianceOrgProfile{EntityID: item.EntityKey, Abbreviation: item.Profile.Abbreviation, LeadershipSummary: item.Profile.LeadershipSummary, InfluenceScopeSummary: item.Profile.InfluenceScopeSummary}
		if err := profile.Validate(); err != nil {
			return nil, fmt.Errorf("alliance %q profile: %w", item.EntityKey, err)
		}
		if profile.Abbreviation != "" && !containsNormalized(item.Aliases, profile.Abbreviation) {
			return nil, fmt.Errorf("alliance %q abbreviation is absent from aliases", item.EntityKey)
		}
		if item.Action != "keep" && item.Action != "create" {
			return nil, fmt.Errorf("alliance %q has unsupported action %q", item.EntityKey, item.Action)
		}
		keys[item.EntityKey], rows[item.SheetRow] = struct{}{}, struct{}{}
		actions[item.Action]++
	}
	if actions["keep"] != 9 || actions["create"] != 36 {
		return nil, fmt.Errorf("alliance actions = keep %d create %d, want 9/36", actions["keep"], actions["create"])
	}
	return keys, nil
}

func validateApprovedEconomies(items []ApprovedEconomy) (map[string]struct{}, error) {
	if len(items) != 79 {
		return nil, fmt.Errorf("approved economies = %d, want 79", len(items))
	}
	keys, codes := map[string]struct{}{}, map[string]struct{}{}
	actions := map[string]int{}
	for _, item := range items {
		if item.EntityKey != "economy:"+strings.ToLower(item.CountryCode) || strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.EnglishName) == "" {
			return nil, fmt.Errorf("economy %q has invalid stable key or name", item.EntityKey)
		}
		if _, exists := keys[item.EntityKey]; exists {
			return nil, fmt.Errorf("duplicate economy key %q", item.EntityKey)
		}
		if _, exists := codes[item.CountryCode]; exists {
			return nil, fmt.Errorf("duplicate active economy country code %q", item.CountryCode)
		}
		if err := validateAliases(item.Aliases); err != nil {
			return nil, fmt.Errorf("economy %q aliases: %w", item.EntityKey, err)
		}
		if !containsNormalized(item.Aliases, item.EnglishName) || !containsNormalized(item.Aliases, item.CountryCode) {
			return nil, fmt.Errorf("economy %q aliases must contain English name and country code", item.EntityKey)
		}
		if strings.TrimSpace(item.CurrencyCode) == "" || strings.TrimSpace(item.Region) == "" {
			return nil, fmt.Errorf("economy %q has incomplete approved profile", item.EntityKey)
		}
		if item.Action != "reuse" && item.Action != "create" {
			return nil, fmt.Errorf("economy %q has unsupported action %q", item.EntityKey, item.Action)
		}
		keys[item.EntityKey], codes[item.CountryCode] = struct{}{}, struct{}{}
		actions[item.Action]++
	}
	if actions["reuse"] != 35 || actions["create"] != 44 {
		return nil, fmt.Errorf("economy actions = reuse %d create %d, want 35/44", actions["reuse"], actions["create"])
	}
	return keys, nil
}

func validateApprovedMemberOf(items []ApprovedMemberOf, allianceKeys, economyKeys map[string]struct{}) (map[string]struct{}, error) {
	if len(items) != 133 {
		return nil, fmt.Errorf("approved member_of = %d, want 133", len(items))
	}
	wantCounts := map[string]int{"alliance_org:g7": 7, "alliance_org:nato": 32, "alliance_org:sco": 10, "alliance_org:eu": 27, "alliance_org:asean": 11, "alliance_org:gcc": 6, "alliance_org:eaeu": 5, "alliance_org:opec": 12, "alliance_org:gecf": 12, "alliance_org:brics": 11}
	keys, tuples := map[string]struct{}{}, map[string]struct{}{}
	counts, actions := map[string]int{}, map[string]int{}
	sources := map[string]string{}
	for _, item := range items {
		if item.RelationType != "member_of" || item.MembershipStatus != "formal_active" {
			return nil, fmt.Errorf("member_of %q must be formal_active member_of", item.EdgeKey)
		}
		if _, exists := economyKeys[item.FromKey]; !exists {
			return nil, fmt.Errorf("member_of %q has invalid economy endpoint %q", item.EdgeKey, item.FromKey)
		}
		if _, exists := allianceKeys[item.ToKey]; !exists {
			return nil, fmt.Errorf("member_of %q has invalid alliance endpoint %q", item.EdgeKey, item.ToKey)
		}
		if _, resolved := wantCounts[item.ToKey]; !resolved {
			return nil, fmt.Errorf("member_of %q targets unresolved alliance %q", item.EdgeKey, item.ToKey)
		}
		if strings.TrimSpace(item.EdgeKey) == "" || strings.TrimSpace(item.SourceName) == "" || item.VerifiedAt != "2026-07-14" || !validHTTPSURL(item.SourceURL) {
			return nil, fmt.Errorf("member_of %q lacks approved provenance", item.EdgeKey)
		}
		if _, exists := keys[item.EdgeKey]; exists {
			return nil, fmt.Errorf("duplicate member_of edge key %q", item.EdgeKey)
		}
		tuple := item.FromKey + "\x00" + item.RelationType + "\x00" + item.ToKey
		if _, exists := tuples[tuple]; exists {
			return nil, fmt.Errorf("duplicate member_of tuple %q", tuple)
		}
		if source, exists := sources[item.ToKey]; exists && source != item.SourceURL {
			return nil, fmt.Errorf("member_of target %q uses multiple sources", item.ToKey)
		}
		if item.Action != "keep" && item.Action != "create" {
			return nil, fmt.Errorf("member_of %q has unsupported action %q", item.EdgeKey, item.Action)
		}
		keys[item.EdgeKey], tuples[tuple], sources[item.ToKey] = struct{}{}, struct{}{}, item.SourceURL
		counts[item.ToKey]++
		actions[item.Action]++
	}
	if len(counts) != len(wantCounts) {
		return nil, fmt.Errorf("resolved member_of targets = %d, want %d", len(counts), len(wantCounts))
	}
	for key, want := range wantCounts {
		if counts[key] != want {
			return nil, fmt.Errorf("member_of target %q count = %d, want %d", key, counts[key], want)
		}
	}
	if actions["keep"] != 31 || actions["create"] != 102 {
		return nil, fmt.Errorf("member_of actions = keep %d create %d, want 31/102", actions["keep"], actions["create"])
	}
	return keys, nil
}

func validateAliases(aliases []string) error {
	if len(aliases) > 64 {
		return fmt.Errorf("aliases exceed 64 items")
	}
	seen := map[string]struct{}{}
	for _, alias := range aliases {
		if alias == "" || alias != strings.TrimSpace(alias) {
			return fmt.Errorf("alias must be non-empty and trimmed")
		}
		if utf8.RuneCountInString(alias) > 128 {
			return fmt.Errorf("alias exceeds 128 characters")
		}
		normalized := normalizeAlias(alias)
		if _, exists := seen[normalized]; exists {
			return fmt.Errorf("duplicate alias after NFKC and casefold")
		}
		seen[normalized] = struct{}{}
	}
	return nil
}

func containsNormalized(values []string, want string) bool {
	normalizedWant := normalizeAlias(want)
	for _, value := range values {
		if normalizeAlias(value) == normalizedWant {
			return true
		}
	}
	return false
}

func normalizeAlias(value string) string {
	return cases.Fold().String(norm.NFKC.String(strings.TrimSpace(value)))
}

func validHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func canonicalAllianceChecksum(items []ApprovedAlliance) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{strconv.Itoa(item.SheetRow), item.Name, item.Profile.Abbreviation, item.Profile.LeadershipSummary, item.Profile.InfluenceScopeSummary, item.EntityKey, item.Action, item.Decision}, "\t"))
	}
	return canonicalLinesChecksum(lines)
}

func canonicalAllianceDetailsChecksum(items []ApprovedAlliance) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{strconv.Itoa(item.SheetRow), item.Name, item.Profile.Abbreviation, item.Profile.LeadershipSummary, item.Profile.InfluenceScopeSummary, item.EntityKey, item.Action, item.Decision, canonicalAliases(item.Aliases)}, "\t"))
	}
	return canonicalLinesChecksum(lines)
}

func canonicalEconomyChecksum(items []ApprovedEconomy) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{item.EntityKey, item.CountryCode, item.Name, item.EnglishName, item.CurrencyCode, item.Region, item.Action}, "\t"))
	}
	return canonicalLinesChecksum(lines)
}

func canonicalEconomyDetailsChecksum(items []ApprovedEconomy) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{item.EntityKey, item.CountryCode, item.Name, item.EnglishName, item.CurrencyCode, item.Region, item.Action, canonicalAliases(item.Aliases)}, "\t"))
	}
	return canonicalLinesChecksum(lines)
}

func canonicalMemberOfChecksum(items []ApprovedMemberOf) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{item.EdgeKey, item.FromKey, item.RelationType, item.ToKey, item.MembershipStatus, item.SourceURL, item.VerifiedAt, item.Action}, "\t"))
	}
	return canonicalLinesChecksum(lines)
}

func canonicalMemberOfDetailsChecksum(items []ApprovedMemberOf) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, strings.Join([]string{item.EdgeKey, item.FromKey, item.RelationType, item.ToKey, item.MembershipStatus, item.SourceName, item.SourceURL, item.VerifiedAt, item.Action}, "\t"))
	}
	return canonicalLinesChecksum(lines)
}

func canonicalLinesChecksum(lines []string) string {
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n") + "\n"))
	return hex.EncodeToString(sum[:])
}

func canonicalAliases(aliases []string) string {
	var result strings.Builder
	for _, alias := range aliases {
		result.WriteString(strconv.Itoa(len(alias)))
		result.WriteByte(':')
		result.WriteString(alias)
	}
	return result.String()
}
