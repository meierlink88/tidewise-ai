package researchanalysiscontext

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	ContractVersion             = "research-analysis-context.v1"
	TBoxContractVersion         = "event-semantics.phase-one@1"
	StableOrderingVersion       = "knowledge-available-at-event-id.v1"
	TemporalSemantics           = "retrospective_reconstruction"
	TemporalLimitation          = "Event semantics are filtered by analysis_as_of; TBox and relationship dictionaries are a current-state reconstruction and do not claim strict historical replay"
	MaxDiscoveryWindow          = 366 * 24 * time.Hour
	MaxEventSemanticBundleBytes = 512 * 1024
	MaxDictionaryBytes          = 4 * 1024 * 1024
	MaxPageBytes                = 8 * 1024 * 1024
	MaxEventSemanticBundleRows  = 1_000
	MaxDictionaryRows           = 50_000
)

type Request struct {
	DiscoveryWindowStart   string
	DiscoveryWindowEnd     string
	AnalysisAsOf           string
	PredictionHorizonStart *string
	PredictionHorizonEnd   *string
	PageSize               int
	Cursor                 string
}

type Result struct {
	ContractVersion             string                `json:"contract_version"`
	TBoxContractVersion         string                `json:"tbox_contract_version"`
	TemporalSemantics           string                `json:"temporal_semantics"`
	TemporalLimitation          string                `json:"temporal_limitation"`
	EventPageFingerprint        string                `json:"event_page_fingerprint"`
	ReferenceClosureFingerprint string                `json:"reference_closure_fingerprint"`
	DiscoveryWindowStart        string                `json:"discovery_window_start"`
	DiscoveryWindowEnd          string                `json:"discovery_window_end"`
	AnalysisAsOf                string                `json:"analysis_as_of"`
	PredictionHorizonStart      *string               `json:"prediction_horizon_start,omitempty"`
	PredictionHorizonEnd        *string               `json:"prediction_horizon_end,omitempty"`
	EventSemanticBundles        []EventSemanticBundle `json:"event_semantic_bundles"`
	Dictionaries                Dictionaries          `json:"dictionaries"`
	NextCursor                  string                `json:"next_cursor,omitempty"`
	HasMore                     bool                  `json:"has_more"`
}

type ValidationError struct {
	Reason string
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

func (e *ResourceLimitError) Error() string {
	return e.Reason
}

func (e *ValidationError) Error() string {
	return e.Reason
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, request Request) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("research Analysis Context store is required")
	}
	query, normalized, fingerprint, err := validateRequest(request)
	if err != nil {
		return Result{}, err
	}
	if request.Cursor != "" {
		decoded, err := decodeCursor(request.Cursor)
		if err != nil || decoded.Version != 2 ||
			decoded.Fingerprint != fingerprint {
			return Result{}, &ValidationError{Reason: "cursor does not match the Analysis Context query"}
		}
		after, err := parseUTC("cursor.knowledge_available_at", decoded.KnowledgeAvailableAt)
		if err != nil || !identity.IsUUID(decoded.EventID) {
			return Result{}, &ValidationError{Reason: "cursor is invalid"}
		}
		query.AfterKnowledgeAvailableAt = &after
		query.AfterEventID = decoded.EventID
	}
	page, err := s.store.ListBundles(ctx, query)
	if err != nil {
		return Result{}, err
	}
	dictionaries, err := s.store.ReferenceClosure(
		ctx,
		buildReferenceClosureQuery(query.AnalysisAsOf, page.Bundles),
	)
	if err != nil {
		return Result{}, err
	}
	page.Dictionaries = normalizeDictionaries(dictionaries)
	if !researchContextReferencesResolve(page) {
		return Result{}, ErrReferenceClosureInconsistent
	}
	result := Result{
		ContractVersion:        ContractVersion,
		TBoxContractVersion:    TBoxContractVersion,
		TemporalSemantics:      TemporalSemantics,
		TemporalLimitation:     TemporalLimitation,
		DiscoveryWindowStart:   normalized.DiscoveryWindowStart,
		DiscoveryWindowEnd:     normalized.DiscoveryWindowEnd,
		AnalysisAsOf:           normalized.AnalysisAsOf,
		PredictionHorizonStart: normalized.PredictionHorizonStart,
		PredictionHorizonEnd:   normalized.PredictionHorizonEnd,
		EventSemanticBundles:   make([]EventSemanticBundle, 0, len(page.Bundles)),
		Dictionaries:           page.Dictionaries,
		HasMore:                page.HasMore,
	}
	dictionaryPayload, err := json.Marshal(result.Dictionaries)
	if err != nil {
		return Result{}, errors.New("research Analysis Context dictionaries are invalid")
	}
	if len(dictionaryPayload) > MaxDictionaryBytes {
		return Result{}, &ResourceLimitError{
			Reason:        "Research Analysis Context reference closure exceeds the response budget",
			Component:     "reference_closure",
			ActualRows:    int64Reference(int64(dictionaryRows(result.Dictionaries))),
			MaxRows:       int64Reference(MaxDictionaryRows),
			ActualBytes:   int64Reference(int64(len(dictionaryPayload))),
			MaxBytes:      int64Reference(MaxDictionaryBytes),
			RetryGuidance: "reduce_page_size",
		}
	}
	pageBytes := len(dictionaryPayload)
	for _, bundle := range page.Bundles {
		if bundle.KnowledgeAvailableAt.IsZero() || !identity.IsUUID(bundle.EventID) ||
			bundle.Bundle.Event.ID != bundle.EventID {
			return Result{}, errors.New("research Analysis Context bundle is invalid")
		}
		bundlePayload, err := json.Marshal(bundle.Bundle)
		if err != nil {
			return Result{}, errors.New("research Analysis Context bundle is invalid")
		}
		if len(bundlePayload) > MaxEventSemanticBundleBytes {
			return Result{}, &ResourceLimitError{
				Reason:        "an Event Semantic Bundle exceeds the response budget",
				Component:     "event_semantic_bundle",
				ActualBytes:   int64Reference(int64(len(bundlePayload))),
				MaxBytes:      int64Reference(MaxEventSemanticBundleBytes),
				RetryGuidance: "event_bundle_requires_provider_remediation",
			}
		}
		pageBytes += len(bundlePayload)
		if pageBytes > MaxPageBytes {
			return Result{}, &ResourceLimitError{
				Reason:        "Research Analysis Context page exceeds the response budget",
				Component:     "analysis_context_page",
				ActualBytes:   int64Reference(int64(pageBytes)),
				MaxBytes:      int64Reference(MaxPageBytes),
				RetryGuidance: "reduce_page_size",
			}
		}
		result.EventSemanticBundles = append(result.EventSemanticBundles, bundle.Bundle)
	}
	result.EventPageFingerprint, err = payloadFingerprint(result.EventSemanticBundles)
	if err != nil {
		return Result{}, errors.New("research Analysis Context Event page is invalid")
	}
	result.ReferenceClosureFingerprint, err = payloadFingerprint(result.Dictionaries)
	if err != nil {
		return Result{}, errors.New("research Analysis Context reference closure is invalid")
	}
	if page.HasMore {
		if len(page.Bundles) == 0 {
			return Result{}, errors.New("research Analysis Context continuation has no terminal bundle")
		}
		last := page.Bundles[len(page.Bundles)-1]
		result.NextCursor, err = encodeCursor(contextCursor{
			Version:              2,
			Fingerprint:          fingerprint,
			KnowledgeAvailableAt: last.KnowledgeAvailableAt.UTC().Format(time.RFC3339Nano),
			EventID:              last.EventID,
		})
		if err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

func buildReferenceClosureQuery(
	analysisAsOf time.Time,
	bundles []BundleRecord,
) ReferenceClosureQuery {
	entityIDs := map[string]struct{}{}
	relationIDs := map[string]struct{}{}
	variableDefinitions := map[string]VersionedReference{}
	rules := map[string]VersionedReference{}
	submissionIDs := map[string]struct{}{}
	for _, record := range bundles {
		for _, link := range record.Bundle.EntityLinks {
			entityIDs[link.EntityID] = struct{}{}
			if link.SemanticSubmissionID != "" {
				submissionIDs[link.SemanticSubmissionID] = struct{}{}
			}
		}
		for _, signal := range record.Bundle.VariableSignals {
			entityIDs[signal.SubjectEntityID] = struct{}{}
			variable := VersionedReference{Key: signal.VariableKey, Version: signal.VariableVersion}
			variableDefinitions[versionedKey(variable.Key, variable.Version)] = variable
			if signal.SemanticSubmissionID != "" {
				submissionIDs[signal.SemanticSubmissionID] = struct{}{}
			}
			for _, impact := range signal.DirectImpacts {
				entityIDs[impact.TargetEntityID] = struct{}{}
				affected := VersionedReference{
					Key: impact.AffectedVariableKey, Version: impact.AffectedVariableVersion,
				}
				variableDefinitions[versionedKey(affected.Key, affected.Version)] = affected
				if impact.EntityRelationID != nil {
					relationIDs[*impact.EntityRelationID] = struct{}{}
				}
				if impact.RuleKey != nil && impact.RuleVersion != nil {
					rule := VersionedReference{Key: *impact.RuleKey, Version: *impact.RuleVersion}
					rules[versionedKey(rule.Key, rule.Version)] = rule
				}
				if impact.SemanticSubmissionID != "" {
					submissionIDs[impact.SemanticSubmissionID] = struct{}{}
				}
			}
		}
	}
	return ReferenceClosureQuery{
		AnalysisAsOf:            analysisAsOf,
		EntityIDs:               sortedSet(entityIDs),
		EntityRelationIDs:       sortedSet(relationIDs),
		VariableDefinitions:     sortedVersionedReferences(variableDefinitions),
		DirectTransmissionRules: sortedVersionedReferences(rules),
		SemanticSubmissionIDs:   sortedSet(submissionIDs),
	}
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func sortedVersionedReferences(
	values map[string]VersionedReference,
) []VersionedReference {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]VersionedReference, 0, len(keys))
	for _, key := range keys {
		result = append(result, values[key])
	}
	return result
}

func researchContextReferencesResolve(page StorePage) bool {
	entityTypes := make(map[string]struct{}, len(page.Dictionaries.EntityTypeDefinitions))
	for _, definition := range page.Dictionaries.EntityTypeDefinitions {
		entityTypes[definition.TypeKey] = struct{}{}
	}
	entities := make(map[string]struct{}, len(page.Dictionaries.Entities))
	for _, entity := range page.Dictionaries.Entities {
		if !containsID(entityTypes, entity.EntityType) {
			return false
		}
		entities[entity.EntityID] = struct{}{}
	}
	relationTypes := make(map[string]struct{}, len(page.Dictionaries.RelationDefinitions))
	for _, definition := range page.Dictionaries.RelationDefinitions {
		relationTypes[definition.RelationType] = struct{}{}
	}
	relations := make(map[string]EntityRelation, len(page.Dictionaries.EntityRelations))
	for _, relation := range page.Dictionaries.EntityRelations {
		if !containsID(entities, relation.FromEntityID) ||
			!containsID(entities, relation.ToEntityID) ||
			!containsID(relationTypes, relation.RelationType) {
			return false
		}
		relations[relation.EntityRelationID] = relation
	}
	variables := make(map[string]struct{}, len(page.Dictionaries.VariableDefinitions))
	for _, definition := range page.Dictionaries.VariableDefinitions {
		for _, entityType := range definition.ApplicableEntityTypes {
			if !containsID(entityTypes, entityType) {
				return false
			}
		}
		variables[versionedKey(definition.Key, definition.Version)] = struct{}{}
	}
	rules := make(map[string]struct{}, len(page.Dictionaries.DirectTransmissionRules))
	for _, rule := range page.Dictionaries.DirectTransmissionRules {
		if !containsID(variables, versionedKey(rule.SourceVariableKey, rule.SourceVariableVersion)) ||
			!containsID(variables, versionedKey(rule.AffectedVariableKey, rule.AffectedVariableVersion)) ||
			!containsID(entityTypes, rule.SourceEntityType) ||
			!containsID(entityTypes, rule.TargetEntityType) ||
			!containsID(relationTypes, rule.RelationType) {
			return false
		}
		rules[versionedKey(rule.RuleKey, rule.Version)] = struct{}{}
	}
	chains := make(map[string]struct{}, len(page.Dictionaries.IndustryChains))
	for _, chain := range page.Dictionaries.IndustryChains {
		if !containsID(entities, chain.IndustryChainEntityID) {
			return false
		}
		chains[chain.IndustryChainEntityID] = struct{}{}
	}
	for _, entity := range page.Dictionaries.Entities {
		if entity.EntityType == "industry_chain" &&
			!containsID(chains, entity.EntityID) {
			return false
		}
	}
	memberships := make(map[string]struct{}, len(page.Dictionaries.IndustryChainMemberships))
	for _, membership := range page.Dictionaries.IndustryChainMemberships {
		if !containsID(chains, membership.IndustryChainEntityID) ||
			!containsID(entities, membership.ChainNodeEntityID) {
			return false
		}
		memberships[membership.IndustryChainEntityID+"\x00"+membership.ChainNodeEntityID] = struct{}{}
	}
	for _, edge := range page.Dictionaries.IndustryChainGraphEdges {
		if !containsID(chains, edge.IndustryChainEntityID) ||
			!containsID(memberships, edge.IndustryChainEntityID+"\x00"+edge.FromChainNodeEntityID) ||
			!containsID(memberships, edge.IndustryChainEntityID+"\x00"+edge.ToChainNodeEntityID) {
			return false
		}
	}
	for _, record := range page.Bundles {
		bundle := record.Bundle
		evidence := make(map[string]struct{}, len(bundle.Evidence))
		for _, item := range bundle.Evidence {
			evidence[item.EvidenceID] = struct{}{}
		}
		links := make(map[string]EntityLink, len(bundle.EntityLinks))
		for _, link := range bundle.EntityLinks {
			if !containsID(entities, link.EntityID) || !allIDsResolve(evidence, link.EvidenceIDs) {
				return false
			}
			links[link.EventEntityLinkID] = link
		}
		signals := make(map[string]struct{}, len(bundle.VariableSignals))
		for _, signal := range bundle.VariableSignals {
			link, ok := links[signal.SubjectEventEntityLinkID]
			if !ok || link.EntityID != signal.SubjectEntityID ||
				!containsID(entities, signal.SubjectEntityID) ||
				!containsID(variables, versionedKey(signal.VariableKey, signal.VariableVersion)) ||
				!allIDsResolve(evidence, signal.EvidenceIDs) {
				return false
			}
			for _, measurement := range signal.Measurements {
				if !containsID(evidence, measurement.EvidenceID) {
					return false
				}
			}
			signals[signal.VariableSignalID] = struct{}{}
			for _, impact := range signal.DirectImpacts {
				if impact.SourceVariableSignalID != signal.VariableSignalID ||
					!containsID(entities, impact.TargetEntityID) ||
					!containsID(variables, versionedKey(
						impact.AffectedVariableKey, impact.AffectedVariableVersion,
					)) ||
					!allIDsResolve(evidence, impact.EvidenceIDs) {
					return false
				}
				if impact.EntityRelationID != nil {
					if _, ok := relations[*impact.EntityRelationID]; !ok {
						return false
					}
				}
				if impact.RuleKey != nil && impact.RuleVersion != nil &&
					!containsID(rules, versionedKey(*impact.RuleKey, *impact.RuleVersion)) {
					return false
				}
			}
		}
		if len(signals) != len(bundle.VariableSignals) {
			return false
		}
	}
	return true
}

func versionedKey(key string, version int) string {
	return fmt.Sprintf("%s@%d", key, version)
}

func containsID(set map[string]struct{}, id string) bool {
	_, ok := set[id]
	return ok
}

func allIDsResolve(set map[string]struct{}, ids []string) bool {
	for _, id := range ids {
		if !containsID(set, id) {
			return false
		}
	}
	return true
}

type normalizedRequest struct {
	DiscoveryWindowStart   string
	DiscoveryWindowEnd     string
	AnalysisAsOf           string
	PredictionHorizonStart *string
	PredictionHorizonEnd   *string
}

func validateRequest(
	request Request,
) (StoreQuery, normalizedRequest, string, error) {
	if request.PageSize < 1 || request.PageSize > 50 {
		return StoreQuery{}, normalizedRequest{}, "", &ValidationError{
			Reason: "page_size must be between 1 and 50",
		}
	}
	start, err := parseUTC("discovery_window_start", request.DiscoveryWindowStart)
	if err != nil {
		return StoreQuery{}, normalizedRequest{}, "", err
	}
	end, err := parseUTC("discovery_window_end", request.DiscoveryWindowEnd)
	if err != nil {
		return StoreQuery{}, normalizedRequest{}, "", err
	}
	asOf, err := parseUTC("analysis_as_of", request.AnalysisAsOf)
	if err != nil {
		return StoreQuery{}, normalizedRequest{}, "", err
	}
	if !start.Before(end) {
		return StoreQuery{}, normalizedRequest{}, "", &ValidationError{
			Reason: "discovery_window_end must be after discovery_window_start",
		}
	}
	if end.Sub(start) > MaxDiscoveryWindow {
		return StoreQuery{}, normalizedRequest{}, "", &ResourceLimitError{
			Reason:        "discovery window exceeds the maximum technical budget of 366 days",
			Component:     "analysis_context_query",
			RetryGuidance: "reduce_discovery_window",
		}
	}
	if end.After(asOf) {
		return StoreQuery{}, normalizedRequest{}, "", &ValidationError{
			Reason: "discovery_window_end must not be after analysis_as_of",
		}
	}
	predictionStart, predictionEnd, err := predictionWindow(
		request.PredictionHorizonStart, request.PredictionHorizonEnd, asOf,
	)
	if err != nil {
		return StoreQuery{}, normalizedRequest{}, "", err
	}
	normalized := normalizedRequest{
		DiscoveryWindowStart: start.Format(time.RFC3339Nano),
		DiscoveryWindowEnd:   end.Format(time.RFC3339Nano),
		AnalysisAsOf:         asOf.Format(time.RFC3339Nano),
	}
	if predictionStart != nil {
		value := predictionStart.Format(time.RFC3339Nano)
		normalized.PredictionHorizonStart = &value
		value = predictionEnd.Format(time.RFC3339Nano)
		normalized.PredictionHorizonEnd = &value
	}
	fingerprint := queryFingerprint(normalized, request.PageSize)
	return StoreQuery{
		DiscoveryWindowStart: start, DiscoveryWindowEnd: end, AnalysisAsOf: asOf,
		PredictionHorizonStart: predictionStart, PredictionHorizonEnd: predictionEnd,
		PageSize: request.PageSize,
	}, normalized, fingerprint, nil
}

func predictionWindow(
	rawStart *string,
	rawEnd *string,
	asOf time.Time,
) (*time.Time, *time.Time, error) {
	if (rawStart == nil) != (rawEnd == nil) {
		return nil, nil, &ValidationError{
			Reason: "prediction horizon start and end must be provided together",
		}
	}
	if rawStart == nil {
		return nil, nil, nil
	}
	start, err := parseUTC("prediction_horizon_start", *rawStart)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseUTC("prediction_horizon_end", *rawEnd)
	if err != nil {
		return nil, nil, err
	}
	if start.Before(asOf) || !start.Before(end) {
		return nil, nil, &ValidationError{
			Reason: "prediction horizon must start at or after analysis_as_of and end after its start",
		}
	}
	return &start, &end, nil
}

func parseUTC(name, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, &ValidationError{Reason: fmt.Sprintf("%s must be an RFC3339 UTC timestamp", name)}
	}
	_, offset := parsed.Zone()
	if offset != 0 {
		return time.Time{}, &ValidationError{Reason: fmt.Sprintf("%s must use UTC", name)}
	}
	return parsed.UTC(), nil
}

func queryFingerprint(request normalizedRequest, pageSize int) string {
	payload, _ := json.Marshal(struct {
		ContractVersion       string            `json:"contract_version"`
		TBoxContractVersion   string            `json:"tbox_contract_version"`
		StableOrderingVersion string            `json:"stable_ordering_version"`
		Request               normalizedRequest `json:"request"`
		PageSize              int               `json:"page_size"`
	}{
		ContractVersion:       ContractVersion,
		TBoxContractVersion:   TBoxContractVersion,
		StableOrderingVersion: StableOrderingVersion,
		Request:               request,
		PageSize:              pageSize,
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func payloadFingerprint(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func normalizeDictionaries(value Dictionaries) Dictionaries {
	if value.Entities == nil {
		value.Entities = []Entity{}
	}
	if value.RelationDefinitions == nil {
		value.RelationDefinitions = []RelationDefinition{}
	}
	if value.EntityRelations == nil {
		value.EntityRelations = []EntityRelation{}
	}
	if value.IndustryChains == nil {
		value.IndustryChains = []IndustryChain{}
	}
	if value.IndustryChainMemberships == nil {
		value.IndustryChainMemberships = []IndustryChainMembership{}
	}
	if value.IndustryChainGraphEdges == nil {
		value.IndustryChainGraphEdges = []IndustryChainGraphEdge{}
	}
	if value.EntityTypeDefinitions == nil {
		value.EntityTypeDefinitions = []EntityTypeDefinition{}
	}
	if value.VariableDefinitions == nil {
		value.VariableDefinitions = []VariableDefinition{}
	}
	if value.DirectTransmissionRules == nil {
		value.DirectTransmissionRules = []DirectTransmissionRule{}
	}
	if value.AcceptancePolicies == nil {
		value.AcceptancePolicies = []AcceptancePolicy{}
	}
	return value
}

type contextCursor struct {
	Version              int    `json:"v"`
	Fingerprint          string `json:"fingerprint"`
	KnowledgeAvailableAt string `json:"knowledge_available_at"`
	EventID              string `json:"event_id"`
}

func encodeCursor(cursor contextCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(raw string) (contextCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return contextCursor{}, err
	}
	var cursor contextCursor
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return contextCursor{}, err
	}
	return cursor, nil
}

func dictionaryRows(value Dictionaries) int {
	return len(value.Entities) +
		len(value.RelationDefinitions) +
		len(value.EntityRelations) +
		len(value.IndustryChains) +
		len(value.IndustryChainMemberships) +
		len(value.IndustryChainGraphEdges) +
		len(value.EntityTypeDefinitions) +
		len(value.VariableDefinitions) +
		len(value.DirectTransmissionRules) +
		len(value.AcceptancePolicies)
}

func int64Reference(value int64) *int64 {
	return &value
}
