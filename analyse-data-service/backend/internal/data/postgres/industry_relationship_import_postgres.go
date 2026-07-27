package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
)

func (r repository) InIndustryRelationshipImportTransaction(
	ctx context.Context,
	fn func(biz.Transaction) error,
) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin Industry relationship import transaction: %w", err)
	}
	wrapper := &postgresIndustryRelationshipImportTx{tx: tx}
	if err := fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Industry relationship import transaction: %w", err)
	}
	return nil
}

type postgresIndustryRelationshipImportTx struct {
	tx *sql.Tx
}

func (t *postgresIndustryRelationshipImportTx) LockIndustryRelationshipPackage(
	ctx context.Context,
	packageSHA string,
) error {
	_, err := t.tx.ExecContext(
		ctx,
		`SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`,
		"industry_relationship_import:"+packageSHA,
	)
	return err
}

func (t *postgresIndustryRelationshipImportTx) IndustryRelationshipImportReceipt(
	ctx context.Context,
	packageSHA string,
) (*biz.Receipt, error) {
	var receipt biz.Receipt
	var countsJSON []byte
	err := t.tx.QueryRowContext(ctx, `SELECT id::text, package_sha256, manifest_sha256,
       relation_spec_sha256, approval_basis, package_counts, caller_subject, imported_at
FROM industry_relationship_import_receipts
WHERE package_sha256 = $1`, packageSHA).Scan(
		&receipt.ID, &receipt.PackageSHA256, &receipt.ManifestSHA256,
		&receipt.RelationSpecSHA256, &receipt.ApprovalBasis, &countsJSON,
		&receipt.CallerSubject, &receipt.ImportedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(countsJSON, &receipt.PackageCounts); err != nil {
		return nil, fmt.Errorf("decode Industry relationship receipt counts: %w", err)
	}
	return &receipt, nil
}

func (t *postgresIndustryRelationshipImportTx) PreflightIndustryRelationshipPackage(
	ctx context.Context,
	pkg biz.Package,
) error {
	var receiptTableExists bool
	if err := t.tx.QueryRowContext(
		ctx,
		`SELECT to_regclass('industry_relationship_import_receipts') IS NOT NULL`,
	).Scan(&receiptTableExists); err != nil {
		return err
	}
	if !receiptTableExists {
		return errors.New("migration 000030 is not applied")
	}
	if err := t.resolveExistingMappingTargets(ctx, pkg.IndustryMappings, "industry", "industry_profiles"); err != nil {
		return err
	}
	if err := t.resolveExistingMappingTargets(ctx, pkg.ConceptMappings, "concept", "concept_profiles"); err != nil {
		return err
	}
	if err := t.resolveExistingChainNodes(ctx, pkg); err != nil {
		return err
	}

	entityIDs := make([]string, 0, len(pkg.IndustryChains)+len(pkg.ChainNodeAdditions))
	entityKeys := make([]string, 0, cap(entityIDs))
	for _, item := range pkg.ChainNodeAdditions {
		entityIDs = append(entityIDs, item.EntityID)
		entityKeys = append(entityKeys, item.EntityKey)
	}
	for _, item := range pkg.IndustryChains {
		entityIDs = append(entityIDs, item.EntityID)
		entityKeys = append(entityKeys, item.EntityKey)
	}
	var conflictCount int
	if err := t.tx.QueryRowContext(ctx, `SELECT count(*) FROM entity_nodes
WHERE id = ANY($1::uuid[]) OR entity_key = ANY($2::text[])`, entityIDs, entityKeys).Scan(&conflictCount); err != nil {
		return fmt.Errorf("check new entity conflicts: %w", err)
	}
	if conflictCount != 0 {
		return fmt.Errorf("%d Industry Chain/Chain Node entity identities already exist without this package receipt", conflictCount)
	}
	for _, relation := range pkg.GlobalRelations {
		var existingID string
		err := t.tx.QueryRowContext(ctx, `SELECT id::text FROM chain_node_relations
WHERE id = $1::uuid OR (
    from_chain_node_entity_id = $2::uuid
    AND to_chain_node_entity_id = $3::uuid
    AND relation_type = $4
) LIMIT 1`, relation.ID, relation.FromChainNodeEntityID, relation.ToChainNodeEntityID, relation.RelationType).Scan(&existingID)
		if err == nil {
			return fmt.Errorf("global Chain Node relation %s already exists without this package receipt", existingID)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check global Chain Node relation conflict: %w", err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) resolveExistingMappingTargets(
	ctx context.Context,
	items []biz.EntityMapping,
	entityType, profileTable string,
) error {
	expected := make(map[string]string, len(items))
	for _, item := range items {
		if previous, exists := expected[item.ToEntityID]; exists && previous != item.ToKey {
			return fmt.Errorf("%s UUID maps to conflicting keys %s and %s", entityType, previous, item.ToKey)
		}
		expected[item.ToEntityID] = item.ToKey
	}
	if len(expected) == 0 {
		return nil
	}
	ids := sortedMapKeys(expected)
	query := fmt.Sprintf(`SELECT n.id::text, n.entity_key
FROM entity_nodes n
JOIN %s p ON p.entity_id = n.id
WHERE n.id = ANY($1::uuid[])
  AND n.entity_type = $2
  AND n.status = 'active'
  AND p.review_status = 'approved'`, profileTable)
	rows, err := t.tx.QueryContext(ctx, query, ids, entityType)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[string]string, len(expected))
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return err
		}
		found[id] = key
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !reflect.DeepEqual(found, expected) {
		return fmt.Errorf("%s mapping endpoints do not resolve to approved/active canonical entities", entityType)
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) resolveExistingChainNodes(
	ctx context.Context,
	pkg biz.Package,
) error {
	expected, err := existingChainNodeEndpoints(pkg)
	if err != nil {
		return err
	}
	if len(expected) == 0 {
		return nil
	}
	ids := sortedMapKeys(expected)
	rows, err := t.tx.QueryContext(ctx, `SELECT n.id::text, n.entity_key
FROM entity_nodes n
JOIN chain_node_profiles p ON p.entity_id = n.id
WHERE n.id = ANY($1::uuid[])
  AND n.entity_type = 'chain_node'
  AND n.status = 'active'
  AND p.review_status = 'approved'`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[string]string, len(expected))
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return err
		}
		found[id] = key
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !reflect.DeepEqual(found, expected) {
		return errors.New("Chain Node endpoints do not resolve to approved/active canonical entities")
	}
	return nil
}

func existingChainNodeEndpoints(pkg biz.Package) (map[string]string, error) {
	additions := make(map[string]struct{}, len(pkg.ChainNodeAdditions))
	for _, item := range pkg.ChainNodeAdditions {
		additions[item.EntityID] = struct{}{}
	}
	expected := make(map[string]string)
	addExpected := func(id, key string) error {
		if _, isAddition := additions[id]; isAddition {
			return nil
		}
		if previous, exists := expected[id]; exists && previous != key {
			return fmt.Errorf("Chain Node UUID maps to conflicting keys %s and %s", previous, key)
		}
		expected[id] = key
		return nil
	}
	for _, item := range pkg.Memberships {
		if err := addExpected(item.ChainNodeEntityID, item.NodeKey); err != nil {
			return nil, err
		}
	}
	for _, item := range pkg.GlobalRelations {
		if err := addExpected(item.FromChainNodeEntityID, item.FromNodeKey); err != nil {
			return nil, err
		}
		if err := addExpected(item.ToChainNodeEntityID, item.ToNodeKey); err != nil {
			return nil, err
		}
	}
	return expected, nil
}

func (t *postgresIndustryRelationshipImportTx) InsertIndustryRelationshipPackage(
	ctx context.Context,
	pkg biz.Package,
) error {
	if err := t.insertChainNodeAdditions(ctx, pkg.ChainNodeAdditions); err != nil {
		return err
	}
	if err := t.insertIndustryChains(ctx, pkg.IndustryChains); err != nil {
		return err
	}
	if err := t.insertMappings(ctx, pkg.IndustryMappings); err != nil {
		return err
	}
	if err := t.insertMappings(ctx, pkg.ConceptMappings); err != nil {
		return err
	}
	if err := t.insertMemberships(ctx, pkg.Memberships); err != nil {
		return err
	}
	if err := t.insertGraphEdges(ctx, pkg.GraphEdges); err != nil {
		return err
	}
	return t.insertGlobalRelations(ctx, pkg.GlobalRelations)
}

func (t *postgresIndustryRelationshipImportTx) insertChainNodeAdditions(
	ctx context.Context,
	items []biz.ChainNodeAddition,
) error {
	entityStatement, err := t.tx.PrepareContext(ctx, `INSERT INTO entity_nodes (
    id, entity_type, layer_code, name, canonical_name, aliases, status, entity_key
) VALUES ($1::uuid,'chain_node',$2,$3,$4,$5,$6,$7)`)
	if err != nil {
		return err
	}
	defer entityStatement.Close()
	profileStatement, err := t.tx.PrepareContext(ctx, `INSERT INTO chain_node_profiles (
    entity_id, definition, boundary_note, review_status
) VALUES ($1::uuid,$2,$3,'approved')`)
	if err != nil {
		return err
	}
	defer profileStatement.Close()
	for _, item := range items {
		if _, err := entityStatement.ExecContext(
			ctx, item.EntityID, item.LayerCode, item.Name, item.CanonicalName,
			item.Aliases, item.Status, item.EntityKey,
		); err != nil {
			return fmt.Errorf("insert Chain Node %s: %w", item.EntityKey, err)
		}
		if _, err := profileStatement.ExecContext(
			ctx, item.EntityID, item.Definition, item.BoundaryNote,
		); err != nil {
			return fmt.Errorf("insert Chain Node profile %s: %w", item.EntityKey, err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) insertIndustryChains(
	ctx context.Context,
	items []biz.IndustryChain,
) error {
	entityStatement, err := t.tx.PrepareContext(ctx, `INSERT INTO entity_nodes (
    id, entity_type, layer_code, name, canonical_name, aliases, status, entity_key
) VALUES ($1::uuid,'industry_chain',$2,$3,$4,$5,$6,$7)`)
	if err != nil {
		return err
	}
	defer entityStatement.Close()
	definitionStatement, err := t.tx.PrepareContext(ctx, `INSERT INTO industry_chain_definitions (
    entity_id, scope, target_output, end_use, geography, as_of_date,
    review_status, review_note, technology_route_qualifier, observable_variables
) VALUES ($1::uuid,$2,$3,$4,$5,$6::date,'approved',$7,$8,$9)`)
	if err != nil {
		return err
	}
	defer definitionStatement.Close()
	for _, item := range items {
		if _, err := entityStatement.ExecContext(
			ctx, item.EntityID, item.LayerCode, item.Name, item.CanonicalName,
			item.Aliases, item.Status, item.EntityKey,
		); err != nil {
			return fmt.Errorf("insert Industry Chain %s: %w", item.EntityKey, err)
		}
		if _, err := definitionStatement.ExecContext(
			ctx, item.EntityID, item.Scope, item.TargetOutput, item.EndUse, item.Geography,
			item.AsOfDate, item.ReviewNote, nullablePointerString(item.TechnologyRouteQualifier),
			item.ObservableVariables,
		); err != nil {
			return fmt.Errorf("insert Industry Chain definition %s: %w", item.EntityKey, err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) insertMappings(
	ctx context.Context,
	items []biz.EntityMapping,
) error {
	statement, err := t.tx.PrepareContext(ctx, `INSERT INTO entity_edges (
    id, from_entity_id, to_entity_id, relation_type, evidence_note, status
) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, item := range items {
		if _, err := statement.ExecContext(
			ctx, item.RelationID, item.FromEntityID, item.ToEntityID,
			item.RelationType, item.EvidenceNote, item.Status,
		); err != nil {
			return fmt.Errorf("insert mapping %s: %w", item.RelationKey, err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) insertMemberships(
	ctx context.Context,
	items []biz.Membership,
) error {
	statement, err := t.tx.PrepareContext(ctx, `INSERT INTO industry_chain_node_memberships (
    industry_chain_entity_id, chain_node_entity_id, position, contextual_stage,
    review_status, status, inclusion_reason, evidence_ids, source_name, source_url, verified_at
) VALUES ($1::uuid,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, item := range items {
		if _, err := statement.ExecContext(
			ctx, item.IndustryChainEntityID, item.ChainNodeEntityID, item.Position,
			item.ContextualStage, item.ReviewStatus, item.Status, item.InclusionReason,
			item.EvidenceIDs, item.SourceName, item.SourceURL, item.VerifiedAt,
		); err != nil {
			return fmt.Errorf("insert membership %s: %w", item.RelationKey, err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) insertGraphEdges(
	ctx context.Context,
	items []biz.GraphEdge,
) error {
	statement, err := t.tx.PrepareContext(ctx, `INSERT INTO industry_chain_graph_edges (
    id, industry_chain_entity_id, from_chain_node_entity_id, to_chain_node_entity_id,
    relation_type, mechanism, condition_note, segment_kind, omitted_step_note,
    review_status, status, evidence_ids, source_name, source_url, verified_at
) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, item := range items {
		if _, err := statement.ExecContext(
			ctx, item.ID, item.IndustryChainEntityID, item.FromChainNodeEntityID,
			item.ToChainNodeEntityID, item.RelationType, item.Mechanism,
			nullablePointerString(item.ConditionNote), item.SegmentKind,
			nullablePointerString(item.OmittedStepNote), item.ReviewStatus, item.Status,
			item.EvidenceIDs, item.SourceName, item.SourceURL, item.VerifiedAt,
		); err != nil {
			return fmt.Errorf("insert graph edge %s: %w", item.RelationKey, err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) insertGlobalRelations(
	ctx context.Context,
	items []biz.GlobalChainNodeRelation,
) error {
	statement, err := t.tx.PrepareContext(ctx, `INSERT INTO chain_node_relations (
    id, from_chain_node_entity_id, to_chain_node_entity_id, relation_type,
    mechanism, condition_note, evidence_note, provenance, verified_at, status
) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9,$10)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, item := range items {
		if _, err := statement.ExecContext(
			ctx, item.ID, item.FromChainNodeEntityID, item.ToChainNodeEntityID,
			item.RelationType, item.Mechanism, nullablePointerString(item.ConditionNote),
			item.EvidenceNote, item.Provenance, item.VerifiedAt, item.Status,
		); err != nil {
			return fmt.Errorf("insert global Chain Node relation %s: %w", item.RelationKey, err)
		}
	}
	return nil
}

func (t *postgresIndustryRelationshipImportTx) VerifyIndustryRelationshipPackage(
	ctx context.Context,
	pkg biz.Package,
) error {
	if err := t.resolveExistingMappingTargets(ctx, pkg.IndustryMappings, "industry", "industry_profiles"); err != nil {
		return fmt.Errorf("verify Industry mapping targets: %w", err)
	}
	if err := t.resolveExistingMappingTargets(ctx, pkg.ConceptMappings, "concept", "concept_profiles"); err != nil {
		return fmt.Errorf("verify Concept mapping targets: %w", err)
	}
	if err := t.resolveExistingChainNodes(ctx, pkg); err != nil {
		return fmt.Errorf("verify existing Chain Node endpoints: %w", err)
	}
	checks := []struct {
		label string
		query string
		ids   []string
		want  int
	}{
		{
			label: "Industry Chain entities",
			query: `SELECT count(*) FROM entity_nodes n
JOIN industry_chain_definitions d ON d.entity_id=n.id
WHERE n.id=ANY($1::uuid[]) AND n.entity_type='industry_chain'
  AND n.status='active' AND d.review_status='approved'`,
			ids: industryChainIDs(pkg.IndustryChains), want: len(pkg.IndustryChains),
		},
		{
			label: "Chain Node additions",
			query: `SELECT count(*) FROM entity_nodes n
JOIN chain_node_profiles p ON p.entity_id=n.id
WHERE n.id=ANY($1::uuid[]) AND n.entity_type='chain_node'
  AND n.status='active' AND p.review_status='approved'`,
			ids: chainNodeAdditionIDs(pkg.ChainNodeAdditions), want: len(pkg.ChainNodeAdditions),
		},
		{
			label: "Industry mappings",
			query: `SELECT count(*) FROM entity_edges WHERE id=ANY($1::uuid[])
  AND relation_type='mapped_to_industry' AND status='active'`,
			ids: mappingIDs(pkg.IndustryMappings), want: len(pkg.IndustryMappings),
		},
		{
			label: "Concept mappings",
			query: `SELECT count(*) FROM entity_edges WHERE id=ANY($1::uuid[])
  AND relation_type='mapped_to_concept' AND status='active'`,
			ids: mappingIDs(pkg.ConceptMappings), want: len(pkg.ConceptMappings),
		},
		{
			label: "memberships",
			query: `SELECT count(*) FROM industry_chain_node_memberships
WHERE industry_chain_entity_id=ANY($1::uuid[]) AND review_status='approved' AND status='active'`,
			ids: industryChainIDs(pkg.IndustryChains), want: len(pkg.Memberships),
		},
		{
			label: "chain graph edges",
			query: `SELECT count(*) FROM industry_chain_graph_edges
WHERE industry_chain_entity_id=ANY($1::uuid[]) AND review_status='approved' AND status='active'`,
			ids: industryChainIDs(pkg.IndustryChains), want: len(pkg.GraphEdges),
		},
		{
			label: "global Chain Node relations",
			query: `SELECT count(*) FROM chain_node_relations WHERE id=ANY($1::uuid[]) AND status='active'`,
			ids:   globalRelationIDs(pkg.GlobalRelations), want: len(pkg.GlobalRelations),
		},
	}
	for _, check := range checks {
		if len(check.ids) == 0 {
			if check.want != 0 {
				return fmt.Errorf("%s has no verification IDs", check.label)
			}
			continue
		}
		var got int
		if err := t.tx.QueryRowContext(ctx, check.query, check.ids).Scan(&got); err != nil {
			return fmt.Errorf("verify %s: %w", check.label, err)
		}
		if got != check.want {
			return fmt.Errorf("verify %s: got %d rows, want %d", check.label, got, check.want)
		}
	}
	return t.verifyPersistedContent(ctx, pkg)
}

func (t *postgresIndustryRelationshipImportTx) InsertIndustryRelationshipImportReceipt(
	ctx context.Context,
	receipt biz.Receipt,
) error {
	counts, err := json.Marshal(receipt.PackageCounts)
	if err != nil {
		return fmt.Errorf("encode Industry relationship receipt counts: %w", err)
	}
	_, err = t.tx.ExecContext(ctx, `INSERT INTO industry_relationship_import_receipts (
    id, package_sha256, manifest_sha256, relation_spec_sha256, approval_basis,
    package_counts, caller_subject, imported_at
) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8)`,
		receipt.ID, receipt.PackageSHA256, receipt.ManifestSHA256,
		receipt.RelationSpecSHA256, receipt.ApprovalBasis, counts,
		receipt.CallerSubject, receipt.ImportedAt,
	)
	return err
}

func (t *postgresIndustryRelationshipImportTx) verifyPersistedContent(
	ctx context.Context,
	pkg biz.Package,
) error {
	for _, item := range pkg.IndustryChains {
		var got biz.IndustryChain
		var route sql.NullString
		var aliasesJSON, observableVariablesJSON []byte
		var asOf time.Time
		err := t.tx.QueryRowContext(ctx, `SELECT n.id::text,n.entity_key,n.entity_type,n.layer_code,
       n.name,n.canonical_name,array_to_json(n.aliases),n.status,d.scope,d.target_output,d.end_use,
       d.technology_route_qualifier,array_to_json(d.observable_variables),d.geography,d.as_of_date,
       d.review_status,d.review_note
FROM entity_nodes n JOIN industry_chain_definitions d ON d.entity_id=n.id
WHERE n.id=$1::uuid`, item.EntityID).Scan(
			&got.EntityID, &got.EntityKey, &got.EntityType, &got.LayerCode,
			&got.Name, &got.CanonicalName, &aliasesJSON, &got.Status, &got.Scope,
			&got.TargetOutput, &got.EndUse, &route, &observableVariablesJSON, &got.Geography, &asOf,
			&got.ReviewStatus, &got.ReviewNote,
		)
		if err != nil {
			return err
		}
		got.Aliases, err = decodeStringArrayJSON(aliasesJSON, "Industry Chain aliases")
		if err != nil {
			return err
		}
		got.ObservableVariables, err = decodeStringArrayJSON(
			observableVariablesJSON,
			"Industry Chain observable_variables",
		)
		if err != nil {
			return err
		}
		got.AsOfDate = asOf.Format("2006-01-02")
		got.RelationshipApprovalBasis = item.RelationshipApprovalBasis
		if route.Valid {
			got.TechnologyRouteQualifier = &route.String
		}
		if !reflect.DeepEqual(got, item) {
			return fmt.Errorf("persisted Industry Chain %s content drift", item.EntityKey)
		}
	}
	for _, item := range pkg.ChainNodeAdditions {
		var entityID, entityKey, entityType, layerCode, name, canonicalName, status string
		var definition, boundary, reviewStatus string
		var aliasesJSON []byte
		err := t.tx.QueryRowContext(ctx, `SELECT n.id::text,n.entity_key,n.entity_type,n.layer_code,
       n.name,n.canonical_name,array_to_json(n.aliases),n.status,p.definition,p.boundary_note,p.review_status
FROM entity_nodes n JOIN chain_node_profiles p ON p.entity_id=n.id
WHERE n.id=$1::uuid`, item.EntityID).Scan(
			&entityID, &entityKey, &entityType, &layerCode, &name, &canonicalName,
			&aliasesJSON, &status, &definition, &boundary, &reviewStatus,
		)
		if err != nil {
			return err
		}
		aliases, err := decodeStringArrayJSON(aliasesJSON, "Chain Node aliases")
		if err != nil {
			return err
		}
		if entityID != item.EntityID || entityKey != item.EntityKey ||
			entityType != item.EntityType || layerCode != item.LayerCode ||
			name != item.Name || canonicalName != item.CanonicalName ||
			!reflect.DeepEqual(aliases, item.Aliases) || status != item.Status ||
			definition != item.Definition || boundary != item.BoundaryNote ||
			reviewStatus != item.ReviewStatus {
			return fmt.Errorf("persisted Chain Node %s content drift", item.EntityKey)
		}
	}
	for _, items := range [][]biz.EntityMapping{pkg.IndustryMappings, pkg.ConceptMappings} {
		for _, item := range items {
			var id, from, to, relationType, evidenceNote, status string
			err := t.tx.QueryRowContext(ctx, `SELECT id::text,from_entity_id::text,to_entity_id::text,
       relation_type,evidence_note,status FROM entity_edges WHERE id=$1::uuid`, item.RelationID).Scan(
				&id, &from, &to, &relationType, &evidenceNote, &status,
			)
			if err != nil {
				return err
			}
			if id != item.RelationID || from != item.FromEntityID || to != item.ToEntityID ||
				relationType != item.RelationType || evidenceNote != item.EvidenceNote ||
				status != item.Status {
				return fmt.Errorf("persisted mapping %s content drift", item.RelationKey)
			}
		}
	}
	for _, item := range pkg.Memberships {
		var chainID, nodeID, stage, reviewStatus, status, reason, sourceName, sourceURL string
		var evidenceIDsJSON []byte
		var position int
		var verifiedAt time.Time
		err := t.tx.QueryRowContext(ctx, `SELECT industry_chain_entity_id::text,
       chain_node_entity_id::text,position,contextual_stage,review_status,status,
       inclusion_reason,array_to_json(evidence_ids),source_name,source_url,verified_at
FROM industry_chain_node_memberships
WHERE industry_chain_entity_id=$1::uuid AND chain_node_entity_id=$2::uuid`,
			item.IndustryChainEntityID, item.ChainNodeEntityID).Scan(
			&chainID, &nodeID, &position, &stage, &reviewStatus, &status, &reason, &evidenceIDsJSON,
			&sourceName, &sourceURL, &verifiedAt,
		)
		if err != nil {
			return err
		}
		evidenceIDs, err := decodeStringArrayJSON(evidenceIDsJSON, "membership evidence_ids")
		if err != nil {
			return err
		}
		if chainID != item.IndustryChainEntityID || nodeID != item.ChainNodeEntityID ||
			position != item.Position || stage != item.ContextualStage ||
			reviewStatus != item.ReviewStatus || status != item.Status ||
			reason != item.InclusionReason || !reflect.DeepEqual(evidenceIDs, item.EvidenceIDs) ||
			sourceName != item.SourceName ||
			sourceURL != item.SourceURL || !verifiedAt.Equal(item.VerifiedAt) {
			return fmt.Errorf("persisted membership %s content drift", item.RelationKey)
		}
	}
	for _, item := range pkg.GraphEdges {
		var id, chainID, fromID, toID, relationType, mechanism string
		var condition, omitted sql.NullString
		var segment, reviewStatus, status, sourceName, sourceURL string
		var evidenceIDsJSON []byte
		var verifiedAt time.Time
		err := t.tx.QueryRowContext(ctx, `SELECT id::text,industry_chain_entity_id::text,
       from_chain_node_entity_id::text,to_chain_node_entity_id::text,relation_type,
       mechanism,condition_note,segment_kind,omitted_step_note,review_status,status,
       array_to_json(evidence_ids),source_name,source_url,verified_at
FROM industry_chain_graph_edges WHERE id=$1::uuid`, item.ID).Scan(
			&id, &chainID, &fromID, &toID, &relationType, &mechanism, &condition,
			&segment, &omitted, &reviewStatus, &status, &evidenceIDsJSON, &sourceName,
			&sourceURL, &verifiedAt,
		)
		if err != nil {
			return err
		}
		evidenceIDs, err := decodeStringArrayJSON(evidenceIDsJSON, "graph edge evidence_ids")
		if err != nil {
			return err
		}
		if id != item.ID || chainID != item.IndustryChainEntityID ||
			fromID != item.FromChainNodeEntityID || toID != item.ToChainNodeEntityID ||
			relationType != item.RelationType || mechanism != item.Mechanism ||
			nullableEqual(condition, item.ConditionNote) == false ||
			segment != item.SegmentKind || nullableEqual(omitted, item.OmittedStepNote) == false ||
			reviewStatus != item.ReviewStatus || status != item.Status ||
			!reflect.DeepEqual(evidenceIDs, item.EvidenceIDs) ||
			sourceName != item.SourceName || sourceURL != item.SourceURL ||
			!verifiedAt.Equal(item.VerifiedAt) {
			return fmt.Errorf("persisted graph edge %s content drift", item.RelationKey)
		}
	}
	for _, item := range pkg.GlobalRelations {
		var id, fromID, toID, relationType, mechanism, evidenceNote, provenance, status string
		var condition sql.NullString
		var verifiedAt time.Time
		err := t.tx.QueryRowContext(ctx, `SELECT id::text,from_chain_node_entity_id::text,
       to_chain_node_entity_id::text,relation_type,mechanism,condition_note,
       evidence_note,provenance,verified_at,status
FROM chain_node_relations WHERE id=$1::uuid`, item.ID).Scan(
			&id, &fromID, &toID, &relationType, &mechanism, &condition,
			&evidenceNote, &provenance, &verifiedAt, &status,
		)
		if err != nil {
			return err
		}
		if id != item.ID || fromID != item.FromChainNodeEntityID ||
			toID != item.ToChainNodeEntityID || relationType != item.RelationType ||
			mechanism != item.Mechanism || nullableEqual(condition, item.ConditionNote) == false ||
			evidenceNote != item.EvidenceNote || provenance != item.Provenance ||
			!verifiedAt.Equal(item.VerifiedAt) || status != item.Status {
			return fmt.Errorf("persisted global relation %s content drift", item.RelationKey)
		}
	}
	return nil
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func nullablePointerString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableEqual(value sql.NullString, expected *string) bool {
	if expected == nil {
		return !value.Valid
	}
	return value.Valid && value.String == *expected
}

func decodeStringArrayJSON(raw []byte, label string) ([]string, error) {
	var values []string
	if len(raw) == 0 {
		return nil, fmt.Errorf("decode %s: empty JSON payload", label)
	}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if values == nil {
		values = []string{}
	}
	return values, nil
}

func industryChainIDs(items []biz.IndustryChain) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.EntityID)
	}
	return result
}

func chainNodeAdditionIDs(items []biz.ChainNodeAddition) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.EntityID)
	}
	return result
}

func mappingIDs(items []biz.EntityMapping) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.RelationID)
	}
	return result
}

func globalRelationIDs(items []biz.GlobalChainNodeRelation) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.ID)
	}
	return result
}

var _ biz.Store = repository{}
var _ biz.Transaction = (*postgresIndustryRelationshipImportTx)(nil)
