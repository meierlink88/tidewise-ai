package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
)

const industryGraphReceiptApprovalBasis = "user_explicit_delegated_review"

var errIndustryGraphSnapshotDatabase = errors.New(
	"PostgreSQL Industry graph snapshot data is unavailable",
)

const industryGraphReceiptQuery = `SELECT package_sha256, approval_basis
FROM industry_relationship_import_receipts
WHERE package_sha256 = $1`

const industryGraphNodesQuery = `SELECT node.id::text, node.entity_key, node.entity_type,
       node.canonical_name, COALESCE(array_to_json(node.aliases)::text, '[]')
FROM entity_nodes node
WHERE node.status = 'active'
  AND btrim(node.entity_key) <> ''
  AND (
    (
      node.entity_type = 'industry'
      AND EXISTS (
        SELECT 1 FROM industry_profiles profile
        WHERE profile.entity_id = node.id AND profile.review_status = 'approved'
      )
    )
    OR (
      node.entity_type = 'concept'
      AND EXISTS (
        SELECT 1 FROM concept_profiles profile
        WHERE profile.entity_id = node.id AND profile.review_status = 'approved'
      )
    )
    OR (
      node.entity_type = 'industry_chain'
      AND EXISTS (
        SELECT 1 FROM industry_chain_definitions definition
        WHERE definition.entity_id = node.id AND definition.review_status = 'approved'
      )
    )
    OR (
      node.entity_type = 'chain_node'
      AND EXISTS (
        SELECT 1 FROM chain_node_profiles profile
        WHERE profile.entity_id = node.id AND profile.review_status = 'approved'
      )
    )
  )
ORDER BY node.id`

const industryGraphIndustryHierarchyQuery = `SELECT child_node.id::text, parent_node.id::text,
       child_node.entity_key, parent_node.entity_key
FROM industry_profiles child
JOIN industry_profiles parent ON parent.entity_id = child.parent_industry_entity_id
JOIN entity_nodes child_node ON child_node.id = child.entity_id
JOIN entity_nodes parent_node ON parent_node.id = parent.entity_id
WHERE child.review_status = 'approved'
  AND parent.review_status = 'approved'
  AND child_node.status = 'active'
  AND parent_node.status = 'active'
  AND child_node.entity_type = 'industry'
  AND parent_node.entity_type = 'industry'
ORDER BY child_node.id, parent_node.id`

const industryGraphMappingsQuery = `SELECT edge.from_entity_id::text, edge.to_entity_id::text,
       edge.relation_type, chain.entity_key, target.entity_key, edge.evidence_note
FROM entity_edges edge
JOIN entity_nodes chain ON chain.id = edge.from_entity_id
JOIN industry_chain_definitions definition ON definition.entity_id = chain.id
JOIN entity_nodes target ON target.id = edge.to_entity_id
JOIN industry_profiles profile ON profile.entity_id = target.id
WHERE edge.relation_type = 'mapped_to_industry'
  AND edge.status = 'active'
  AND chain.status = 'active'
  AND target.status = 'active'
  AND definition.review_status = 'approved'
  AND profile.review_status = 'approved'
UNION ALL
SELECT edge.from_entity_id::text, edge.to_entity_id::text,
       edge.relation_type, chain.entity_key, target.entity_key, edge.evidence_note
FROM entity_edges edge
JOIN entity_nodes chain ON chain.id = edge.from_entity_id
JOIN industry_chain_definitions definition ON definition.entity_id = chain.id
JOIN entity_nodes target ON target.id = edge.to_entity_id
JOIN concept_profiles profile ON profile.entity_id = target.id
WHERE edge.relation_type = 'mapped_to_concept'
  AND edge.status = 'active'
  AND chain.status = 'active'
  AND target.status = 'active'
  AND definition.review_status = 'approved'
  AND profile.review_status = 'approved'
ORDER BY 1, 3, 2`

const industryGraphMembershipsQuery = `SELECT membership.industry_chain_entity_id::text,
       membership.chain_node_entity_id::text, chain.entity_key, node.entity_key,
       membership.contextual_stage, membership.position, membership.inclusion_reason
FROM industry_chain_node_memberships membership
JOIN entity_nodes chain ON chain.id = membership.industry_chain_entity_id
JOIN industry_chain_definitions definition ON definition.entity_id = chain.id
JOIN entity_nodes node ON node.id = membership.chain_node_entity_id
JOIN chain_node_profiles profile ON profile.entity_id = node.id
WHERE membership.review_status = 'approved'
  AND membership.status = 'active'
  AND chain.status = 'active'
  AND node.status = 'active'
  AND definition.review_status = 'approved'
  AND profile.review_status = 'approved'
ORDER BY membership.industry_chain_entity_id, membership.position,
         membership.chain_node_entity_id`

const industryGraphChainEdgesQuery = `SELECT edge.industry_chain_entity_id::text,
       edge.from_chain_node_entity_id::text, edge.to_chain_node_entity_id::text,
       chain.entity_key, from_node.entity_key, edge.relation_type, to_node.entity_key,
       edge.mechanism
FROM industry_chain_graph_edges edge
JOIN entity_nodes chain ON chain.id = edge.industry_chain_entity_id
JOIN industry_chain_definitions definition ON definition.entity_id = chain.id
JOIN entity_nodes from_node ON from_node.id = edge.from_chain_node_entity_id
JOIN chain_node_profiles from_profile ON from_profile.entity_id = from_node.id
JOIN entity_nodes to_node ON to_node.id = edge.to_chain_node_entity_id
JOIN chain_node_profiles to_profile ON to_profile.entity_id = to_node.id
JOIN industry_chain_node_memberships from_membership
  ON from_membership.industry_chain_entity_id = edge.industry_chain_entity_id
 AND from_membership.chain_node_entity_id = edge.from_chain_node_entity_id
JOIN industry_chain_node_memberships to_membership
  ON to_membership.industry_chain_entity_id = edge.industry_chain_entity_id
 AND to_membership.chain_node_entity_id = edge.to_chain_node_entity_id
WHERE edge.review_status = 'approved'
  AND edge.status = 'active'
  AND chain.status = 'active'
  AND from_node.status = 'active'
  AND to_node.status = 'active'
  AND definition.review_status = 'approved'
  AND from_profile.review_status = 'approved'
  AND to_profile.review_status = 'approved'
  AND from_membership.review_status = 'approved'
  AND from_membership.status = 'active'
  AND to_membership.review_status = 'approved'
  AND to_membership.status = 'active'
ORDER BY edge.industry_chain_entity_id, edge.from_chain_node_entity_id,
         edge.relation_type, edge.to_chain_node_entity_id`

const industryGraphGlobalNodeHierarchyQuery = `SELECT relation.id::text,
       relation.from_chain_node_entity_id::text, relation.to_chain_node_entity_id::text,
       relation.mechanism
FROM chain_node_relations relation
JOIN entity_nodes from_node ON from_node.id = relation.from_chain_node_entity_id
JOIN chain_node_profiles from_profile ON from_profile.entity_id = from_node.id
JOIN entity_nodes to_node ON to_node.id = relation.to_chain_node_entity_id
JOIN chain_node_profiles to_profile ON to_profile.entity_id = to_node.id
WHERE relation.relation_type = 'is_subcategory_of'
  AND relation.status = 'active'
  AND from_node.status = 'active'
  AND to_node.status = 'active'
  AND from_profile.review_status = 'approved'
  AND to_profile.review_status = 'approved'
ORDER BY relation.id`

type IndustryGraphSnapshotReader struct {
	db *sql.DB
}

func NewIndustryGraphSnapshotReader(db *sql.DB) *IndustryGraphSnapshotReader {
	return &IndustryGraphSnapshotReader{db: db}
}

func (r *IndustryGraphSnapshotReader) ReadIndustryGraphSnapshot(
	ctx context.Context,
	expectedPackageSHA string,
) (biz.Projection, error) {
	if r == nil || r.db == nil {
		return biz.Projection{}, errors.New("PostgreSQL Industry graph database is required")
	}
	if !biz.ValidPackageSHA256(expectedPackageSHA) {
		return biz.Projection{}, errors.New("Industry graph package SHA-256 must contain 64 lowercase hexadecimal characters")
	}

	tx, err := beginIndustryGraphSnapshotTransaction(ctx, r.db)
	if err != nil {
		return biz.Projection{}, industryGraphSnapshotDatabaseError(
			"begin PostgreSQL Industry graph snapshot",
		)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := verifyIndustryGraphReceipt(ctx, tx, expectedPackageSHA); err != nil {
		return biz.Projection{}, err
	}
	nodes, err := readIndustryGraphNodes(ctx, tx)
	if err != nil {
		return biz.Projection{}, err
	}
	relationships, err := readIndustryGraphRelationships(ctx, tx)
	if err != nil {
		return biz.Projection{}, err
	}
	nodes = industryGraphEndpointClosure(nodes, relationships)

	projection := biz.Projection{
		PackageSHA256: expectedPackageSHA,
		Nodes:         nodes,
		Relationships: relationships,
	}
	if err := biz.ValidateProjection(projection); err != nil {
		return biz.Projection{}, fmt.Errorf("validate PostgreSQL Industry graph snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return biz.Projection{}, industryGraphSnapshotDatabaseError(
			"commit PostgreSQL Industry graph read-only snapshot",
		)
	}
	return projection, nil
}

func beginIndustryGraphSnapshotTransaction(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	return db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
}

func verifyIndustryGraphReceipt(ctx context.Context, tx *sql.Tx, expectedPackageSHA string) error {
	var packageSHA, approvalBasis string
	err := tx.QueryRowContext(ctx, industryGraphReceiptQuery, expectedPackageSHA).Scan(
		&packageSHA,
		&approvalBasis,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("approved Industry relationship import receipt %s was not found", expectedPackageSHA)
	}
	if err != nil {
		return industryGraphSnapshotDatabaseError("read Industry relationship import receipt")
	}
	if packageSHA != expectedPackageSHA {
		return errors.New("Industry relationship import receipt package SHA-256 differs from the requested package")
	}
	if approvalBasis != industryGraphReceiptApprovalBasis {
		return errors.New("Industry relationship import receipt is not approved for graph projection")
	}
	return nil
}

func readIndustryGraphNodes(ctx context.Context, tx *sql.Tx) ([]biz.Node, error) {
	rows, err := tx.QueryContext(ctx, industryGraphNodesQuery)
	if err != nil {
		return nil, industryGraphSnapshotDatabaseError("query approved Industry graph nodes")
	}
	defer rows.Close()

	nodes := make([]biz.Node, 0)
	for rows.Next() {
		var node biz.Node
		var entityType, aliasesJSON string
		if err := rows.Scan(
			&node.EntityID,
			&node.EntityKey,
			&entityType,
			&node.CanonicalName,
			&aliasesJSON,
		); err != nil {
			return nil, industryGraphSnapshotDatabaseError("scan approved Industry graph node")
		}
		node.EntityType, err = mapIndustryGraphEntityType(entityType)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(aliasesJSON), &node.Aliases); err != nil {
			return nil, fmt.Errorf("decode Industry graph node %q aliases: %w", node.EntityID, err)
		}
		if node.Aliases == nil {
			node.Aliases = []string{}
		}
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, industryGraphSnapshotDatabaseError("iterate approved Industry graph nodes")
	}
	return nodes, nil
}

func readIndustryGraphRelationships(ctx context.Context, tx *sql.Tx) ([]biz.Relationship, error) {
	relationships := make([]biz.Relationship, 0)
	readers := []func(context.Context, *sql.Tx) ([]biz.Relationship, error){
		readIndustryHierarchyRelationships,
		readIndustryMappingRelationships,
		readIndustryChainMemberships,
		readIndustryChainGraphEdges,
		readGlobalChainNodeHierarchy,
	}
	for _, read := range readers {
		items, err := read(ctx, tx)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, items...)
	}
	sort.Slice(relationships, func(left, right int) bool {
		return relationships[left].RelationKey < relationships[right].RelationKey
	})
	return relationships, nil
}

func readIndustryHierarchyRelationships(ctx context.Context, tx *sql.Tx) ([]biz.Relationship, error) {
	rows, err := tx.QueryContext(ctx, industryGraphIndustryHierarchyQuery)
	if err != nil {
		return nil, industryGraphSnapshotDatabaseError("query approved Industry hierarchy")
	}
	defer rows.Close()

	relationships := make([]biz.Relationship, 0)
	for rows.Next() {
		var relationship biz.Relationship
		var fromKey, toKey string
		if err := rows.Scan(
			&relationship.FromEntityID,
			&relationship.ToEntityID,
			&fromKey,
			&toKey,
		); err != nil {
			return nil, industryGraphSnapshotDatabaseError("scan approved Industry hierarchy")
		}
		relationship.Type = biz.RelationshipTypeIsSubcategoryOf
		relationship.RelationKey = fromKey + "|is_subcategory_of|" + toKey
		relationship.Mechanism = "Authoritative Industry classification hierarchy"
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, industryGraphSnapshotDatabaseError("iterate approved Industry hierarchy")
	}
	return relationships, nil
}

func readIndustryMappingRelationships(ctx context.Context, tx *sql.Tx) ([]biz.Relationship, error) {
	rows, err := tx.QueryContext(ctx, industryGraphMappingsQuery)
	if err != nil {
		return nil, industryGraphSnapshotDatabaseError("query approved Industry graph mappings")
	}
	defer rows.Close()

	relationships := make([]biz.Relationship, 0)
	for rows.Next() {
		var relationship biz.Relationship
		var relationType, fromKey, toKey, evidenceNote string
		if err := rows.Scan(
			&relationship.FromEntityID,
			&relationship.ToEntityID,
			&relationType,
			&fromKey,
			&toKey,
			&evidenceNote,
		); err != nil {
			return nil, industryGraphSnapshotDatabaseError("scan approved Industry graph mapping")
		}
		var err error
		relationship.Type, err = mapIndustryGraphRelationshipType(relationType)
		if err != nil {
			return nil, err
		}
		relationship.ChainID = relationship.FromEntityID
		relationship.RelationKey = fromKey + "|" + relationType + "|" + toKey
		relationship.Mechanism = industryMappingMechanism(evidenceNote)
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, industryGraphSnapshotDatabaseError("iterate approved Industry graph mappings")
	}
	return relationships, nil
}

func readIndustryChainMemberships(ctx context.Context, tx *sql.Tx) ([]biz.Relationship, error) {
	rows, err := tx.QueryContext(ctx, industryGraphMembershipsQuery)
	if err != nil {
		return nil, industryGraphSnapshotDatabaseError("query approved Industry Chain memberships")
	}
	defer rows.Close()

	relationships := make([]biz.Relationship, 0)
	for rows.Next() {
		var relationship biz.Relationship
		var chainKey, nodeKey string
		var position int
		if err := rows.Scan(
			&relationship.FromEntityID,
			&relationship.ToEntityID,
			&chainKey,
			&nodeKey,
			&relationship.ContextualStage,
			&position,
			&relationship.Mechanism,
		); err != nil {
			return nil, industryGraphSnapshotDatabaseError("scan approved Industry Chain membership")
		}
		relationship.Type = biz.RelationshipTypeHasNode
		relationship.ChainID = relationship.FromEntityID
		relationship.RelationKey = chainKey + "|has_node|" + nodeKey
		relationship.Position = intPointer(position)
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, industryGraphSnapshotDatabaseError("iterate approved Industry Chain memberships")
	}
	return relationships, nil
}

func readIndustryChainGraphEdges(ctx context.Context, tx *sql.Tx) ([]biz.Relationship, error) {
	rows, err := tx.QueryContext(ctx, industryGraphChainEdgesQuery)
	if err != nil {
		return nil, industryGraphSnapshotDatabaseError("query approved Industry Chain graph edges")
	}
	defer rows.Close()

	relationships := make([]biz.Relationship, 0)
	for rows.Next() {
		var relationship biz.Relationship
		var chainKey, fromKey, relationType, toKey string
		if err := rows.Scan(
			&relationship.ChainID,
			&relationship.FromEntityID,
			&relationship.ToEntityID,
			&chainKey,
			&fromKey,
			&relationType,
			&toKey,
			&relationship.Mechanism,
		); err != nil {
			return nil, industryGraphSnapshotDatabaseError("scan approved Industry Chain graph edge")
		}
		var err error
		relationship.Type, err = mapIndustryGraphRelationshipType(relationType)
		if err != nil {
			return nil, err
		}
		relationship.RelationKey = chainKey + "|" + fromKey + "|" + relationType + "|" + toKey
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, industryGraphSnapshotDatabaseError("iterate approved Industry Chain graph edges")
	}
	return relationships, nil
}

func readGlobalChainNodeHierarchy(ctx context.Context, tx *sql.Tx) ([]biz.Relationship, error) {
	rows, err := tx.QueryContext(ctx, industryGraphGlobalNodeHierarchyQuery)
	if err != nil {
		return nil, industryGraphSnapshotDatabaseError("query approved global Chain Node hierarchy")
	}
	defer rows.Close()

	relationships := make([]biz.Relationship, 0)
	for rows.Next() {
		var relationship biz.Relationship
		var relationID string
		if err := rows.Scan(
			&relationID,
			&relationship.FromEntityID,
			&relationship.ToEntityID,
			&relationship.Mechanism,
		); err != nil {
			return nil, industryGraphSnapshotDatabaseError("scan approved global Chain Node hierarchy")
		}
		relationship.Type = biz.RelationshipTypeIsSubcategoryOf
		relationship.RelationKey = "chain_node_relation:" + relationID
		relationships = append(relationships, relationship)
	}
	if err := rows.Err(); err != nil {
		return nil, industryGraphSnapshotDatabaseError("iterate approved global Chain Node hierarchy")
	}
	return relationships, nil
}

func industryGraphEndpointClosure(
	nodes []biz.Node,
	relationships []biz.Relationship,
) []biz.Node {
	endpoints := make(map[string]struct{}, len(relationships)*2)
	for _, relationship := range relationships {
		endpoints[relationship.FromEntityID] = struct{}{}
		endpoints[relationship.ToEntityID] = struct{}{}
	}
	filtered := make([]biz.Node, 0, len(nodes))
	for _, node := range nodes {
		if _, ok := endpoints[node.EntityID]; ok {
			filtered = append(filtered, node)
		}
	}
	sort.Slice(filtered, func(left, right int) bool {
		return filtered[left].EntityID < filtered[right].EntityID
	})
	return filtered
}

func mapIndustryGraphEntityType(value string) (biz.EntityType, error) {
	switch value {
	case string(biz.EntityTypeIndustry):
		return biz.EntityTypeIndustry, nil
	case string(biz.EntityTypeConcept):
		return biz.EntityTypeConcept, nil
	case string(biz.EntityTypeIndustryChain):
		return biz.EntityTypeIndustryChain, nil
	case string(biz.EntityTypeChainNode):
		return biz.EntityTypeChainNode, nil
	default:
		return "", fmt.Errorf("unsupported PostgreSQL Industry graph entity type %q", value)
	}
}

func mapIndustryGraphRelationshipType(value string) (biz.RelationshipType, error) {
	switch value {
	case "mapped_to_industry":
		return biz.RelationshipTypeMappedToIndustry, nil
	case "mapped_to_concept":
		return biz.RelationshipTypeMappedToConcept, nil
	case "input_to":
		return biz.RelationshipTypeInputTo, nil
	case "is_component_of":
		return biz.RelationshipTypeIsComponentOf, nil
	case "depends_on":
		return biz.RelationshipTypeDependsOn, nil
	default:
		return "", fmt.Errorf("unsupported PostgreSQL Industry graph relationship type %q", value)
	}
}

func industryMappingMechanism(evidenceNote string) string {
	const evidenceSuffix = " evidence="
	if index := strings.LastIndex(evidenceNote, evidenceSuffix); index > 0 {
		return evidenceNote[:index]
	}
	return evidenceNote
}

func industryGraphSnapshotDatabaseError(operation string) error {
	return fmt.Errorf("%s: %w", operation, errIndustryGraphSnapshotDatabase)
}

func intPointer(value int) *int {
	return &value
}
