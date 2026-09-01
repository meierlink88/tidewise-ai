package research

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
)

type ResearchResourceLimitError struct {
	Reason, Component     string
	ActualRows, MaxRows   *int64
	ActualBytes, MaxBytes *int64
	RetryGuidance         string
}

func (e *ResearchResourceLimitError) Error() string {
	return e.Reason
}

type GraphSubgraph = entitybiz.ResearchGraphSubgraph
type GraphEntity = entitybiz.ResearchGraphEntity
type GraphRelationDefinition = entitybiz.ResearchGraphRelation
type GraphEntityRelation = entitybiz.ResearchGraphEntityRelation
type GraphIndustryChain = entitybiz.ResearchGraphIndustryChain
type GraphIndustryChainMembership = entitybiz.ResearchGraphMembership
type GraphIndustryChainEdge = entitybiz.ResearchGraphIndustryEdge

type GraphStore interface {
	SearchResearchGraph(context.Context, GraphQuery) (GraphSubgraph, error)
}

type GraphQuery = entitybiz.ResearchGraphQuery

const (
	GraphContractVersion       = "research-graph-search.v2"
	GraphStableOrderingVersion = "depth-seed-relation-endpoints-edge-id.v1"
	GraphMaxDepth              = 5
	GraphMaxSeedEntities       = 20
	GraphMaxRelationFilters    = 20
	GraphMaxNodeBudget         = 500
	GraphMaxEdgeBudget         = 1_000
	GraphMaxResultBytes        = 4 * 1024 * 1024
)

type Direction = entitybiz.ResearchGraphDirection

const (
	DirectionOutgoing = entitybiz.ResearchGraphDirectionOutgoing
	DirectionIncoming = entitybiz.ResearchGraphDirectionIncoming
	DirectionBoth     = entitybiz.ResearchGraphDirectionBoth
)

type RelationFilter = entitybiz.ResearchGraphRelationFilter

type GraphSearchRequest struct {
	AnalysisAsOf    string           `json:"analysis_as_of"`
	SeedEntityIDs   []string         `json:"seed_entity_ids"`
	RelationFilters []RelationFilter `json:"relation_filters"`
	MaxDepth        int              `json:"max_depth"`
	IndustryChainID *string          `json:"industry_chain_id,omitempty"`
	NodeBudget      int              `json:"node_budget"`
	EdgeBudget      int              `json:"edge_budget"`
}

type GraphSearchResult struct {
	ContractVersion          string                                  `json:"contract_version"`
	AnalysisAsOf             string                                  `json:"analysis_as_of"`
	QueryFingerprint         string                                  `json:"query_fingerprint"`
	GraphFingerprint         string                                  `json:"graph_fingerprint"`
	ActualDepth              int                                     `json:"actual_depth"`
	Entities                 []entitybiz.ResearchGraphEntity         `json:"entities"`
	RelationDefinitions      []entitybiz.ResearchGraphRelation       `json:"relation_definitions"`
	EntityRelations          []entitybiz.ResearchGraphEntityRelation `json:"entity_relations"`
	IndustryChains           []entitybiz.ResearchGraphIndustryChain  `json:"industry_chains"`
	IndustryChainMemberships []entitybiz.ResearchGraphMembership     `json:"industry_chain_memberships"`
	IndustryChainGraphEdges  []entitybiz.ResearchGraphIndustryEdge   `json:"industry_chain_graph_edges"`
}

type GraphValidationError = entitybiz.ResearchGraphValidationError

type UseCase struct {
	graphStore GraphStore
}

func NewUseCase(graphStore GraphStore) (*UseCase, error) {
	if graphStore == nil {
		return nil, errors.New("Research Graph store is required")
	}
	return &UseCase{graphStore: graphStore}, nil
}

func (s *UseCase) Search(ctx context.Context, request GraphSearchRequest) (GraphSearchResult, error) {
	if s == nil || s.graphStore == nil {
		return GraphSearchResult{}, errors.New("research graph store is required")
	}
	query, normalized, err := validateGraphSearchRequest(request)
	if err != nil {
		return GraphSearchResult{}, err
	}
	graph, err := s.graphStore.SearchResearchGraph(ctx, query)
	if err != nil {
		var limit *entitybiz.ResearchGraphResourceLimitError
		if errors.As(err, &limit) {
			return GraphSearchResult{}, &ResearchResourceLimitError{
				Reason: limit.Reason, Component: limit.Component,
				ActualRows: limit.ActualRows, MaxRows: limit.MaxRows,
				ActualBytes: limit.ActualBytes, MaxBytes: limit.MaxBytes,
				RetryGuidance: limit.RetryGuidance,
			}
		}
		return GraphSearchResult{}, err
	}
	graph = normalizeSubgraph(graph)
	if graph.ActualDepth < 0 || graph.ActualDepth > query.MaxDepth || !referencesResolve(graph) {
		return GraphSearchResult{}, errors.New("research graph result is not reference complete")
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
		return GraphSearchResult{}, &ResearchResourceLimitError{
			Reason:        reason,
			Component:     component,
			ActualRows:    &actual,
			MaxRows:       &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	graphPayload, err := json.Marshal(graph)
	if err != nil {
		return GraphSearchResult{}, errors.New("research graph result is invalid")
	}
	if len(graphPayload) > GraphMaxResultBytes {
		actual := int64(len(graphPayload))
		maximum := int64(GraphMaxResultBytes)
		return GraphSearchResult{}, &ResearchResourceLimitError{
			Reason:        "research graph result exceeds the response budget",
			Component:     "research_graph_result",
			ActualBytes:   &actual,
			MaxBytes:      &maximum,
			RetryGuidance: "reduce_depth_relation_types_or_chain_scope",
		}
	}
	queryFingerprint, err := payloadFingerprint(normalized)
	if err != nil {
		return GraphSearchResult{}, err
	}
	graphFingerprint, err := payloadFingerprint(graph)
	if err != nil {
		return GraphSearchResult{}, err
	}
	return GraphSearchResult{
		ContractVersion:          GraphContractVersion,
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

type normalizedGraphSearchRequest struct {
	GraphContractVersion       string           `json:"contract_version"`
	GraphStableOrderingVersion string           `json:"stable_ordering_version"`
	AnalysisAsOf               string           `json:"analysis_as_of"`
	SeedEntityIDs              []string         `json:"seed_entity_ids"`
	RelationFilters            []RelationFilter `json:"relation_filters"`
	MaxDepth                   int              `json:"max_depth"`
	IndustryChainID            *string          `json:"industry_chain_id,omitempty"`
	NodeBudget                 int              `json:"node_budget"`
	EdgeBudget                 int              `json:"edge_budget"`
}

func validateGraphSearchRequest(request GraphSearchRequest) (GraphQuery, normalizedGraphSearchRequest, error) {
	asOf, err := time.Parse(time.RFC3339, request.AnalysisAsOf)
	if err != nil {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "analysis_as_of must be an RFC3339 UTC timestamp"}
	}
	_, offset := asOf.Zone()
	if offset != 0 {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "analysis_as_of must use UTC"}
	}
	if len(request.SeedEntityIDs) < 1 || len(request.SeedEntityIDs) > GraphMaxSeedEntities {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: fmt.Sprintf("seed_entity_ids must contain between 1 and %d IDs", GraphMaxSeedEntities)}
	}
	seedSet := map[string]struct{}{}
	for _, id := range request.SeedEntityIDs {
		if !entitybiz.IsObjectID(id) {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "seed_entity_ids contains an invalid Object ID"}
		}
		if _, exists := seedSet[id]; exists {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "seed_entity_ids must contain unique IDs"}
		}
		seedSet[id] = struct{}{}
	}
	seeds := sortedSet(seedSet)
	if len(request.RelationFilters) < 1 || len(request.RelationFilters) > GraphMaxRelationFilters {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: fmt.Sprintf("relation_filters must contain between 1 and %d filters", GraphMaxRelationFilters)}
	}
	filterSet := map[string]RelationFilter{}
	for _, filter := range request.RelationFilters {
		filter.RelationType = strings.TrimSpace(filter.RelationType)
		if filter.RelationType == "" || (filter.Direction != DirectionOutgoing && filter.Direction != DirectionIncoming && filter.Direction != DirectionBoth) {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "relation_filters is invalid"}
		}
		if _, exists := filterSet[filter.RelationType]; exists {
			return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "relation_filters must configure each relation_type exactly once"}
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
	if request.MaxDepth < 1 || request.MaxDepth > GraphMaxDepth {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: fmt.Sprintf("max_depth must be between 1 and %d", GraphMaxDepth)}
	}
	if request.NodeBudget < 1 || request.NodeBudget > GraphMaxNodeBudget || request.EdgeBudget < 1 || request.EdgeBudget > GraphMaxEdgeBudget {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "node_budget or edge_budget exceeds the supported range"}
	}
	if request.IndustryChainID != nil && !entitybiz.IsIndustryChainID(*request.IndustryChainID) {
		return GraphQuery{}, normalizedGraphSearchRequest{}, &GraphValidationError{Reason: "industry_chain_id must be an IndustryChain ID"}
	}
	asOf = asOf.UTC()
	normalized := normalizedGraphSearchRequest{
		GraphContractVersion:       GraphContractVersion,
		GraphStableOrderingVersion: GraphStableOrderingVersion,
		AnalysisAsOf:               asOf.Format(time.RFC3339Nano),
		SeedEntityIDs:              seeds,
		RelationFilters:            filters,
		MaxDepth:                   request.MaxDepth,
		IndustryChainID:            request.IndustryChainID,
		NodeBudget:                 request.NodeBudget,
		EdgeBudget:                 request.EdgeBudget,
	}
	return GraphQuery{
		AnalysisAsOf:    asOf,
		SeedEntityIDs:   seeds,
		RelationFilters: filters,
		MaxDepth:        request.MaxDepth,
		IndustryChainID: request.IndustryChainID,
		NodeBudget:      request.NodeBudget,
		EdgeBudget:      request.EdgeBudget,
		FactPolicy:      entitybiz.ApprovedActiveResearchGraphFactPolicy(),
	}, normalized, nil
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func payloadFingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode research payload: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode research payload: %w", err)
	}
	var canonical bytes.Buffer
	if err := writeCanonicalJSON(&canonical, decoded); err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func writeCanonicalJSON(writer *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		writer.WriteString("null")
	case string:
		return writeCanonicalString(writer, typed)
	case json.Number:
		writer.WriteString(typed.String())
	case []any:
		writer.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writeCanonicalJSON(writer, item); err != nil {
				return err
			}
		}
		writer.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		writer.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				writer.WriteByte(',')
			}
			if err := writeCanonicalString(writer, key); err != nil {
				return err
			}
			writer.WriteByte(':')
			if err := writeCanonicalJSON(writer, typed[key]); err != nil {
				return err
			}
		}
		writer.WriteByte('}')
	default:
		return fmt.Errorf("unsupported V1 canonical JSON value %T", value)
	}
	return nil
}

func writeCanonicalString(writer *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return errors.New("canonical JSON string contains invalid UTF-8")
	}
	const hexadecimal = "0123456789abcdef"
	writer.WriteByte('"')
	for index := 0; index < len(value); index++ {
		character := value[index]
		switch character {
		case '"', '\\':
			writer.WriteByte('\\')
			writer.WriteByte(character)
		case '\b':
			writer.WriteString(`\b`)
		case '\t':
			writer.WriteString(`\t`)
		case '\n':
			writer.WriteString(`\n`)
		case '\f':
			writer.WriteString(`\f`)
		case '\r':
			writer.WriteString(`\r`)
		default:
			if character < 0x20 {
				writer.WriteString(`\u00`)
				writer.WriteByte(hexadecimal[character>>4])
				writer.WriteByte(hexadecimal[character&0x0f])
				continue
			}
			writer.WriteByte(character)
		}
	}
	writer.WriteByte('"')
	return nil
}

func referencesResolve(graph GraphSubgraph) bool {
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
		if _, ok := entities[chain.IndustryChainID]; !ok {
			return false
		}
		chains[chain.IndustryChainID] = struct{}{}
	}
	memberships := map[string]struct{}{}
	for _, membership := range graph.IndustryChainMemberships {
		if _, ok := chains[membership.IndustryChainID]; !ok {
			return false
		}
		if _, ok := entities[membership.ChainNodeID]; !ok {
			return false
		}
		memberships[membership.IndustryChainID+"\x00"+membership.ChainNodeID] = struct{}{}
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
		if _, ok := memberships[edge.IndustryChainID+"\x00"+edge.FromChainNodeID]; !ok {
			return false
		}
		if _, ok := memberships[edge.IndustryChainID+"\x00"+edge.ToChainNodeID]; !ok {
			return false
		}
	}
	return true
}

func normalizeSubgraph(graph GraphSubgraph) GraphSubgraph {
	if graph.Entities == nil {
		graph.Entities = []entitybiz.ResearchGraphEntity{}
	}
	if graph.RelationDefinitions == nil {
		graph.RelationDefinitions = []entitybiz.ResearchGraphRelation{}
	}
	if graph.EntityRelations == nil {
		graph.EntityRelations = []entitybiz.ResearchGraphEntityRelation{}
	}
	if graph.IndustryChains == nil {
		graph.IndustryChains = []entitybiz.ResearchGraphIndustryChain{}
	}
	if graph.IndustryChainMemberships == nil {
		graph.IndustryChainMemberships = []entitybiz.ResearchGraphMembership{}
	}
	if graph.IndustryChainGraphEdges == nil {
		graph.IndustryChainGraphEdges = []entitybiz.ResearchGraphIndustryEdge{}
	}
	return graph
}
