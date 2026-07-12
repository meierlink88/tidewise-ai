package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

const listPhysicalConstraintsQuery = `
SELECT id, industry_chain_entity_id, chain_node_entity_id, topology_edge_id,
       constraint_type, mechanism, physical_limit_note, mitigation_path,
	   source_name, source_url, verified_at, review_status, status, generated_by_ai
FROM industry_chain_physical_constraints
WHERE review_status = 'approved' AND status = 'active'
  AND (cardinality($1::uuid[]) = 0 OR industry_chain_entity_id = ANY($1::uuid[]))
  AND (cardinality($2::uuid[]) = 0 AND cardinality($3::uuid[]) = 0
       OR chain_node_entity_id = ANY($2::uuid[])
       OR topology_edge_id = ANY($3::uuid[]))
ORDER BY industry_chain_entity_id, id`

func (r *InMemoryRepository) SeedIndustryChainPhysicalConstraints(values []domain.IndustryChainPhysicalConstraint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, value := range values {
		r.physicalConstraints[value.ID] = value
	}
}

func (r *InMemoryRepository) ListPhysicalConstraints(_ context.Context, filter PhysicalConstraintFilter) ([]domain.IndustryChainPhysicalConstraint, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]domain.IndustryChainPhysicalConstraint, 0)
	for _, value := range r.physicalConstraints {
		if value.Status != domain.StatusActive || value.ReviewStatus != domain.ReviewStatusApproved {
			continue
		}
		if !matchesID(filter.ChainIDs, value.IndustryChainEntityID) || !matchesSubject(filter, value) {
			continue
		}
		items = append(items, value)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (r PostgresRepository) ListPhysicalConstraints(ctx context.Context, filter PhysicalConstraintFilter) ([]domain.IndustryChainPhysicalConstraint, error) {
	rows, err := r.db.QueryContext(ctx, listPhysicalConstraintsQuery, filter.ChainIDs, filter.NodeIDs, filter.TopologyEdgeIDs)
	if err != nil {
		return nil, fmt.Errorf("query physical constraints: %w", err)
	}
	defer rows.Close()
	items := make([]domain.IndustryChainPhysicalConstraint, 0)
	for rows.Next() {
		var value domain.IndustryChainPhysicalConstraint
		var nodeID, edgeID sql.NullString
		if err := rows.Scan(&value.ID, &value.IndustryChainEntityID, &nodeID, &edgeID, &value.ConstraintType, &value.Mechanism, &value.PhysicalLimitNote, &value.MitigationPath, &value.SourceName, &value.SourceURL, &value.VerifiedAt, &value.ReviewStatus, &value.Status, &value.GeneratedByAI); err != nil {
			return nil, err
		}
		if nodeID.Valid {
			value.ChainNodeEntityID = nodeID.String
		}
		if edgeID.Valid {
			value.TopologyEdgeID = edgeID.String
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func matchesID(values []string, target string) bool {
	return len(values) == 0 || containsID(values, target)
}
func matchesSubject(filter PhysicalConstraintFilter, value domain.IndustryChainPhysicalConstraint) bool {
	if len(filter.NodeIDs) == 0 && len(filter.TopologyEdgeIDs) == 0 {
		return true
	}
	return (value.ChainNodeEntityID != "" && containsID(filter.NodeIDs, value.ChainNodeEntityID)) || (value.TopologyEdgeID != "" && containsID(filter.TopologyEdgeIDs, value.TopologyEdgeID))
}
func containsID(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
