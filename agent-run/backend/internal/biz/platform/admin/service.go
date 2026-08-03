package admin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

var (
	ErrUnknownTarget = errors.New("configuration target is not registered")
	ErrInvalidConfig = errors.New("configuration update is invalid")
)

type Registry struct {
	ModelProviderKeys []string
	ConnectorKeys     []string
}

type Store interface {
	LoadModelProviderConfigs(context.Context) (map[string]agentrun.ModelProviderConfig, error)
	ListModelProviderConfigViews(context.Context) ([]agentrun.ModelProviderConfigView, error)
	UpsertModelProviderConfig(context.Context, agentrun.ModelProviderConfig) error
	LoadConnectorConfigs(context.Context) (map[string]agentrun.ConnectorConfig, error)
	ListConnectorConfigViews(context.Context) ([]agentrun.ConnectorConfigView, error)
	UpsertConnectorConfig(context.Context, agentrun.ConnectorConfig) error
	ListAgentExecutions(context.Context, agentrun.ExecutionListQuery) (agentrun.ExecutionPage, error)
	ListAgentStatuses(context.Context) ([]agentrun.AgentStatus, error)
	ListMonitoringStatusCounts(context.Context, time.Time) ([]agentrun.MonitoringStatusCount, error)
	GetMonitoringBusinessTotals(context.Context, time.Time) (agentrun.MonitoringBusinessTotals, error)
	ListCollectorMonitoring(context.Context, agentrun.MonitoringListQuery) (agentrun.CollectorMonitoringPage, error)
	ListArtifactExtractionMonitoring(context.Context, agentrun.MonitoringListQuery) (agentrun.ArtifactExtractionMonitoringPage, error)
	ListSemanticMonitoring(context.Context, agentrun.MonitoringListQuery) (agentrun.SemanticMonitoringPage, error)
}

type Service struct {
	store         Store
	modelKeys     []string
	modelSet      map[string]struct{}
	connectorKeys []string
	connectorSet  map[string]struct{}
	environment   string
	schedules     ScheduleManager
	now           func() time.Time
}

type ScheduleManager interface {
	List(context.Context) ([]agentrun.AgentSchedule, error)
	Get(context.Context, string) (agentrun.AgentSchedule, error)
	Put(context.Context, agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error)
	Patch(context.Context, string, agentrun.PatchAgentScheduleInput) (agentrun.AgentSchedule, error)
}

type Option func(*Service)

func WithScheduleManager(manager ScheduleManager) Option {
	return func(service *Service) {
		service.schedules = manager
	}
}

func WithNow(now func() time.Time) Option {
	return func(service *Service) {
		if now != nil {
			service.now = now
		}
	}
}

type ModelProviderPatch struct {
	BaseURL *string
	Model   *string
	APIKey  *string
}

type ConnectorPatch struct {
	BaseURL *string
	APIKey  *string
}

func New(store Store, registry Registry, environment string, options ...Option) (*Service, error) {
	if store == nil {
		return nil, errors.New("Admin configuration Store is required")
	}
	if environment != "dev" && environment != "uat" {
		return nil, errors.New("Admin configuration environment is unsupported")
	}
	modelKeys, modelSet, err := normalizeRegistry(registry.ModelProviderKeys)
	if err != nil {
		return nil, fmt.Errorf("Model Provider registry: %w", err)
	}
	connectorKeys, connectorSet, err := normalizeRegistry(registry.ConnectorKeys)
	if err != nil {
		return nil, fmt.Errorf("Connector registry: %w", err)
	}
	service := &Service{
		store: store, modelKeys: modelKeys, modelSet: modelSet,
		connectorKeys: connectorKeys, connectorSet: connectorSet, environment: environment,
		now: time.Now,
	}
	for _, option := range options {
		option(service)
	}
	return service, nil
}

func (s *Service) MonitoringSummary(ctx context.Context, window agentrun.MonitoringWindow) (agentrun.MonitoringSummary, error) {
	since, generatedAt, ok := s.monitoringSince(window)
	if !ok {
		return agentrun.MonitoringSummary{}, errors.New("Monitoring window is invalid")
	}
	rawCounts, err := s.store.ListMonitoringStatusCounts(ctx, since)
	if err != nil {
		return agentrun.MonitoringSummary{}, err
	}
	business, err := s.store.GetMonitoringBusinessTotals(ctx, since)
	if err != nil {
		return agentrun.MonitoringSummary{}, err
	}
	result := agentrun.MonitoringSummary{
		Window: window, GeneratedAt: generatedAt, Business: business,
		Collector: agentrun.MonitoringStageSummary{Kind: agentrun.MonitoringCollector},
		Artifact:  agentrun.MonitoringStageSummary{Kind: agentrun.MonitoringArtifactExtraction},
		Semantic:  agentrun.MonitoringStageSummary{Kind: agentrun.MonitoringSemantic},
	}
	for _, raw := range rawCounts {
		state, known := agentrun.MonitoringStateForStatus(raw.Kind, raw.Status)
		if !known {
			continue
		}
		stage := monitoringStage(&result, raw.Kind)
		switch state {
		case agentrun.MonitoringStateSuccess:
			stage.Counts.Success += raw.Count
		case agentrun.MonitoringStateRunning:
			stage.Counts.Running += raw.Count
		case agentrun.MonitoringStateFailure:
			stage.Counts.Failure += raw.Count
		}
	}
	return result, nil
}

func (s *Service) ListCollectorMonitoring(ctx context.Context, window agentrun.MonitoringWindow, state agentrun.MonitoringState, page, pageSize int) (agentrun.CollectorMonitoringPage, error) {
	query, ok := s.monitoringQuery(window, state, agentrun.MonitoringCollector, page, pageSize)
	if !ok {
		return agentrun.CollectorMonitoringPage{}, errors.New("Monitoring query is invalid")
	}
	return s.store.ListCollectorMonitoring(ctx, query)
}

func (s *Service) ListArtifactExtractionMonitoring(ctx context.Context, window agentrun.MonitoringWindow, state agentrun.MonitoringState, page, pageSize int) (agentrun.ArtifactExtractionMonitoringPage, error) {
	query, ok := s.monitoringQuery(window, state, agentrun.MonitoringArtifactExtraction, page, pageSize)
	if !ok {
		return agentrun.ArtifactExtractionMonitoringPage{}, errors.New("Monitoring query is invalid")
	}
	return s.store.ListArtifactExtractionMonitoring(ctx, query)
}

func (s *Service) ListSemanticMonitoring(ctx context.Context, window agentrun.MonitoringWindow, state agentrun.MonitoringState, page, pageSize int) (agentrun.SemanticMonitoringPage, error) {
	query, ok := s.monitoringQuery(window, state, agentrun.MonitoringSemantic, page, pageSize)
	if !ok {
		return agentrun.SemanticMonitoringPage{}, errors.New("Monitoring query is invalid")
	}
	return s.store.ListSemanticMonitoring(ctx, query)
}

func (s *Service) monitoringQuery(window agentrun.MonitoringWindow, state agentrun.MonitoringState, kind agentrun.MonitoringKind, page, pageSize int) (agentrun.MonitoringListQuery, bool) {
	since, _, ok := s.monitoringSince(window)
	statuses, statusOK := agentrun.MonitoringStatuses(kind, state)
	if !ok || !statusOK || page < 1 || pageSize < 1 || pageSize > 100 {
		return agentrun.MonitoringListQuery{}, false
	}
	return agentrun.MonitoringListQuery{Since: since, Statuses: statuses, Page: page, PageSize: pageSize}, true
}

func (s *Service) monitoringSince(window agentrun.MonitoringWindow) (time.Time, time.Time, bool) {
	duration, ok := window.Duration()
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	now := s.now().UTC()
	return now.Add(-duration), now, true
}

func monitoringStage(summary *agentrun.MonitoringSummary, kind agentrun.MonitoringKind) *agentrun.MonitoringStageSummary {
	switch kind {
	case agentrun.MonitoringArtifactExtraction:
		return &summary.Artifact
	case agentrun.MonitoringSemantic:
		return &summary.Semantic
	default:
		return &summary.Collector
	}
}

func (s *Service) ListAgentSchedules(ctx context.Context) ([]agentrun.AgentSchedule, error) {
	if s.schedules == nil {
		return nil, errors.New("Agent Schedule manager is unavailable")
	}
	return s.schedules.List(ctx)
}

func (s *Service) GetAgentSchedule(ctx context.Context, agentKey string) (agentrun.AgentSchedule, error) {
	if s.schedules == nil {
		return agentrun.AgentSchedule{}, errors.New("Agent Schedule manager is unavailable")
	}
	return s.schedules.Get(ctx, agentKey)
}

func (s *Service) PutAgentSchedule(ctx context.Context, input agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error) {
	if s.schedules == nil {
		return agentrun.AgentSchedule{}, errors.New("Agent Schedule manager is unavailable")
	}
	return s.schedules.Put(ctx, input)
}

func (s *Service) PatchAgentSchedule(
	ctx context.Context,
	agentKey string,
	input agentrun.PatchAgentScheduleInput,
) (agentrun.AgentSchedule, error) {
	if s.schedules == nil {
		return agentrun.AgentSchedule{}, errors.New("Agent Schedule manager is unavailable")
	}
	return s.schedules.Patch(ctx, agentKey, input)
}

func (s *Service) ListAgentExecutions(ctx context.Context, query agentrun.ExecutionListQuery) (agentrun.ExecutionPage, error) {
	return s.store.ListAgentExecutions(ctx, query)
}

func (s *Service) ListAgentStatuses(ctx context.Context) ([]agentrun.AgentStatus, error) {
	return s.store.ListAgentStatuses(ctx)
}

func (s *Service) ListModelProviders(ctx context.Context) ([]agentrun.ModelProviderConfigView, error) {
	views, err := s.store.ListModelProviderConfigViews(ctx)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]agentrun.ModelProviderConfigView, len(views))
	for _, view := range views {
		existing[view.ProviderKey] = view
	}
	result := make([]agentrun.ModelProviderConfigView, 0, len(s.modelKeys))
	for _, key := range s.modelKeys {
		view, exists := existing[key]
		if !exists {
			view = agentrun.ModelProviderConfigView{ProviderKey: key}
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) GetModelProvider(ctx context.Context, key string) (agentrun.ModelProviderConfigView, error) {
	if _, exists := s.modelSet[key]; !exists {
		return agentrun.ModelProviderConfigView{}, ErrUnknownTarget
	}
	views, err := s.ListModelProviders(ctx)
	if err != nil {
		return agentrun.ModelProviderConfigView{}, err
	}
	for _, view := range views {
		if view.ProviderKey == key {
			return view, nil
		}
	}
	return agentrun.ModelProviderConfigView{}, ErrUnknownTarget
}

func (s *Service) PatchModelProvider(ctx context.Context, key string, patch ModelProviderPatch) (agentrun.ModelProviderConfigView, error) {
	if _, exists := s.modelSet[key]; !exists {
		return agentrun.ModelProviderConfigView{}, ErrUnknownTarget
	}
	configs, err := s.store.LoadModelProviderConfigs(ctx)
	if err != nil {
		return agentrun.ModelProviderConfigView{}, err
	}
	config := configs[key]
	config.ProviderKey = key
	if patch.BaseURL != nil {
		config.BaseURL = strings.TrimSpace(*patch.BaseURL)
	}
	if patch.Model != nil {
		config.Model = strings.TrimSpace(*patch.Model)
	}
	if patch.APIKey != nil {
		if *patch.APIKey == "" {
			return agentrun.ModelProviderConfigView{}, fmt.Errorf("%w: Model Provider Key cannot be cleared", ErrInvalidConfig)
		}
		config.APIKey = *patch.APIKey
	}
	if err := s.validateModel(config); err != nil {
		return agentrun.ModelProviderConfigView{}, err
	}
	if err := s.store.UpsertModelProviderConfig(ctx, config); err != nil {
		return agentrun.ModelProviderConfigView{}, err
	}
	return s.GetModelProvider(ctx, key)
}

func (s *Service) ListConnectors(ctx context.Context) ([]agentrun.ConnectorConfigView, error) {
	views, err := s.store.ListConnectorConfigViews(ctx)
	if err != nil {
		return nil, err
	}
	existing := make(map[string]agentrun.ConnectorConfigView, len(views))
	for _, view := range views {
		existing[view.ConnectorKey] = view
	}
	result := make([]agentrun.ConnectorConfigView, 0, len(s.connectorKeys))
	for _, key := range s.connectorKeys {
		view, exists := existing[key]
		if !exists {
			view = agentrun.ConnectorConfigView{ConnectorKey: key}
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) GetConnector(ctx context.Context, key string) (agentrun.ConnectorConfigView, error) {
	if _, exists := s.connectorSet[key]; !exists {
		return agentrun.ConnectorConfigView{}, ErrUnknownTarget
	}
	views, err := s.ListConnectors(ctx)
	if err != nil {
		return agentrun.ConnectorConfigView{}, err
	}
	for _, view := range views {
		if view.ConnectorKey == key {
			return view, nil
		}
	}
	return agentrun.ConnectorConfigView{}, ErrUnknownTarget
}

func (s *Service) PatchConnector(ctx context.Context, key string, patch ConnectorPatch) (agentrun.ConnectorConfigView, error) {
	if _, exists := s.connectorSet[key]; !exists {
		return agentrun.ConnectorConfigView{}, ErrUnknownTarget
	}
	configs, err := s.store.LoadConnectorConfigs(ctx)
	if err != nil {
		return agentrun.ConnectorConfigView{}, err
	}
	config := configs[key]
	config.ConnectorKey = key
	if patch.BaseURL != nil {
		config.BaseURL = strings.TrimSpace(*patch.BaseURL)
	}
	if patch.APIKey != nil {
		config.APIKey = *patch.APIKey
	}
	if err := s.validateConnector(config); err != nil {
		return agentrun.ConnectorConfigView{}, err
	}
	if err := s.store.UpsertConnectorConfig(ctx, config); err != nil {
		return agentrun.ConnectorConfigView{}, err
	}
	return s.GetConnector(ctx, key)
}

func (s *Service) validateModel(config agentrun.ModelProviderConfig) error {
	if err := s.validateURL(config.BaseURL); err != nil {
		return err
	}
	if strings.TrimSpace(config.Model) == "" {
		return fmt.Errorf("%w: Model Provider model is required", ErrInvalidConfig)
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return fmt.Errorf("%w: Model Provider Key is required", ErrInvalidConfig)
	}
	return nil
}

func (s *Service) validateConnector(config agentrun.ConnectorConfig) error {
	return s.validateURL(config.BaseURL)
}

func (s *Service) validateURL(value string) error {
	if !agentrun.ConfigurationBaseURLValid(value, s.environment) {
		return fmt.Errorf("%w: Base URL is not allowed", ErrInvalidConfig)
	}
	return nil
}

func normalizeRegistry(keys []string) ([]string, map[string]struct{}, error) {
	set := make(map[string]struct{}, len(keys))
	for _, raw := range keys {
		key := strings.TrimSpace(raw)
		if key == "" {
			return nil, nil, errors.New("registered key is blank")
		}
		if _, exists := set[key]; exists {
			return nil, nil, fmt.Errorf("duplicate registered key %q", key)
		}
		set[key] = struct{}{}
	}
	normalized := make([]string, 0, len(set))
	for key := range set {
		normalized = append(normalized, key)
	}
	sort.Strings(normalized)
	return normalized, set, nil
}
