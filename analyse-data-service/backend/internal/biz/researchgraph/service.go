package researchgraph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

const (
	ContractVersion       = "research-graph-search.v1"
	StableOrderingVersion = "depth-seed-relation-endpoints-edge-id.v1"
	MaxDepth              = 5
	MaxSeedEntities       = 20
	MaxRelationFilters    = 20
	MaxNodeBudget         = 500
	MaxEdgeBudget         = 1_000
	MaxResultBytes        = 4 * 1024 * 1024
)

type Direction string

const (
	DirectionOutgoing Direction = "outgoing"
	DirectionIncoming Direction = "incoming"
	DirectionBoth     Direction = "both"
)

type RelationFilter struct {
	RelationType string    `json:"relation_type"`
	Direction    Direction `json:"direction"`
}

type Request struct {
	AnalysisAsOf          string           `json:"analysis_as_of"`
	SeedEntityIDs         []string         `json:"seed_entity_ids"`
	RelationFilters       []RelationFilter `json:"relation_filters"`
	MaxDepth              int              `json:"max_depth"`
	IndustryChainEntityID *string          `json:"industry_chain_entity_id,omitempty"`
	NodeBudget            int              `json:"node_budget"`
	EdgeBudget            int              `json:"edge_budget"`
}

type Result struct {
	ContractVersion          string                    `json:"contract_version"`
	AnalysisAsOf             string                    `json:"analysis_as_of"`
	QueryFingerprint         string                    `json:"query_fingerprint"`
	GraphFingerprint         string                    `json:"graph_fingerprint"`
	ActualDepth              int                       `json:"actual_depth"`
	Entities                 []Entity                  `json:"entities"`
	RelationDefinitions      []RelationDefinition      `json:"relation_definitions"`
	EntityRelations          []EntityRelation          `json:"entity_relations"`
	IndustryChains           []IndustryChain           `json:"industry_chains"`
	IndustryChainMemberships []IndustryChainMembership `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []IndustryChainGraphEdge  `json:"industry_chain_graph_edges"`
}

type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return e.Reason
}

type ResourceLimitError struct {
	Reason        string
	Component     string
	ActualRows    *int64
	MaxRows       *int64
	ActualBytes   *int64
	MaxBytes      *int64
	RetryGuidance string
}

func ApprovedActiveFactPolicy() FactEligibilityPolicy {
	return FactEligibilityPolicy{
		EntityStatus:              "active",
		EntityRelationStatus:      "active",
		IndustryChainReviewStatus: "approved",
		MembershipReviewStatus:    "approved",
		MembershipStatus:          "active",
		GraphEdgeReviewStatus:     "approved",
		GraphEdgeStatus:           "active",
	}
}

func (e *ResourceLimitError) Error() string {
	return e.Reason
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Search(
	ctx context.Context,
	request Request,
) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("research graph store is required")
	}
	query, normalized, err := validateRequest(request)
	if err != nil {
		return Result{}, err
	}
	graph, err := s.store.Search(ctx, query)
	if err != nil {
		return Result{}, err
	}
	graph = normalizeSubgraph(graph)
	if graph.ActualDepth < 0 || graph.ActualDepth > query.MaxDepth ||
		!referencesResolve(graph) {
		return Result{}, errors.New("research graph result is not reference complete")
	}
	edgeCount := len(graph.EntityRelations) + len(graph.IndustryChainGraphEdges)
	if len(graph.Entities) > query.NodeBudget || edgeCount > query.EdgeBudget {
		component := "research_graph_nodes"
		actual := int64(len(graph.Entities))
		maximum := int64(query.NodeBudget)
		reason := "research graph result exceeds the requested node budget"
		if len(graph.Entities) <= query.NodeBudget {
			component = "research_graph_edges"
			actual = int64(edgeCount)
			maximum = int64(query.EdgeBudget)
			reason = "research graph result exceeds the requested edge budget"
		}
		return Result{}, &ResourceLimitError{
			Reason:     reason,
			Component:  component,
			ActualRows: &actual, MaxRows: &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	graphPayload, err := json.Marshal(graph)
	if err != nil {
		return Result{}, errors.New("research graph result is invalid")
	}
	if len(graphPayload) > MaxResultBytes {
		actual := int64(len(graphPayload))
		maximum := int64(MaxResultBytes)
		return Result{}, &ResourceLimitError{
			Reason:      "research graph result exceeds the response budget",
			Component:   "research_graph_result",
			ActualBytes: &actual, MaxBytes: &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	queryFingerprint, err := fingerprint(normalized)
	if err != nil {
		return Result{}, err
	}
	graphFingerprint, err := fingerprint(graph)
	if err != nil {
		return Result{}, err
	}
	return Result{
		ContractVersion:          ContractVersion,
		AnalysisAsOf:             normalized.AnalysisAsOf,
		QueryFingerprint:         queryFingerprint,
		GraphFingerprint:         graphFingerprint,
		ActualDepth:              graph.ActualDepth,
		Entities:                 graph.Entities,
		RelationDefinitions:      graph.RelationDefinitions,
		EntityRelations:          graph.EntityRelations,
		IndustryChains:           graph.IndustryChains,
		IndustryChainMemberships: graph.IndustryChainMemberships,
		IndustryChainGraphEdges:  graph.IndustryChainGraphEdges,
	}, nil
}

type normalizedRequest struct {
	ContractVersion       string           `json:"contract_version"`
	StableOrderingVersion string           `json:"stable_ordering_version"`
	AnalysisAsOf          string           `json:"analysis_as_of"`
	SeedEntityIDs         []string         `json:"seed_entity_ids"`
	RelationFilters       []RelationFilter `json:"relation_filters"`
	MaxDepth              int              `json:"max_depth"`
	IndustryChainEntityID *string          `json:"industry_chain_entity_id,omitempty"`
	NodeBudget            int              `json:"node_budget"`
	EdgeBudget            int              `json:"edge_budget"`
}

func validateRequest(request Request) (Query, normalizedRequest, error) {
	asOf, err := time.Parse(time.RFC3339, request.AnalysisAsOf)
	if err != nil {
		return Query{}, normalizedRequest{}, &ValidationError{
			Reason: "analysis_as_of must be an RFC3339 UTC timestamp",
		}
	}
	_, offset := asOf.Zone()
	if offset != 0 {
		return Query{}, normalizedRequest{}, &ValidationError{Reason: "analysis_as_of must use UTC"}
	}
	if len(request.SeedEntityIDs) < 1 || len(request.SeedEntityIDs) > MaxSeedEntities {
		return Query{}, normalizedRequest{}, &ValidationError{
			Reason: fmt.Sprintf("seed_entity_ids must contain between 1 and %d IDs", MaxSeedEntities),
		}
	}
	seedSet := map[string]struct{}{}
	for _, id := range request.SeedEntityIDs {
		if !identity.IsUUID(id) {
			return Query{}, normalizedRequest{}, &ValidationError{Reason: "seed_entity_ids contains an invalid UUID"}
		}
		if _, exists := seedSet[id]; exists {
			return Query{}, normalizedRequest{}, &ValidationError{
				Reason: "seed_entity_ids must contain unique IDs",
			}
		}
		seedSet[id] = struct{}{}
	}
	seeds := sortedStrings(seedSet)
	if len(request.RelationFilters) < 1 || len(request.RelationFilters) > MaxRelationFilters {
		return Query{}, normalizedRequest{}, &ValidationError{
			Reason: fmt.Sprintf("relation_filters must contain between 1 and %d filters", MaxRelationFilters),
		}
	}
	filterSet := map[string]RelationFilter{}
	for _, filter := range request.RelationFilters {
		filter.RelationType = strings.TrimSpace(filter.RelationType)
		if filter.RelationType == "" ||
			(filter.Direction != DirectionOutgoing &&
				filter.Direction != DirectionIncoming &&
				filter.Direction != DirectionBoth) {
			return Query{}, normalizedRequest{}, &ValidationError{Reason: "relation_filters is invalid"}
		}
		if _, exists := filterSet[filter.RelationType]; exists {
			return Query{}, normalizedRequest{}, &ValidationError{
				Reason: "relation_filters must configure each relation_type exactly once",
			}
		}
		filterSet[filter.RelationType] = filter
	}
	filterKeys := make([]string, 0, len(filterSet))
	for key := range filterSet {
		filterKeys = append(filterKeys, key)
	}
	sort.Strings(filterKeys)
	filters := make([]RelationFilter, 0, len(filterKeys))
	for _, key := range filterKeys {
		filters = append(filters, filterSet[key])
	}
	if request.MaxDepth < 1 || request.MaxDepth > MaxDepth {
		return Query{}, normalizedRequest{}, &ValidationError{
			Reason: fmt.Sprintf("max_depth must be between 1 and %d", MaxDepth),
		}
	}
	if request.NodeBudget < 1 || request.NodeBudget > MaxNodeBudget ||
		request.EdgeBudget < 1 || request.EdgeBudget > MaxEdgeBudget {
		return Query{}, normalizedRequest{}, &ValidationError{
			Reason: "node_budget or edge_budget exceeds the supported range",
		}
	}
	if request.IndustryChainEntityID != nil &&
		!identity.IsUUID(*request.IndustryChainEntityID) {
		return Query{}, normalizedRequest{}, &ValidationError{
			Reason: "industry_chain_entity_id must be a UUID",
		}
	}
	asOf = asOf.UTC()
	normalized := normalizedRequest{
		ContractVersion:       ContractVersion,
		StableOrderingVersion: StableOrderingVersion,
		AnalysisAsOf:          asOf.Format(time.RFC3339Nano),
		SeedEntityIDs:         seeds,
		RelationFilters:       filters,
		MaxDepth:              request.MaxDepth,
		IndustryChainEntityID: request.IndustryChainEntityID,
		NodeBudget:            request.NodeBudget,
		EdgeBudget:            request.EdgeBudget,
	}
	return Query{
		AnalysisAsOf:          asOf,
		SeedEntityIDs:         seeds,
		RelationFilters:       filters,
		MaxDepth:              request.MaxDepth,
		IndustryChainEntityID: request.IndustryChainEntityID,
		NodeBudget:            request.NodeBudget,
		EdgeBudget:            request.EdgeBudget,
		FactPolicy:            ApprovedActiveFactPolicy(),
	}, normalized, nil
}

func referencesResolve(graph Subgraph) bool {
	entities := map[string]struct{}{}
	for _, entity := range graph.Entities {
		entities[entity.EntityID] = struct{}{}
	}
	relationTypes := map[string]struct{}{}
	for _, definition := range graph.RelationDefinitions {
		relationTypes[definition.RelationType] = struct{}{}
	}
	chains := map[string]struct{}{}
	for _, chain := range graph.IndustryChains {
		if _, ok := entities[chain.IndustryChainEntityID]; !ok {
			return false
		}
		chains[chain.IndustryChainEntityID] = struct{}{}
	}
	memberships := map[string]struct{}{}
	for _, membership := range graph.IndustryChainMemberships {
		if _, ok := chains[membership.IndustryChainEntityID]; !ok {
			return false
		}
		if _, ok := entities[membership.ChainNodeEntityID]; !ok {
			return false
		}
		memberships[membership.IndustryChainEntityID+"\x00"+membership.ChainNodeEntityID] = struct{}{}
	}
	for _, relation := range graph.EntityRelations {
		if _, ok := entities[relation.FromEntityID]; !ok {
			return false
		}
		if _, ok := entities[relation.ToEntityID]; !ok {
			return false
		}
		if _, ok := relationTypes[relation.RelationType]; !ok {
			return false
		}
	}
	for _, edge := range graph.IndustryChainGraphEdges {
		if _, ok := relationTypes[edge.RelationType]; !ok {
			return false
		}
		if _, ok := memberships[edge.IndustryChainEntityID+"\x00"+edge.FromChainNodeEntityID]; !ok {
			return false
		}
		if _, ok := memberships[edge.IndustryChainEntityID+"\x00"+edge.ToChainNodeEntityID]; !ok {
			return false
		}
	}
	return true
}

func normalizeSubgraph(graph Subgraph) Subgraph {
	if graph.Entities == nil {
		graph.Entities = []Entity{}
	}
	if graph.RelationDefinitions == nil {
		graph.RelationDefinitions = []RelationDefinition{}
	}
	if graph.EntityRelations == nil {
		graph.EntityRelations = []EntityRelation{}
	}
	if graph.IndustryChains == nil {
		graph.IndustryChains = []IndustryChain{}
	}
	if graph.IndustryChainMemberships == nil {
		graph.IndustryChainMemberships = []IndustryChainMembership{}
	}
	if graph.IndustryChainGraphEdges == nil {
		graph.IndustryChainGraphEdges = []IndustryChainGraphEdge{}
	}
	return graph
}

func sortedStrings(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func fingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
