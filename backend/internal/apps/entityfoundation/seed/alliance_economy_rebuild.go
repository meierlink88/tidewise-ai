package seed

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

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
}

type allianceEconomyRebuildPreflight struct {
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

func (r PostgresRepository) AuditAllianceEconomyRebuildDependencies(ctx context.Context) (AllianceEconomyDependencyReport, error) {
	return auditAllianceEconomyRebuildDependencies(ctx, r.db)
}

func auditAllianceEconomyRebuildDependencies(ctx context.Context, executor postgresExecutor) (AllianceEconomyDependencyReport, error) {
	if err := assertAllianceEconomyLocalDatabase(ctx, executor); err != nil {
		return AllianceEconomyDependencyReport{}, err
	}
	rows, err := executor.QueryContext(ctx, allianceEconomyDependencyCountsSQL())
	if err != nil {
		return AllianceEconomyDependencyReport{}, fmt.Errorf("query alliance economy dependency counts: %w", err)
	}
	var counts []AllianceEconomyDependencyCount
	for rows.Next() {
		var item AllianceEconomyDependencyCount
		if err := rows.Scan(&item.Scope, &item.RelationType, &item.FromType, &item.ToType, &item.RowCount); err != nil {
			rows.Close()
			return AllianceEconomyDependencyReport{}, fmt.Errorf("scan alliance economy dependency count: %w", err)
		}
		counts = append(counts, item)
	}
	if err := rows.Close(); err != nil {
		return AllianceEconomyDependencyReport{}, err
	}
	fingerprintRows, err := executor.QueryContext(ctx, allianceEconomyDependencyFingerprintsSQL())
	if err != nil {
		return AllianceEconomyDependencyReport{}, fmt.Errorf("query alliance economy dependency fingerprints: %w", err)
	}
	var fingerprints []string
	for fingerprintRows.Next() {
		var fingerprint string
		if err := fingerprintRows.Scan(&fingerprint); err != nil {
			fingerprintRows.Close()
			return AllianceEconomyDependencyReport{}, fmt.Errorf("scan alliance economy dependency fingerprint: %w", err)
		}
		fingerprints = append(fingerprints, fingerprint)
	}
	if err := fingerprintRows.Close(); err != nil {
		return AllianceEconomyDependencyReport{}, err
	}

	fkRows, err := executor.QueryContext(ctx, allianceEconomyForeignKeysSQL())
	if err != nil {
		return AllianceEconomyDependencyReport{}, fmt.Errorf("query entity node foreign keys: %w", err)
	}
	var foreignKeys []AllianceEconomyForeignKey
	for fkRows.Next() {
		var item AllianceEconomyForeignKey
		if err := fkRows.Scan(&item.TableName, &item.ColumnName, &item.ReferencedTable, &item.DeleteRule); err != nil {
			fkRows.Close()
			return AllianceEconomyDependencyReport{}, fmt.Errorf("scan entity node foreign key: %w", err)
		}
		foreignKeys = append(foreignKeys, item)
	}
	if err := fkRows.Close(); err != nil {
		return AllianceEconomyDependencyReport{}, err
	}
	protected, err := allianceEconomyCleanupProtectionFingerprints(ctx, executor)
	if err != nil {
		return AllianceEconomyDependencyReport{}, err
	}
	return buildAllianceEconomyDependencyReport(counts, foreignKeys, fingerprints, protected)
}

func buildAllianceEconomyDependencyReport(counts []AllianceEconomyDependencyCount, foreignKeys []AllianceEconomyForeignKey, fingerprints, protected []string) (AllianceEconomyDependencyReport, error) {
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
	report.ProtectedChecksum = allianceEconomyFingerprintChecksum(protected)
	return report, nil
}

func (r PostgresRepository) CleanupAllianceEconomyLocal(ctx context.Context, reviewedDependencyChecksum string) (AllianceEconomyCleanupResult, error) {
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("begin alliance economy cleanup: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`); err != nil {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("lock alliance economy cleanup scope: %w", err)
	}
	report, err := auditAllianceEconomyRebuildDependencies(ctx, tx)
	if err != nil {
		return AllianceEconomyCleanupResult{}, err
	}
	if report.Blocked {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("cross-domain alliance/economy dependencies require an explicit Review decision")
	}
	if strings.TrimSpace(reviewedDependencyChecksum) == "" || reviewedDependencyChecksum != report.Checksum {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("alliance/economy dependency snapshot differs from reviewed checksum")
	}
	var result AllianceEconomyCleanupResult
	if err := tx.QueryRowContext(ctx, allianceEconomyCleanupSQL()).Scan(&result.DeletedMemberOf, &result.DeletedAllianceProfiles, &result.DeletedAlliances); err != nil {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("cleanup alliance economy scope: %w", err)
	}
	if err := tx.QueryRowContext(ctx, allianceEconomyCleanupRemainingSQL()).Scan(&result.RemainingAlliances, &result.RemainingAllianceProfiles, &result.RemainingMemberOf, &result.RemainingEconomies, &result.RemainingEconomyProfiles); err != nil {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("query alliance economy cleanup remaining scope: %w", err)
	}
	if result.RemainingAlliances != 0 || result.RemainingAllianceProfiles != 0 || result.RemainingMemberOf != 0 || result.RemainingEconomies != 50 || result.RemainingEconomyProfiles != 50 {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("alliance/economy cleanup zero assertion failed: alliance=%d alliance_profile=%d member_of=%d economy=%d economy_profile=%d", result.RemainingAlliances, result.RemainingAllianceProfiles, result.RemainingMemberOf, result.RemainingEconomies, result.RemainingEconomyProfiles)
	}
	postReport, err := auditAllianceEconomyRebuildDependencies(ctx, tx)
	if err != nil {
		return AllianceEconomyCleanupResult{}, err
	}
	if postReport.Blocked || postReport.ProtectedChecksum != report.ProtectedChecksum {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("alliance/economy cleanup changed protected cross-domain facts")
	}
	if err := tx.Commit(); err != nil {
		return AllianceEconomyCleanupResult{}, fmt.Errorf("commit alliance economy cleanup: %w", err)
	}
	return result, nil
}

func (r PostgresRepository) RebuildApprovedAllianceEconomyLocal(ctx context.Context, manifest AllianceEconomyManifest) (AllianceEconomyRebuildResult, error) {
	if err := manifest.Validate(); err != nil {
		return AllianceEconomyRebuildResult{}, err
	}
	alliances, economies, memberOf, err := allianceEconomyRebuildPayloads(manifest)
	if err != nil {
		return AllianceEconomyRebuildResult{}, err
	}
	tx, err := r.root.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("begin alliance economy rebuild: %w", err)
	}
	defer tx.Rollback()
	if err := assertAllianceEconomyLocalDatabase(ctx, tx); err != nil {
		return AllianceEconomyRebuildResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `LOCK TABLE entity_nodes, entity_edges, alliance_org_profiles, economy_profiles IN EXCLUSIVE MODE`); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("lock alliance economy rebuild scope: %w", err)
	}
	var preflight allianceEconomyRebuildPreflight
	if err := tx.QueryRowContext(ctx, allianceEconomyRebuildPreflightSQL(), alliances, economies, memberOf).Scan(
		&preflight.SchemaReady, &preflight.IDConflicts, &preflight.KeyConflicts, &preflight.UnexpectedAllianceNodes, &preflight.UnexpectedAllianceEdges,
		&preflight.Alliances, &preflight.AllianceProfiles, &preflight.Economies, &preflight.EconomyProfiles, &preflight.NonTargetEconomies, &preflight.NonTargetEconomyProfiles, &preflight.MemberOf,
	); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("preflight approved alliance economy rebuild: %w", err)
	}
	if !preflight.SchemaReady {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("migration 000018 must be applied in the separately authorized local rebuild package")
	}
	if preflight.IDConflicts != 0 || preflight.KeyConflicts != 0 || preflight.UnexpectedAllianceNodes != 0 || preflight.UnexpectedAllianceEdges != 0 {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("alliance/economy rebuild identity or scope collision: %+v", preflight)
	}
	cleanupReady := preflight.Alliances == 0 && preflight.AllianceProfiles == 0 && preflight.Economies == 35 && preflight.EconomyProfiles == 35 && preflight.NonTargetEconomies == 15 && preflight.NonTargetEconomyProfiles == 15 && preflight.MemberOf == 0
	exact := preflight.Alliances == 45 && preflight.AllianceProfiles == 45 && preflight.Economies == 79 && preflight.EconomyProfiles == 79 && preflight.NonTargetEconomies == 15 && preflight.NonTargetEconomyProfiles == 15 && preflight.MemberOf == 133
	if !cleanupReady && !exact {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("alliance/economy rebuild requires scoped cleanup or an exact idempotent target: %+v", preflight)
	}
	protected, err := allianceEconomyRebuildProtectionFingerprints(ctx, tx, economies)
	if err != nil {
		return AllianceEconomyRebuildResult{}, err
	}
	protectedChecksum := allianceEconomyFingerprintChecksum(protected)
	if exact {
		var existing AllianceEconomyRebuildResult
		if err := tx.QueryRowContext(ctx, allianceEconomyExactQuerySQL(), alliances, economies, memberOf).Scan(&existing.Alliances, &existing.AllianceProfiles, &existing.Economies, &existing.EconomyProfiles, &existing.MemberOf, &existing.NonTargetEconomies, &existing.NonTargetEconomyProfiles, &existing.Orphans, &existing.DuplicateTuples, &existing.Mismatches); err != nil {
			return AllianceEconomyRebuildResult{}, fmt.Errorf("verify idempotent alliance economy target: %w", err)
		}
		if existing != (AllianceEconomyRebuildResult{Alliances: 45, AllianceProfiles: 45, Economies: 79, EconomyProfiles: 79, MemberOf: 133, NonTargetEconomies: 15, NonTargetEconomyProfiles: 15}) {
			return AllianceEconomyRebuildResult{}, fmt.Errorf("existing alliance/economy target is not an exact idempotent match: %+v", existing)
		}
	}
	var result AllianceEconomyRebuildResult
	if _, err := tx.ExecContext(ctx, allianceEconomyEntityRebuildSQL(), alliances, economies); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("rebuild approved alliance and economy entities: %w", err)
	}
	if _, err := tx.ExecContext(ctx, allianceEconomyProfileRebuildSQL(), alliances, economies); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("rebuild approved alliance and economy profiles: %w", err)
	}
	if _, err := tx.ExecContext(ctx, allianceEconomyMemberRebuildSQL(), memberOf); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("rebuild approved member_of relationships: %w", err)
	}
	if err := tx.QueryRowContext(ctx, allianceEconomyExactQuerySQL(), alliances, economies, memberOf).Scan(&result.Alliances, &result.AllianceProfiles, &result.Economies, &result.EconomyProfiles, &result.MemberOf, &result.NonTargetEconomies, &result.NonTargetEconomyProfiles, &result.Orphans, &result.DuplicateTuples, &result.Mismatches); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("query rebuilt alliance economy manifest: %w", err)
	}
	if result != (AllianceEconomyRebuildResult{Alliances: 45, AllianceProfiles: 45, Economies: 79, EconomyProfiles: 79, MemberOf: 133, NonTargetEconomies: 15, NonTargetEconomyProfiles: 15}) {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("alliance/economy rebuild exact assertion failed: %+v", result)
	}
	postProtected, err := allianceEconomyRebuildProtectionFingerprints(ctx, tx, economies)
	if err != nil {
		return AllianceEconomyRebuildResult{}, err
	}
	if allianceEconomyFingerprintChecksum(postProtected) != protectedChecksum {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("alliance/economy rebuild changed protected cross-domain facts")
	}
	result.ManifestChecksum = approvedAllianceEconomyManifestSHA256
	if err := tx.Commit(); err != nil {
		return AllianceEconomyRebuildResult{}, fmt.Errorf("commit alliance economy rebuild: %w", err)
	}
	return result, nil
}

func allianceEconomyRebuildPayloads(manifest AllianceEconomyManifest) ([]byte, []byte, []byte, error) {
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
	a, err := json.Marshal(alliances)
	if err != nil {
		return nil, nil, nil, err
	}
	e, err := json.Marshal(economies)
	if err != nil {
		return nil, nil, nil, err
	}
	m, err := json.Marshal(edges)
	if err != nil {
		return nil, nil, nil, err
	}
	return a, e, m, nil
}

func allianceEconomyDependencyCountsSQL() string {
	return `SELECT scope, relation_type, from_type, to_type, row_count FROM (
    SELECT 'entity_nodes'::text AS scope, ''::text AS relation_type, entity_type::text AS from_type, ''::text AS to_type, count(*)::bigint AS row_count
    FROM entity_nodes WHERE entity_type IN ('alliance_org','economy') GROUP BY entity_type
    UNION ALL
    SELECT 'alliance_org_profiles', '', '', '', count(*) FROM alliance_org_profiles
    UNION ALL
    SELECT 'economy_profiles', '', '', '', count(*) FROM economy_profiles
    UNION ALL
    SELECT 'entity_edges', e.relation_type, f.entity_type, t.entity_type, count(*)
    FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id
    WHERE f.entity_type IN ('alliance_org','economy') OR t.entity_type IN ('alliance_org','economy')
    GROUP BY e.relation_type,f.entity_type,t.entity_type
    UNION ALL
    SELECT 'indirect_index_profiles', '', 'economy', 'index', count(*)
    FROM index_profiles i JOIN market_profiles m ON m.entity_id=i.market_entity_id JOIN entity_nodes n ON n.id=m.economy_entity_id WHERE n.entity_type='economy'
    UNION ALL
    SELECT 'indirect_market_edges', e.relation_type, 'market', t.entity_type, count(*)
    FROM entity_edges e JOIN market_profiles m ON m.entity_id=e.from_entity_id JOIN entity_nodes n ON n.id=m.economy_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id
    WHERE n.entity_type='economy' AND e.relation_type IN ('tracks_index','observes_benchmark') GROUP BY e.relation_type,t.entity_type
) dependencies ORDER BY scope,relation_type,from_type,to_type`
}

func allianceEconomyDependencyFingerprintsSQL() string {
	return `WITH target AS (SELECT id FROM entity_nodes WHERE entity_type IN ('alliance_org','economy'))
SELECT fingerprint FROM (
  SELECT 'node|'||id||'|'||entity_key||'|'||entity_type||'|'||layer_code||'|'||name||'|'||canonical_name||'|'||aliases::text||'|'||status AS fingerprint FROM entity_nodes WHERE id IN (SELECT id FROM target)
  UNION ALL SELECT 'alliance_profile|'||p.entity_id||'|'||p.org_code||'|'||p.org_type||'|'||p.primary_domain||'|'||p.scope_region||'|'||p.official_url FROM alliance_org_profiles p
  UNION ALL SELECT 'economy_profile|'||p.entity_id||'|'||p.country_code||'|'||p.currency_code||'|'||p.region FROM economy_profiles p
  UNION ALL SELECT 'edge|'||e.id||'|'||f.entity_key||'|'||e.relation_type||'|'||t.entity_key||'|'||e.status||'|'||e.source_name||'|'||e.source_url||'|'||COALESCE(e.verified_at::text,'')
    FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id
    WHERE e.from_entity_id IN (SELECT id FROM target) OR e.to_entity_id IN (SELECT id FROM target)
) dependencies ORDER BY fingerprint`
}

func allianceEconomyForeignKeysSQL() string {
	return `SELECT tc.table_name, kcu.column_name, ccu.table_name AS referenced_table, rc.delete_rule
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu ON kcu.constraint_name=tc.constraint_name AND kcu.constraint_schema=tc.constraint_schema
JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name=tc.constraint_name AND ccu.constraint_schema=tc.constraint_schema
JOIN information_schema.referential_constraints rc ON rc.constraint_name=tc.constraint_name AND rc.constraint_schema=tc.constraint_schema
WHERE tc.constraint_type='FOREIGN KEY' AND tc.table_schema=current_schema() AND ccu.table_name='entity_nodes'
ORDER BY tc.table_name,kcu.column_name`
}

func allianceEconomyCleanupProtectionFingerprints(ctx context.Context, executor postgresExecutor) ([]string, error) {
	return allianceEconomyFingerprintRows(ctx, executor, allianceEconomyCleanupProtectionSQL())
}

func allianceEconomyRebuildProtectionFingerprints(ctx context.Context, executor postgresExecutor, economies []byte) ([]string, error) {
	return allianceEconomyFingerprintRows(ctx, executor, allianceEconomyRebuildProtectionSQL(), economies)
}

func allianceEconomyFingerprintRows(ctx context.Context, executor postgresExecutor, statement string, args ...any) ([]string, error) {
	rows, err := executor.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("query alliance/economy protected fingerprints: %w", err)
	}
	defer rows.Close()
	var fingerprints []string
	for rows.Next() {
		var fingerprint string
		if err := rows.Scan(&fingerprint); err != nil {
			return nil, fmt.Errorf("scan alliance/economy protected fingerprint: %w", err)
		}
		fingerprints = append(fingerprints, fingerprint)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fingerprints, nil
}

func allianceEconomyFingerprintChecksum(fingerprints []string) string {
	ordered := append([]string(nil), fingerprints...)
	sort.Strings(ordered)
	sum := sha256.Sum256([]byte(strings.Join(ordered, "\n")))
	return hex.EncodeToString(sum[:])
}

func allianceEconomyCleanupProtectionSQL() string {
	return `SELECT fingerprint FROM (
  SELECT 'economy_node|'||to_jsonb(n)::text AS fingerprint FROM entity_nodes n WHERE n.entity_type='economy'
  UNION ALL SELECT 'economy_profile|'||to_jsonb(p)::text FROM economy_profiles p
  UNION ALL SELECT 'economy_edge|'||to_jsonb(e)::text FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id
    WHERE (f.entity_type='economy' OR t.entity_type='economy') AND NOT (e.relation_type='member_of' AND f.entity_type='economy' AND t.entity_type='alliance_org')
  UNION ALL SELECT 'market_profile|'||to_jsonb(p)::text FROM market_profiles p JOIN entity_nodes n ON n.id=p.economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'company_profile|'||to_jsonb(p)::text FROM company_profiles p JOIN entity_nodes n ON n.id=p.registration_economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'person_profile|'||to_jsonb(p)::text FROM person_profiles p JOIN entity_nodes n ON n.id=p.economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'index_profile|'||to_jsonb(i)::text FROM index_profiles i JOIN market_profiles m ON m.entity_id=i.market_entity_id JOIN entity_nodes n ON n.id=m.economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'market_edge|'||to_jsonb(e)::text FROM entity_edges e JOIN market_profiles m ON m.entity_id=e.from_entity_id JOIN entity_nodes n ON n.id=m.economy_entity_id WHERE n.entity_type='economy' AND e.relation_type IN ('tracks_index','observes_benchmark')
  UNION ALL SELECT 'benchmark_profile|'||to_jsonb(p)::text FROM benchmark_profiles p
) protected ORDER BY fingerprint`
}

func allianceEconomyRebuildProtectionSQL() string {
	return `WITH economy_input AS (SELECT * FROM jsonb_to_recordset($1::jsonb) AS x("ID" uuid,"EntityKey" text))
SELECT fingerprint FROM (
  SELECT 'non_target_economy_node|'||to_jsonb(n)::text AS fingerprint FROM entity_nodes n WHERE n.entity_type='economy' AND NOT EXISTS (SELECT 1 FROM economy_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)
  UNION ALL SELECT 'non_target_economy_profile|'||to_jsonb(p)::text FROM economy_profiles p JOIN entity_nodes n ON n.id=p.entity_id WHERE NOT EXISTS (SELECT 1 FROM economy_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)
  UNION ALL SELECT 'economy_edge|'||to_jsonb(e)::text FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id
    WHERE (f.entity_type='economy' OR t.entity_type='economy') AND NOT (e.relation_type='member_of' AND f.entity_type='economy' AND t.entity_type='alliance_org')
  UNION ALL SELECT 'market_profile|'||to_jsonb(p)::text FROM market_profiles p JOIN entity_nodes n ON n.id=p.economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'company_profile|'||to_jsonb(p)::text FROM company_profiles p JOIN entity_nodes n ON n.id=p.registration_economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'person_profile|'||to_jsonb(p)::text FROM person_profiles p JOIN entity_nodes n ON n.id=p.economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'index_profile|'||to_jsonb(i)::text FROM index_profiles i JOIN market_profiles m ON m.entity_id=i.market_entity_id JOIN entity_nodes n ON n.id=m.economy_entity_id WHERE n.entity_type='economy'
  UNION ALL SELECT 'market_edge|'||to_jsonb(e)::text FROM entity_edges e JOIN market_profiles m ON m.entity_id=e.from_entity_id JOIN entity_nodes n ON n.id=m.economy_entity_id WHERE n.entity_type='economy' AND e.relation_type IN ('tracks_index','observes_benchmark')
  UNION ALL SELECT 'benchmark_profile|'||to_jsonb(p)::text FROM benchmark_profiles p
) protected ORDER BY fingerprint`
}

func allianceEconomyCleanupSQL() string {
	return `WITH deleted_member_of AS (
    DELETE FROM entity_edges e USING entity_nodes f, entity_nodes t
    WHERE e.from_entity_id=f.id AND e.to_entity_id=t.id
      AND e.relation_type='member_of' AND f.entity_type='economy' AND t.entity_type='alliance_org'
    RETURNING 1
), deleted_alliance_profiles AS (DELETE FROM alliance_org_profiles RETURNING 1),
deleted_alliances AS (DELETE FROM entity_nodes WHERE entity_type='alliance_org' RETURNING 1)
SELECT (SELECT count(*) FROM deleted_member_of), (SELECT count(*) FROM deleted_alliance_profiles),
       (SELECT count(*) FROM deleted_alliances)`
}

func allianceEconomyCleanupRemainingSQL() string {
	return `SELECT
  (SELECT count(*) FROM entity_nodes WHERE entity_type='alliance_org'),
  (SELECT count(*) FROM alliance_org_profiles),
  (SELECT count(*) FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id WHERE e.relation_type='member_of' AND f.entity_type='economy' AND t.entity_type='alliance_org'),
  (SELECT count(*) FROM entity_nodes WHERE entity_type='economy'),
  (SELECT count(*) FROM economy_profiles)`
}

func allianceEconomyEntityRebuildSQL() string {
	return `WITH alliance_input AS (
    SELECT * FROM jsonb_to_recordset($1::jsonb) AS x("ID" uuid,"EntityKey" text,"Name" text,"Aliases" text[])
), economy_input AS (
    SELECT * FROM jsonb_to_recordset($2::jsonb) AS x("ID" uuid,"EntityKey" text,"Name" text,"Aliases" text[])
), upsert_alliances AS (
    INSERT INTO entity_nodes(id,entity_key,entity_type,layer_code,name,canonical_name,aliases,status)
    SELECT "ID","EntityKey",'alliance_org','alliance',"Name","Name","Aliases",'active' FROM alliance_input
    ON CONFLICT(id) DO UPDATE SET entity_key=EXCLUDED.entity_key,entity_type=EXCLUDED.entity_type,layer_code=EXCLUDED.layer_code,name=EXCLUDED.name,canonical_name=EXCLUDED.canonical_name,aliases=EXCLUDED.aliases,status='active',updated_at=now() RETURNING 1
), upsert_economies AS (
    INSERT INTO entity_nodes(id,entity_key,entity_type,layer_code,name,canonical_name,aliases,status)
    SELECT "ID","EntityKey",'economy','economy',"Name","Name","Aliases",'active' FROM economy_input
    ON CONFLICT(id) DO UPDATE SET entity_key=EXCLUDED.entity_key,entity_type=EXCLUDED.entity_type,layer_code=EXCLUDED.layer_code,name=EXCLUDED.name,canonical_name=EXCLUDED.canonical_name,aliases=EXCLUDED.aliases,status='active',updated_at=now() RETURNING 1
)
SELECT (SELECT count(*) FROM upsert_alliances)+(SELECT count(*) FROM upsert_economies)`
}

func allianceEconomyProfileRebuildSQL() string {
	return `WITH alliance_input AS (
    SELECT * FROM jsonb_to_recordset($1::jsonb) AS x("ID" uuid,"Abbreviation" text,"LeadershipSummary" text,"InfluenceScopeSummary" text)
), economy_input AS (
    SELECT * FROM jsonb_to_recordset($2::jsonb) AS x("ID" uuid,"CountryCode" text,"CurrencyCode" text,"Region" text)
), upsert_alliance_profiles AS (
    INSERT INTO alliance_org_profiles(entity_id,abbreviation,leadership_summary,influence_scope_summary)
    SELECT "ID","Abbreviation","LeadershipSummary","InfluenceScopeSummary" FROM alliance_input
    ON CONFLICT(entity_id) DO UPDATE SET abbreviation=EXCLUDED.abbreviation,leadership_summary=EXCLUDED.leadership_summary,influence_scope_summary=EXCLUDED.influence_scope_summary RETURNING 1
), upsert_economy_profiles AS (
    INSERT INTO economy_profiles(entity_id,country_code,currency_code,region)
    SELECT "ID","CountryCode","CurrencyCode","Region" FROM economy_input
    ON CONFLICT(entity_id) DO UPDATE SET country_code=EXCLUDED.country_code,currency_code=EXCLUDED.currency_code,region=EXCLUDED.region RETURNING 1
)
SELECT (SELECT count(*) FROM upsert_alliance_profiles)+(SELECT count(*) FROM upsert_economy_profiles)`
}

func allianceEconomyMemberRebuildSQL() string {
	return `WITH edge_input AS (
    SELECT * FROM jsonb_to_recordset($1::jsonb) AS x("ID" uuid,"FromID" uuid,"ToID" uuid,"SourceName" text,"SourceURL" text,"VerifiedAt" text)
), upsert_edges AS (
    INSERT INTO entity_edges(id,from_entity_id,to_entity_id,relation_type,evidence_note,status,source_name,source_url,verified_at)
    SELECT "ID","FromID","ToID",'member_of','formal_active','active',"SourceName","SourceURL","VerifiedAt"::timestamptz FROM edge_input
    ON CONFLICT(id) DO UPDATE SET from_entity_id=EXCLUDED.from_entity_id,to_entity_id=EXCLUDED.to_entity_id,relation_type='member_of',evidence_note='formal_active',status='active',source_name=EXCLUDED.source_name,source_url=EXCLUDED.source_url,verified_at=EXCLUDED.verified_at,updated_at=now() RETURNING 1
)
SELECT count(*) FROM upsert_edges`
}

func allianceEconomyRebuildPreflightSQL() string {
	return `WITH alliance_input AS (
    SELECT * FROM jsonb_to_recordset($1::jsonb) AS x("ID" uuid,"EntityKey" text)
), economy_input AS (
    SELECT * FROM jsonb_to_recordset($2::jsonb) AS x("ID" uuid,"EntityKey" text)
), edge_input AS (
    SELECT * FROM jsonb_to_recordset($3::jsonb) AS x("ID" uuid,"FromID" uuid,"ToID" uuid)
), entity_input AS (
    SELECT "ID","EntityKey",'alliance_org'::text AS entity_type FROM alliance_input
    UNION ALL SELECT "ID","EntityKey",'economy' FROM economy_input
)
SELECT
  EXISTS (SELECT 1 FROM goose_db_version WHERE version_id=18 AND is_applied) AS schema_ready,
  (SELECT count(*) FROM entity_input i JOIN entity_nodes n ON n.id=i."ID" WHERE n.entity_key IS DISTINCT FROM i."EntityKey" OR n.entity_type IS DISTINCT FROM i.entity_type) AS id_conflicts,
  (SELECT count(*) FROM entity_input i JOIN entity_nodes n ON n.entity_key=i."EntityKey" WHERE n.id IS DISTINCT FROM i."ID") AS key_conflicts,
  (SELECT count(*) FROM entity_nodes n WHERE n.entity_type='alliance_org' AND NOT EXISTS (SELECT 1 FROM alliance_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)) AS unexpected_alliance_nodes,
  (SELECT count(*) FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id
    WHERE (f.entity_type='alliance_org' OR t.entity_type='alliance_org') AND NOT EXISTS (SELECT 1 FROM edge_input i WHERE i."ID"=e.id AND i."FromID"=e.from_entity_id AND i."ToID"=e.to_entity_id AND e.relation_type='member_of')) AS unexpected_alliance_edges,
  (SELECT count(*) FROM entity_nodes WHERE entity_type='alliance_org') AS alliances,
  (SELECT count(*) FROM alliance_org_profiles) AS alliance_profiles,
  (SELECT count(*) FROM entity_nodes n JOIN economy_input i ON i."ID"=n.id AND i."EntityKey"=n.entity_key WHERE n.entity_type='economy') AS economies,
  (SELECT count(*) FROM economy_profiles p JOIN economy_input i ON i."ID"=p.entity_id) AS economy_profiles,
  (SELECT count(*) FROM entity_nodes n WHERE n.entity_type='economy' AND NOT EXISTS (SELECT 1 FROM economy_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)) AS non_target_economies,
  (SELECT count(*) FROM economy_profiles p JOIN entity_nodes n ON n.id=p.entity_id WHERE n.entity_type='economy' AND NOT EXISTS (SELECT 1 FROM economy_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)) AS non_target_economy_profiles,
  (SELECT count(*) FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id WHERE e.relation_type='member_of' AND f.entity_type='economy' AND t.entity_type='alliance_org') AS member_of`
}

func allianceEconomyExactQuerySQL() string {
	return `WITH alliance_input AS (
    SELECT * FROM jsonb_to_recordset($1::jsonb) AS x("ID" uuid,"EntityKey" text,"Name" text,"Aliases" text[],"Abbreviation" text,"LeadershipSummary" text,"InfluenceScopeSummary" text)
), economy_input AS (
    SELECT * FROM jsonb_to_recordset($2::jsonb) AS x("ID" uuid,"EntityKey" text,"Name" text,"Aliases" text[],"CountryCode" text,"CurrencyCode" text,"Region" text)
), edge_input AS (
    SELECT * FROM jsonb_to_recordset($3::jsonb) AS x("ID" uuid,"EdgeKey" text,"FromID" uuid,"ToID" uuid,"SourceName" text,"SourceURL" text,"VerifiedAt" text)
), actual AS (
    SELECT
      (SELECT count(*) FROM entity_nodes WHERE entity_type='alliance_org' AND status='active') AS alliances,
      (SELECT count(*) FROM alliance_org_profiles) AS alliance_profiles,
      (SELECT count(*) FROM entity_nodes n JOIN economy_input i ON i."ID"=n.id AND i."EntityKey"=n.entity_key WHERE n.entity_type='economy' AND n.status='active') AS economies,
      (SELECT count(*) FROM economy_profiles p JOIN economy_input i ON i."ID"=p.entity_id) AS economy_profiles,
      (SELECT count(*) FROM entity_nodes n WHERE n.entity_type='economy' AND NOT EXISTS (SELECT 1 FROM economy_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)) AS non_target_economies,
      (SELECT count(*) FROM economy_profiles p JOIN entity_nodes n ON n.id=p.entity_id WHERE n.entity_type='economy' AND NOT EXISTS (SELECT 1 FROM economy_input i WHERE i."ID"=n.id AND i."EntityKey"=n.entity_key)) AS non_target_economy_profiles,
      (SELECT count(*) FROM entity_edges e JOIN entity_nodes f ON f.id=e.from_entity_id JOIN entity_nodes t ON t.id=e.to_entity_id WHERE e.relation_type='member_of' AND e.status='active' AND f.entity_type='economy' AND t.entity_type='alliance_org') AS member_of,
      (SELECT count(*) FROM entity_edges e LEFT JOIN entity_nodes f ON f.id=e.from_entity_id LEFT JOIN entity_nodes t ON t.id=e.to_entity_id WHERE f.id IS NULL OR t.id IS NULL) AS orphans,
      (SELECT count(*) FROM (SELECT from_entity_id,to_entity_id,relation_type FROM entity_edges WHERE relation_type='member_of' AND status='active' GROUP BY 1,2,3 HAVING count(*)>1) d) AS duplicate_tuples,
      (SELECT count(*) FROM entity_nodes n WHERE n.entity_type='alliance_org' AND NOT EXISTS (SELECT 1 FROM alliance_input a WHERE a."ID"=n.id AND a."EntityKey"=n.entity_key))
      + (SELECT count(*) FROM alliance_input i
         LEFT JOIN entity_nodes n ON n.id=i."ID"
         LEFT JOIN alliance_org_profiles p ON p.entity_id=i."ID"
         WHERE n.id IS NULL OR n.entity_key IS DISTINCT FROM i."EntityKey" OR n.entity_type<>'alliance_org' OR n.layer_code<>'alliance'
            OR n.name IS DISTINCT FROM i."Name" OR n.canonical_name IS DISTINCT FROM i."Name" OR n.aliases IS DISTINCT FROM i."Aliases" OR n.status<>'active'
            OR p.entity_id IS NULL OR p.abbreviation IS DISTINCT FROM i."Abbreviation" OR p.leadership_summary IS DISTINCT FROM i."LeadershipSummary"
            OR p.influence_scope_summary IS DISTINCT FROM i."InfluenceScopeSummary")
      + (SELECT count(*) FROM economy_input i
         LEFT JOIN entity_nodes n ON n.id=i."ID"
         LEFT JOIN economy_profiles p ON p.entity_id=i."ID"
         WHERE n.id IS NULL OR n.entity_key IS DISTINCT FROM i."EntityKey" OR n.entity_type<>'economy' OR n.layer_code<>'economy'
            OR n.name IS DISTINCT FROM i."Name" OR n.canonical_name IS DISTINCT FROM i."Name" OR n.aliases IS DISTINCT FROM i."Aliases" OR n.status<>'active'
            OR p.entity_id IS NULL OR p.country_code IS DISTINCT FROM i."CountryCode" OR p.currency_code IS DISTINCT FROM i."CurrencyCode"
            OR p.region IS DISTINCT FROM i."Region")
      + (SELECT count(*) FROM edge_input i LEFT JOIN entity_edges e ON e.id=i."ID"
         WHERE e.id IS NULL OR e.from_entity_id IS DISTINCT FROM i."FromID" OR e.to_entity_id IS DISTINCT FROM i."ToID"
            OR e.relation_type<>'member_of' OR e.evidence_note<>'formal_active' OR e.status<>'active'
            OR e.source_name IS DISTINCT FROM i."SourceName" OR e.source_url IS DISTINCT FROM i."SourceURL"
            OR e.verified_at::date IS DISTINCT FROM i."VerifiedAt"::date) AS mismatches
)
SELECT alliances,alliance_profiles,economies,economy_profiles,member_of,non_target_economies,non_target_economy_profiles,orphans,duplicate_tuples,mismatches FROM actual`
}

func assertAllianceEconomyLocalDatabase(ctx context.Context, executor postgresExecutor) error {
	var databaseName string
	if err := executor.QueryRowContext(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		return fmt.Errorf("read alliance/economy database identity: %w", err)
	}
	if databaseName != "tidewise_local" {
		return fmt.Errorf("alliance/economy cleanup and rebuild require database tidewise_local, got %q", databaseName)
	}
	return nil
}
