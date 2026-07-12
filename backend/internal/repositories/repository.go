package repositories

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type SourceCatalogFilter struct {
	SourceID      string
	ProviderKey   string
	IngestChannel string
	SourceType    string
	Limit         int
}

type SourceCatalogRepository interface {
	ActiveSources(context.Context, SourceCatalogFilter) ([]domain.SourceCatalog, error)
	SourceCatalogStats(context.Context) (SourceCatalogStats, error)
}

type SourceCatalogStats struct {
	Total           int
	ByProviderKey   map[string]int
	ByIngestChannel map[string]int
	BySourceType    map[string]int
	ByUsagePolicy   map[string]int
	ByStatus        map[string]int
}

type RawDocumentRepository interface {
	UpsertRawDocument(context.Context, domain.RawDocument) (RawDocumentWriteResult, error)
	UpdateRawDocumentStatus(context.Context, string, domain.IngestStatus) error
}

type RawDocumentWriteResult struct {
	Document    domain.RawDocument
	Created     bool
	DuplicateOf string
}

type BenchmarkObservationRepository interface {
	UpsertBenchmarkObservation(context.Context, domain.BenchmarkObservation) (BenchmarkObservationWriteResult, error)
	ListBenchmarkObservations(context.Context, BenchmarkObservationFilter) ([]domain.BenchmarkObservation, error)
}

type BenchmarkObservationWriteResult struct {
	Observation domain.BenchmarkObservation
	Created     bool
}

type BenchmarkObservationFilter struct {
	BenchmarkEntityID string
	Limit             int
}

type RawDocumentListFilter struct {
	Title    string
	Page     int
	PageSize int
}

type RawDocumentPage struct {
	Items    []domain.RawDocument
	Total    int
	Page     int
	PageSize int
}

type EventListFilter struct {
	Title         string
	EventStatus   domain.EventStatus
	FactStatus    domain.FactStatus
	EventTimeFrom *time.Time
	EventTimeTo   *time.Time
	FirstSeenFrom *time.Time
	FirstSeenTo   *time.Time
	Page          int
	PageSize      int
}

type EventPage struct {
	Items    []domain.Event
	Total    int
	Page     int
	PageSize int
}

type SourceCatalogListFilter struct {
	Status domain.SourceCatalogStatus
}

type AdminQueryRepository interface {
	ListRawDocuments(context.Context, RawDocumentListFilter) (RawDocumentPage, error)
	ListEvents(context.Context, EventListFilter) (EventPage, error)
	ListSourceCatalogs(context.Context, SourceCatalogListFilter) ([]domain.SourceCatalog, error)
}

type SchedulerRepository interface {
	LoadSchedulerConfig(context.Context) (domain.SchedulerConfig, error)
	SaveSchedulerConfig(context.Context, domain.SchedulerConfig) (domain.SchedulerConfig, error)
	CreateIngestionRun(context.Context, domain.IngestionRun) (domain.IngestionRun, error)
	RecordIngestionRunSource(context.Context, domain.IngestionRunSource) error
	CompleteIngestionRun(context.Context, domain.IngestionRun) error
	RecentIngestionRuns(context.Context, int) ([]domain.IngestionRun, error)
}

type GraphProjectionType string

const (
	GraphProjectionTypeEntityGraph GraphProjectionType = "entity_graph"
)

type GraphProjectionMode string

const (
	GraphProjectionModeProjectEntities GraphProjectionMode = "project_entities"
	GraphProjectionModeRebuildEntities GraphProjectionMode = "rebuild_entities"
)

type GraphProjectionRunStatus string

const (
	GraphProjectionRunStatusRunning   GraphProjectionRunStatus = "running"
	GraphProjectionRunStatusSucceeded GraphProjectionRunStatus = "succeeded"
	GraphProjectionRunStatusFailed    GraphProjectionRunStatus = "failed"
	GraphProjectionRunStatusPartial   GraphProjectionRunStatus = "partial"
)

type GraphProjectionRunItemType string

const (
	GraphProjectionRunItemTypeEntity       GraphProjectionRunItemType = "entity_node"
	GraphProjectionRunItemTypeRelationship GraphProjectionRunItemType = "entity_relationship"
)

type GraphProjectionRunItemStatus string

const (
	GraphProjectionRunItemStatusProjected GraphProjectionRunItemStatus = "projected"
	GraphProjectionRunItemStatusSkipped   GraphProjectionRunItemStatus = "skipped"
	GraphProjectionRunItemStatusFailed    GraphProjectionRunItemStatus = "failed"
)

type GraphEntityNode struct {
	ID            string
	EntityKey     string
	EntityType    domain.EntityType
	LayerCode     string
	Name          string
	CanonicalName string
	Aliases       []string
	Status        domain.Status
	UpdatedAt     time.Time
}

type GraphEntityEdge struct {
	ID           string
	FromEntityID string
	ToEntityID   string
	RelationType string
	EvidenceNote string
	Status       domain.Status
	UpdatedAt    time.Time
}

type GraphProjectionRun struct {
	ID             string
	ProjectionType GraphProjectionType
	Mode           GraphProjectionMode
	Status         GraphProjectionRunStatus
	StartedAt      time.Time
	FinishedAt     *time.Time
	SourceRowCount int
	ProjectedCount int
	SkippedCount   int
	FailedCount    int
	ErrorSummary   string
	ConfigSummary  map[string]any
}

type GraphProjectionRunItem struct {
	ID           string
	RunID        string
	ItemType     GraphProjectionRunItemType
	ItemKey      string
	Status       GraphProjectionRunItemStatus
	ErrorMessage string
}

type GraphProjectionRepository interface {
	ListGraphEntityNodes(context.Context) ([]GraphEntityNode, error)
	ListGraphEntityEdges(context.Context) ([]GraphEntityEdge, error)
	CreateGraphProjectionRun(context.Context, GraphProjectionRun) (GraphProjectionRun, error)
	RecordGraphProjectionRunItem(context.Context, GraphProjectionRunItem) error
	CompleteGraphProjectionRun(context.Context, GraphProjectionRun) error
	RecentGraphProjectionRuns(context.Context, int) ([]GraphProjectionRun, error)
}

type InMemoryRepository struct {
	mu              sync.Mutex
	sources         []domain.SourceCatalog
	documents       map[string]domain.RawDocument
	events          map[string]domain.Event
	schedulerConfig domain.SchedulerConfig
	ingestionRuns   map[string]domain.IngestionRun
	runSources      map[string][]domain.IngestionRunSource
	graphEntities   map[string]GraphEntityNode
	graphEdges      map[string]GraphEntityEdge
	graphRuns       map[string]GraphProjectionRun
	graphRunItems   map[string][]GraphProjectionRunItem
	observations    map[string]domain.BenchmarkObservation
}

func NewInMemoryRepository(sources []domain.SourceCatalog) *InMemoryRepository {
	copiedSources := make([]domain.SourceCatalog, len(sources))
	for index, source := range sources {
		copiedSources[index] = cloneSource(normalizeInMemorySource(source))
	}

	return &InMemoryRepository{
		sources:         copiedSources,
		documents:       map[string]domain.RawDocument{},
		events:          map[string]domain.Event{},
		schedulerConfig: defaultSchedulerConfig(),
		ingestionRuns:   map[string]domain.IngestionRun{},
		runSources:      map[string][]domain.IngestionRunSource{},
		graphEntities:   map[string]GraphEntityNode{},
		graphEdges:      map[string]GraphEntityEdge{},
		graphRuns:       map[string]GraphProjectionRun{},
		graphRunItems:   map[string][]GraphProjectionRunItem{},
		observations:    map[string]domain.BenchmarkObservation{},
	}
}

func (r *InMemoryRepository) SeedSource(_ context.Context, source domain.SourceCatalog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	source = cloneSource(normalizeInMemorySource(source))
	for index, existing := range r.sources {
		if existing.ID == source.ID {
			r.sources[index] = source
			return nil
		}
	}
	r.sources = append(r.sources, source)
	return nil
}

func (r *InMemoryRepository) SeedEvent(_ context.Context, event domain.Event) error {
	if err := event.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.events[event.ID] = cloneEvent(event)
	return nil
}

func (r *InMemoryRepository) ActiveSources(_ context.Context, filter SourceCatalogFilter) ([]domain.SourceCatalog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []domain.SourceCatalog
	for _, source := range r.sources {
		if source.Status != domain.SourceCatalogStatusActive {
			continue
		}
		if filter.SourceID != "" && source.ID != filter.SourceID {
			continue
		}
		if filter.ProviderKey != "" && source.ProviderKey != filter.ProviderKey {
			continue
		}
		if filter.IngestChannel != "" && source.IngestChannel != filter.IngestChannel {
			continue
		}
		if filter.SourceType != "" && source.SourceType != filter.SourceType {
			continue
		}
		result = append(result, cloneSource(normalizeInMemorySource(source)))
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result, nil
}

func (r *InMemoryRepository) SourceCatalogStats(_ context.Context) (SourceCatalogStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := newSourceCatalogStats()
	for _, source := range r.sources {
		stats.Total++
		incrementStats(stats.ByProviderKey, source.ProviderKey)
		incrementStats(stats.ByIngestChannel, source.IngestChannel)
		incrementStats(stats.BySourceType, source.SourceType)
		incrementStats(stats.ByUsagePolicy, source.UsagePolicy)
		incrementStats(stats.ByStatus, string(source.Status))
	}
	return stats, nil
}

func (r *InMemoryRepository) SeedGraphEntity(entity GraphEntityNode) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.graphEntities[entity.ID] = normalizeGraphEntityNode(entity)
}

func (r *InMemoryRepository) SeedGraphEdge(edge GraphEntityEdge) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.graphEdges[edge.ID] = edge
}

func (r *InMemoryRepository) UpsertBenchmarkObservation(_ context.Context, observation domain.BenchmarkObservation) (BenchmarkObservationWriteResult, error) {
	if err := observation.Validate(); err != nil {
		return BenchmarkObservationWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	entity, ok := r.graphEntities[observation.BenchmarkEntityID]
	if !ok {
		return BenchmarkObservationWriteResult{}, fmt.Errorf("benchmark entity %q not found", observation.BenchmarkEntityID)
	}
	if entity.EntityType != domain.EntityTypeBenchmark {
		return BenchmarkObservationWriteResult{}, fmt.Errorf("entity %q type %q is not benchmark", observation.BenchmarkEntityID, entity.EntityType)
	}

	key := benchmarkObservationKey(observation.BenchmarkEntityID, observation.ObservedAt, observation.SourceName)
	existing, ok := r.observations[key]
	if ok {
		observation.ID = existing.ID
		r.observations[key] = observation
		return BenchmarkObservationWriteResult{Observation: observation, Created: false}, nil
	}
	r.observations[key] = observation
	return BenchmarkObservationWriteResult{Observation: observation, Created: true}, nil
}

func (r *InMemoryRepository) ListBenchmarkObservations(_ context.Context, filter BenchmarkObservationFilter) ([]domain.BenchmarkObservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	observations := make([]domain.BenchmarkObservation, 0, len(r.observations))
	for _, observation := range r.observations {
		if filter.BenchmarkEntityID != "" && observation.BenchmarkEntityID != filter.BenchmarkEntityID {
			continue
		}
		observations = append(observations, observation)
	}
	sort.SliceStable(observations, func(i, j int) bool {
		if !observations[i].ObservedAt.Equal(observations[j].ObservedAt) {
			return observations[i].ObservedAt.After(observations[j].ObservedAt)
		}
		return observations[i].SourceName < observations[j].SourceName
	})
	if filter.Limit > 0 && len(observations) > filter.Limit {
		observations = observations[:filter.Limit]
	}
	return observations, nil
}

func benchmarkObservationKey(benchmarkEntityID string, observedAt time.Time, sourceName string) string {
	return strings.Join([]string{benchmarkEntityID, observedAt.UTC().Format(time.RFC3339Nano), strings.ToLower(strings.TrimSpace(sourceName))}, "|")
}

func (r *InMemoryRepository) ListGraphEntityNodes(context.Context) ([]GraphEntityNode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := make([]GraphEntityNode, 0, len(r.graphEntities))
	for _, node := range r.graphEntities {
		if node.Status != domain.StatusActive {
			continue
		}
		nodes = append(nodes, normalizeGraphEntityNode(node))
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes, nil
}

func (r *InMemoryRepository) ListGraphEntityEdges(context.Context) ([]GraphEntityEdge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	edges := make([]GraphEntityEdge, 0, len(r.graphEdges))
	for _, edge := range r.graphEdges {
		from, fromOK := r.graphEntities[edge.FromEntityID]
		to, toOK := r.graphEntities[edge.ToEntityID]
		if edge.Status != domain.StatusActive || !fromOK || !toOK || from.Status != domain.StatusActive || to.Status != domain.StatusActive {
			continue
		}
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})
	return edges, nil
}

func (r *InMemoryRepository) CreateGraphProjectionRun(_ context.Context, run GraphProjectionRun) (GraphProjectionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	run = normalizeGraphProjectionRun(run)
	r.graphRuns[run.ID] = cloneGraphProjectionRun(run)
	return cloneGraphProjectionRun(run), nil
}

func (r *InMemoryRepository) RecordGraphProjectionRunItem(_ context.Context, item GraphProjectionRunItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.graphRuns[item.RunID]; !ok {
		return fmt.Errorf("graph projection run %q not found", item.RunID)
	}
	r.graphRunItems[item.RunID] = append(r.graphRunItems[item.RunID], item)
	return nil
}

func (r *InMemoryRepository) CompleteGraphProjectionRun(_ context.Context, run GraphProjectionRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.graphRuns[run.ID]; !ok {
		return fmt.Errorf("graph projection run %q not found", run.ID)
	}
	r.graphRuns[run.ID] = cloneGraphProjectionRun(normalizeGraphProjectionRun(run))
	return nil
}

func (r *InMemoryRepository) RecentGraphProjectionRuns(_ context.Context, limit int) ([]GraphProjectionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	runs := make([]GraphProjectionRun, 0, len(r.graphRuns))
	for _, run := range r.graphRuns {
		runs = append(runs, cloneGraphProjectionRun(run))
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (r *InMemoryRepository) UpsertRawDocument(_ context.Context, doc domain.RawDocument) (RawDocumentWriteResult, error) {
	if err := doc.Validate(); err != nil {
		return RawDocumentWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.findDuplicate(doc); ok {
		return RawDocumentWriteResult{
			Document:    existing,
			Created:     false,
			DuplicateOf: existing.ID,
		}, nil
	}

	r.documents[doc.ID] = doc
	return RawDocumentWriteResult{
		Document: doc,
		Created:  true,
	}, nil
}

func (r *InMemoryRepository) UpdateRawDocumentStatus(_ context.Context, id string, status domain.IngestStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, ok := r.documents[id]
	if !ok {
		return fmt.Errorf("raw document %q not found", id)
	}
	doc.IngestStatus = status
	r.documents[id] = doc

	return nil
}

func (r *InMemoryRepository) RawDocument(id string) (domain.RawDocument, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, ok := r.documents[id]
	return doc, ok
}

func (r *InMemoryRepository) RawDocumentCount(_ context.Context, sourceID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, doc := range r.documents {
		if sourceID == "" || doc.SourceID == sourceID {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryRepository) ListRawDocuments(_ context.Context, filter RawDocumentListFilter) (RawDocumentPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	title := strings.ToLower(strings.TrimSpace(filter.Title))
	items := make([]domain.RawDocument, 0, len(r.documents))
	for _, doc := range r.documents {
		if title != "" && !strings.Contains(strings.ToLower(doc.Title), title) {
			continue
		}
		items = append(items, cloneRawDocument(doc))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CollectedAt.After(items[j].CollectedAt)
	})
	total := len(items)
	items = pageSlice(items, page, pageSize)
	return RawDocumentPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *InMemoryRepository) ListEvents(_ context.Context, filter EventListFilter) (EventPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	title := strings.ToLower(strings.TrimSpace(filter.Title))
	items := make([]domain.Event, 0, len(r.events))
	for _, event := range r.events {
		if title != "" && !strings.Contains(strings.ToLower(event.Title), title) {
			continue
		}
		if filter.EventStatus != "" && event.EventStatus != filter.EventStatus {
			continue
		}
		if filter.FactStatus != "" && event.FactStatus != filter.FactStatus {
			continue
		}
		if filter.EventTimeFrom != nil {
			if event.EventTime == nil || event.EventTime.Before(*filter.EventTimeFrom) {
				continue
			}
		}
		if filter.EventTimeTo != nil {
			if event.EventTime == nil || event.EventTime.After(*filter.EventTimeTo) {
				continue
			}
		}
		if filter.FirstSeenFrom != nil && event.FirstSeenAt.Before(*filter.FirstSeenFrom) {
			continue
		}
		if filter.FirstSeenTo != nil && event.FirstSeenAt.After(*filter.FirstSeenTo) {
			continue
		}
		items = append(items, cloneEvent(event))
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].FirstSeenAt.Equal(items[j].FirstSeenAt) {
			return items[i].FirstSeenAt.After(items[j].FirstSeenAt)
		}
		if items[i].EventTime == nil {
			return false
		}
		if items[j].EventTime == nil {
			return true
		}
		return items[i].EventTime.After(*items[j].EventTime)
	})
	total := len(items)
	items = pageSlice(items, page, pageSize)
	return EventPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *InMemoryRepository) ListSourceCatalogs(_ context.Context, filter SourceCatalogListFilter) ([]domain.SourceCatalog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := make([]domain.SourceCatalog, 0, len(r.sources))
	for _, source := range r.sources {
		if filter.Status != "" && source.Status != filter.Status {
			continue
		}
		items = append(items, cloneSource(normalizeInMemorySource(source)))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ProviderKey != items[j].ProviderKey {
			return items[i].ProviderKey < items[j].ProviderKey
		}
		if items[i].SourceName != items[j].SourceName {
			return items[i].SourceName < items[j].SourceName
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (r *InMemoryRepository) LoadSchedulerConfig(_ context.Context) (domain.SchedulerConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return cloneSchedulerConfig(r.schedulerConfig), nil
}

func (r *InMemoryRepository) SaveSchedulerConfig(_ context.Context, config domain.SchedulerConfig) (domain.SchedulerConfig, error) {
	config = normalizeSchedulerConfig(config)
	if err := config.Validate(); err != nil {
		return domain.SchedulerConfig{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if config.ConfigVersion <= 0 {
		config.ConfigVersion = 1
	}
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now
	r.schedulerConfig = cloneSchedulerConfig(config)
	return cloneSchedulerConfig(r.schedulerConfig), nil
}

func (r *InMemoryRepository) CreateIngestionRun(_ context.Context, run domain.IngestionRun) (domain.IngestionRun, error) {
	if err := run.Validate(); err != nil {
		return domain.IngestionRun{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	run.UpdatedAt = now
	run.SchedulerConfig = cloneMap(run.SchedulerConfig)
	r.ingestionRuns[run.ID] = run
	return cloneIngestionRun(run), nil
}

func (r *InMemoryRepository) RecordIngestionRunSource(_ context.Context, result domain.IngestionRunSource) error {
	if err := result.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.ingestionRuns[result.RunID]; !ok {
		return fmt.Errorf("ingestion run %q not found", result.RunID)
	}
	now := time.Now()
	if result.CreatedAt.IsZero() {
		result.CreatedAt = now
	}
	result.UpdatedAt = now
	r.runSources[result.RunID] = append(r.runSources[result.RunID], result)
	return nil
}

func (r *InMemoryRepository) CompleteIngestionRun(_ context.Context, run domain.IngestionRun) error {
	if err := run.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.ingestionRuns[run.ID]; !ok {
		return fmt.Errorf("ingestion run %q not found", run.ID)
	}
	run.UpdatedAt = time.Now()
	run.SchedulerConfig = cloneMap(run.SchedulerConfig)
	r.ingestionRuns[run.ID] = run

	config := r.schedulerConfig
	config.LastRunID = run.ID
	config.LastRunAt = &run.StartedAt
	config.UpdatedAt = run.UpdatedAt
	r.schedulerConfig = config
	return nil
}

func (r *InMemoryRepository) RecentIngestionRuns(_ context.Context, limit int) ([]domain.IngestionRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limit <= 0 {
		limit = 20
	}
	runs := make([]domain.IngestionRun, 0, len(r.ingestionRuns))
	for _, run := range r.ingestionRuns {
		runs = append(runs, cloneIngestionRun(run))
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (r *InMemoryRepository) IngestionRunSources(runID string) []domain.IngestionRunSource {
	r.mu.Lock()
	defer r.mu.Unlock()

	items := r.runSources[runID]
	copied := make([]domain.IngestionRunSource, len(items))
	copy(copied, items)
	return copied
}

func (r *InMemoryRepository) findDuplicate(doc domain.RawDocument) (domain.RawDocument, bool) {
	for _, existing := range r.documents {
		if existing.SourceID != doc.SourceID {
			continue
		}
		if doc.SourceExternalID != "" && existing.SourceExternalID == doc.SourceExternalID {
			return existing, true
		}
		if existing.ContentHash == doc.ContentHash {
			return existing, true
		}
	}

	return domain.RawDocument{}, false
}

func cloneSource(source domain.SourceCatalog) domain.SourceCatalog {
	source.SourceConfig = cloneMap(source.SourceConfig)
	source.RateLimitPolicy = cloneMap(source.RateLimitPolicy)
	return source
}

func cloneRawDocument(doc domain.RawDocument) domain.RawDocument {
	if doc.PublishedAt != nil {
		value := *doc.PublishedAt
		doc.PublishedAt = &value
	}
	return doc
}

func cloneEvent(event domain.Event) domain.Event {
	if event.EventTime != nil {
		value := *event.EventTime
		event.EventTime = &value
	}
	if event.KnowableAt != nil {
		value := *event.KnowableAt
		event.KnowableAt = &value
	}
	return event
}

func normalizeInMemorySource(source domain.SourceCatalog) domain.SourceCatalog {
	if source.SourceLevel == "" {
		source.SourceLevel = "secondary"
	}
	if source.AuthType == "" {
		source.AuthType = "none"
	}
	if source.Status == "" {
		source.Status = domain.SourceCatalogStatusActive
	}
	if source.SourceConfig == nil {
		source.SourceConfig = map[string]any{}
	}
	if source.RateLimitPolicy == nil {
		source.RateLimitPolicy = map[string]any{}
	}
	return source
}

func normalizeGraphEntityNode(node GraphEntityNode) GraphEntityNode {
	node.Aliases = append([]string(nil), node.Aliases...)
	if node.EntityKey == "" && node.EntityType != "" && node.ID != "" {
		node.EntityKey = fmt.Sprintf("%s:%s", node.EntityType, node.ID)
	}
	if node.Status == "" {
		node.Status = domain.StatusActive
	}
	return node
}

func normalizeGraphProjectionRun(run GraphProjectionRun) GraphProjectionRun {
	if run.ProjectionType == "" {
		run.ProjectionType = GraphProjectionTypeEntityGraph
	}
	if run.Mode == "" {
		run.Mode = GraphProjectionModeProjectEntities
	}
	if run.Status == "" {
		run.Status = GraphProjectionRunStatusRunning
	}
	if run.ConfigSummary == nil {
		run.ConfigSummary = map[string]any{}
	}
	return run
}

func cloneGraphProjectionRun(run GraphProjectionRun) GraphProjectionRun {
	if run.FinishedAt != nil {
		value := *run.FinishedAt
		run.FinishedAt = &value
	}
	run.ConfigSummary = cloneMap(run.ConfigSummary)
	return run
}

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	copied := make(map[string]any, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

func defaultSchedulerConfig() domain.SchedulerConfig {
	return domain.SchedulerConfig{
		ID:              "default",
		Enabled:         false,
		Mode:            domain.SchedulerModeInterval,
		IntervalMinutes: 60,
		Concurrency:     1,
		BatchSize:       10,
		TimeoutSeconds:  180,
		SourceFilter:    domain.SchedulerSourceFilter{},
		Timezone:        "Asia/Shanghai",
		ConfigVersion:   1,
	}
}

func normalizeSchedulerConfig(config domain.SchedulerConfig) domain.SchedulerConfig {
	if config.ID == "" {
		config.ID = "default"
	}
	if config.Mode == "" {
		config.Mode = domain.SchedulerModeInterval
	}
	if config.Mode == domain.SchedulerModeInterval && config.IntervalMinutes == 0 {
		config.IntervalMinutes = 60
	}
	if config.Concurrency == 0 {
		config.Concurrency = 1
	}
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 180
	}
	if config.Timezone == "" {
		config.Timezone = "Asia/Shanghai"
	}
	if config.ConfigVersion == 0 {
		config.ConfigVersion = 1
	}
	return config
}

func cloneSchedulerConfig(config domain.SchedulerConfig) domain.SchedulerConfig {
	config.FixedTimes = append([]string(nil), config.FixedTimes...)
	return config
}

func cloneIngestionRun(run domain.IngestionRun) domain.IngestionRun {
	run.SchedulerConfig = cloneMap(run.SchedulerConfig)
	return run
}

func normalizePage(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	return page, pageSize
}

func pageSlice[T any](items []T, page int, pageSize int) []T {
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func newSourceCatalogStats() SourceCatalogStats {
	return SourceCatalogStats{
		ByProviderKey:   map[string]int{},
		ByIngestChannel: map[string]int{},
		BySourceType:    map[string]int{},
		ByUsagePolicy:   map[string]int{},
		ByStatus:        map[string]int{},
	}
}

func incrementStats(counts map[string]int, key string) {
	if key == "" {
		key = "unknown"
	}
	counts[key]++
}
