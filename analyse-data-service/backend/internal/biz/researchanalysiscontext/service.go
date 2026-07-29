package researchanalysiscontext

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/identity"
)

const (
	ContractVersion             = "research-analysis-context.v1"
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
	ContractVersion        string                `json:"contract_version"`
	TemporalSemantics      string                `json:"temporal_semantics"`
	TemporalLimitation     string                `json:"temporal_limitation"`
	DictionaryFingerprint  string                `json:"dictionary_fingerprint"`
	DiscoveryWindowStart   string                `json:"discovery_window_start"`
	DiscoveryWindowEnd     string                `json:"discovery_window_end"`
	AnalysisAsOf           string                `json:"analysis_as_of"`
	PredictionHorizonStart *string               `json:"prediction_horizon_start,omitempty"`
	PredictionHorizonEnd   *string               `json:"prediction_horizon_end,omitempty"`
	EventSemanticBundles   []EventSemanticBundle `json:"event_semantic_bundles"`
	Dictionaries           Dictionaries          `json:"dictionaries"`
	NextCursor             string                `json:"next_cursor,omitempty"`
	HasMore                bool                  `json:"has_more"`
}

type ValidationError struct {
	Reason string
}

type ResourceLimitError struct {
	Reason string
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
	var continuation *contextCursor
	if request.Cursor != "" {
		decoded, err := decodeCursor(request.Cursor)
		if err != nil || decoded.Version != 1 ||
			decoded.Fingerprint != fingerprint {
			return Result{}, &ValidationError{Reason: "cursor does not match the Analysis Context query"}
		}
		continuation = &decoded
		after, err := parseUTC("cursor.knowledge_available_at", decoded.KnowledgeAvailableAt)
		if err != nil || !identity.IsUUID(decoded.EventID) {
			return Result{}, &ValidationError{Reason: "cursor is invalid"}
		}
		query.AfterKnowledgeAvailableAt = &after
		query.AfterEventID = decoded.EventID
	}
	page, err := s.store.List(ctx, query)
	if err != nil {
		return Result{}, err
	}
	if !hashPattern(page.DictionaryFingerprint) {
		return Result{}, errors.New("research Analysis Context dictionary fingerprint is invalid")
	}
	if !researchContextReferencesResolve(page) {
		return Result{}, ErrHistoricalSemanticsUnavailable
	}
	if continuation != nil &&
		continuation.DictionaryFingerprint != page.DictionaryFingerprint {
		return Result{}, &ValidationError{
			Reason: "cursor is stale because the Analysis Context dictionary changed; restart from the first page",
		}
	}
	result := Result{
		ContractVersion:        ContractVersion,
		TemporalSemantics:      TemporalSemantics,
		TemporalLimitation:     TemporalLimitation,
		DictionaryFingerprint:  page.DictionaryFingerprint,
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
			Reason: "Research Analysis Context dictionaries exceed the response budget",
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
				Reason: "an Event Semantic Bundle exceeds the response budget",
			}
		}
		pageBytes += len(bundlePayload)
		if pageBytes > MaxPageBytes {
			return Result{}, &ResourceLimitError{
				Reason: "Research Analysis Context page exceeds the response budget",
			}
		}
		result.EventSemanticBundles = append(result.EventSemanticBundles, bundle.Bundle)
	}
	if page.HasMore {
		if len(page.Bundles) == 0 {
			return Result{}, errors.New("research Analysis Context continuation has no terminal bundle")
		}
		last := page.Bundles[len(page.Bundles)-1]
		result.NextCursor, err = encodeCursor(contextCursor{
			Version:               1,
			Fingerprint:           fingerprint,
			DictionaryFingerprint: page.DictionaryFingerprint,
			KnowledgeAvailableAt:  last.KnowledgeAvailableAt.UTC().Format(time.RFC3339Nano),
			EventID:               last.EventID,
		})
		if err != nil {
			return Result{}, err
		}
	}
	return result, nil
}

func researchContextReferencesResolve(page StorePage) bool {
	entities := make(map[string]struct{}, len(page.Dictionaries.Entities))
	for _, entity := range page.Dictionaries.Entities {
		entities[entity.EntityID] = struct{}{}
	}
	relations := make(map[string]EntityRelation, len(page.Dictionaries.EntityRelations))
	for _, relation := range page.Dictionaries.EntityRelations {
		if !containsID(entities, relation.FromEntityID) ||
			!containsID(entities, relation.ToEntityID) {
			return false
		}
		relations[relation.EntityRelationID] = relation
	}
	variables := make(map[string]struct{}, len(page.Dictionaries.VariableDefinitions))
	for _, definition := range page.Dictionaries.VariableDefinitions {
		variables[versionedKey(definition.Key, definition.Version)] = struct{}{}
	}
	rules := make(map[string]struct{}, len(page.Dictionaries.DirectTransmissionRules))
	for _, rule := range page.Dictionaries.DirectTransmissionRules {
		if !containsID(variables, versionedKey(rule.SourceVariableKey, rule.SourceVariableVersion)) ||
			!containsID(variables, versionedKey(rule.AffectedVariableKey, rule.AffectedVariableVersion)) {
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
			Reason: "discovery window exceeds the maximum technical budget of 366 days",
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
		ContractVersion string            `json:"contract_version"`
		Request         normalizedRequest `json:"request"`
		PageSize        int               `json:"page_size"`
	}{
		ContractVersion: ContractVersion,
		Request:         request,
		PageSize:        pageSize,
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

type contextCursor struct {
	Version               int    `json:"v"`
	Fingerprint           string `json:"fingerprint"`
	DictionaryFingerprint string `json:"dictionary_fingerprint"`
	KnowledgeAvailableAt  string `json:"knowledge_available_at"`
	EventID               string `json:"event_id"`
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

func hashPattern(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			if character < 'a' || character > 'f' {
				return false
			}
		}
	}
	return true
}
