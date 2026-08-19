package source

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

const (
	MaxSources              = 200
	MaxSnapshotEnvelopeSize = 500_000
	MaxConfigBytes          = 4_096
)

type OwnershipType string
type ChannelType string
type AdapterKey string
type SourceLevel string

const (
	OwnershipFixed   OwnershipType = "fixed"
	OwnershipDynamic OwnershipType = "dynamic"

	ChannelWebSearch ChannelType = "web_search"
	ChannelAPI       ChannelType = "api"
	ChannelRSS       ChannelType = "rss"

	AdapterBocha          AdapterKey = "bocha"
	AdapterTavily         AdapterKey = "tavily"
	AdapterParallel       AdapterKey = "parallel"
	AdapterCLS            AdapterKey = "cls"
	AdapterEastmoneyFast  AdapterKey = "eastmoney_fast"
	AdapterEastmoneyStock AdapterKey = "eastmoney_stock"
	AdapterSTCN           AdapterKey = "stcn"
	AdapterGenericRSS     AdapterKey = "generic_rss"

	SourceLevelOfficial SourceLevel = "L1_OFFICIAL"
	SourceLevelWire     SourceLevel = "L2_WIRE"
	SourceLevelMedia    SourceLevel = "L3_MEDIA"
	SourceLevelSocial   SourceLevel = "L4_SOCIAL"
)

var (
	ErrNotFound             = errors.New("Source not found")
	ErrConflict             = errors.New("Source conflict")
	ErrFixedDeleteForbidden = errors.New("fixed Source cannot be deleted")
	ErrCapacityExceeded     = errors.New("Source capacity exceeded")
	ErrPersistence          = errors.New("Source persistence failed")
	codePattern             = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type Source struct {
	ID                 string          `json:"id"`
	Code               string          `json:"code"`
	Name               string          `json:"name"`
	OwnershipType      OwnershipType   `json:"ownership_type"`
	ChannelType        ChannelType     `json:"channel_type"`
	AdapterKey         AdapterKey      `json:"adapter_key"`
	Enabled            bool            `json:"enabled"`
	Endpoint           string          `json:"endpoint"`
	AppKey             *string         `json:"app_key"`
	Config             json.RawMessage `json:"config"`
	Priority           int             `json:"priority"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxResults         int             `json:"max_results"`
	DefaultSourceLevel SourceLevel     `json:"default_source_level"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type MutableSource struct {
	Code               string
	Name               string
	AdapterKey         AdapterKey
	Enabled            bool
	Endpoint           string
	AppKey             *string
	Config             json.RawMessage
	Priority           int
	TimeoutSeconds     int
	MaxResults         int
	DefaultSourceLevel SourceLevel
}

type FixedManifestOptions struct {
	Endpoints map[string]string
	AppKeys   map[string]string
}

func CurrentFixedManifest(options FixedManifestOptions) []Source {
	type definition struct {
		code, name, endpoint string
		channel              ChannelType
		adapter              AdapterKey
		enabled              bool
		level                SourceLevel
		credential           bool
	}
	definitions := []definition{
		{"bocha", "博查", "https://api.bochaai.com/v1/web-search", ChannelWebSearch, AdapterBocha, true, SourceLevelMedia, true},
		{"tavily", "Tavily", "https://api.tavily.com/search", ChannelWebSearch, AdapterTavily, false, SourceLevelMedia, true},
		{"parallel_search", "Parallel Search", "https://api.parallel.ai/v1/search", ChannelWebSearch, AdapterParallel, false, SourceLevelMedia, true},
		{"cls_telegraph", "财联社电报", "https://www.cls.cn/v1/roll/get_roll_list", ChannelAPI, AdapterCLS, true, SourceLevelWire, false},
		{"eastmoney_fastnews", "东方财富 7x24", "https://np-weblist.eastmoney.com/comm/web/getFastNewsList", ChannelAPI, AdapterEastmoneyFast, true, SourceLevelMedia, false},
		{"eastmoney_stock_news", "东方财富个股新闻", "https://search-api-web.eastmoney.com/search/jsonp", ChannelAPI, AdapterEastmoneyStock, true, SourceLevelMedia, false},
		{"stcn_quicknews", "证券时报快讯", "https://www.stcn.com/article/list.html", ChannelAPI, AdapterSTCN, true, SourceLevelMedia, false},
	}
	result := make([]Source, 0, len(definitions))
	for _, item := range definitions {
		endpoint := item.endpoint
		if override := strings.TrimSpace(options.Endpoints[item.code]); override != "" {
			endpoint = override
		}
		var appKey *string
		if item.credential {
			if value := strings.TrimSpace(options.AppKeys[item.code]); value != "" {
				appKey = &value
			}
		}
		result = append(result, Source{
			Code: item.code, Name: item.name, OwnershipType: OwnershipFixed,
			ChannelType: item.channel, AdapterKey: item.adapter, Enabled: item.enabled,
			Endpoint: endpoint, AppKey: appKey, Config: json.RawMessage(`{}`), Priority: 1,
			TimeoutSeconds: 30, MaxResults: 10, DefaultSourceLevel: item.level,
		})
	}
	return result
}

type UseCase struct{ store Store }

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Source store is required")
	}
	return &UseCase{store: store}, nil
}

func (s *UseCase) CreateDynamic(ctx context.Context, input MutableSource) (Source, error) {
	id, err := coreid.Derive(coreid.Source, "source", input.Code)
	if err != nil {
		return Source{}, &ValidationError{Field: "code", Message: "must be a stable Source code"}
	}
	candidate := sourceFromMutable(input)
	candidate.ID = id
	candidate.OwnershipType = OwnershipDynamic
	candidate.ChannelType = ChannelRSS
	candidate.AdapterKey = AdapterGenericRSS
	if err := validateSource(candidate, false); err != nil {
		return Source{}, err
	}
	var created Source
	err = s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		existing, err := tx.List(ctx)
		if err != nil {
			return err
		}
		for _, item := range existing {
			if item.Code == candidate.Code || item.ID == candidate.ID {
				return ErrConflict
			}
		}
		if err := validateProjectedSet(append(existing, candidate)); err != nil {
			return err
		}
		created, err = tx.Insert(ctx, candidate)
		return err
	})
	if err != nil {
		return Source{}, err
	}
	return cloneSource(created), nil
}

func (s *UseCase) PublishFixed(ctx context.Context, manifest []Source) ([]Source, error) {
	prepared := make([]Source, len(manifest))
	seen := make(map[string]struct{}, len(manifest))
	for index, item := range manifest {
		item.OwnershipType = OwnershipFixed
		item, err := prepareSource(item, index, seen, false)
		if err != nil {
			return nil, err
		}
		prepared[index] = item
	}
	var result []Source
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		existing, err := tx.List(ctx)
		if err != nil {
			return err
		}
		byCode := make(map[string]Source, len(existing))
		for _, item := range existing {
			byCode[item.Code] = item
		}
		projected := cloneSources(existing)
		result = make([]Source, 0, len(prepared))
		for _, item := range prepared {
			if current, ok := byCode[item.Code]; ok {
				if current.ID != item.ID || current.OwnershipType != OwnershipFixed || current.ChannelType != item.ChannelType {
					return ErrConflict
				}
				result = append(result, cloneSource(current))
				continue
			}
			projected = append(projected, item)
			created, err := tx.Insert(ctx, item)
			if err != nil {
				return err
			}
			result = append(result, cloneSource(created))
		}
		return validateProjectedSet(projected)
	})
	if err != nil {
		return nil, err
	}
	return cloneSources(result), nil
}

func (s *UseCase) List(ctx context.Context) ([]Source, error) {
	items, err := s.store.List(ctx, false)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := validateSource(item, true); err != nil {
			return nil, fmt.Errorf("%w: invalid persisted Source", ErrPersistence)
		}
	}
	result := cloneSources(items)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority < result[j].Priority
		}
		if result[i].Code != result[j].Code {
			return result[i].Code < result[j].Code
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (s *UseCase) Update(ctx context.Context, id string, input MutableSource) (Source, error) {
	if !coreid.Is(id, coreid.Source) {
		return Source{}, &ValidationError{Field: "source_id", Message: "must be a stable Source ID"}
	}
	var updated Source
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		items, err := tx.List(ctx)
		if err != nil {
			return err
		}
		index := sourceIndex(items, id)
		if index < 0 {
			return ErrNotFound
		}
		candidate := sourceFromMutable(input)
		candidate.ID = items[index].ID
		candidate.Code = items[index].Code
		candidate.OwnershipType = items[index].OwnershipType
		candidate.ChannelType = items[index].ChannelType
		candidate.CreatedAt = items[index].CreatedAt
		candidate.UpdatedAt = items[index].UpdatedAt
		if err := validateSource(candidate, false); err != nil {
			return err
		}
		items[index] = candidate
		if err := validateProjectedSet(items); err != nil {
			return err
		}
		updated, err = tx.Update(ctx, candidate)
		return err
	})
	if err != nil {
		return Source{}, err
	}
	return cloneSource(updated), nil
}

func (s *UseCase) Delete(ctx context.Context, id string) error {
	if !coreid.Is(id, coreid.Source) {
		return &ValidationError{Field: "source_id", Message: "must be a stable Source ID"}
	}
	return s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		items, err := tx.List(ctx)
		if err != nil {
			return err
		}
		index := sourceIndex(items, id)
		if index < 0 {
			return ErrNotFound
		}
		if items[index].OwnershipType == OwnershipFixed {
			return ErrFixedDeleteForbidden
		}
		return tx.Delete(ctx, id)
	})
}

func (s *UseCase) Import(ctx context.Context, input []Source) ([]Source, error) {
	prepared := make([]Source, len(input))
	seen := make(map[string]struct{}, len(input))
	for index, item := range input {
		item, err := prepareSource(item, index, seen, true)
		if err != nil {
			return nil, err
		}
		prepared[index] = item
	}
	if err := validateProjectedSet(prepared); err != nil {
		return nil, err
	}
	var result []Source
	err := s.store.InTransaction(ctx, func(tx Transaction) error {
		if err := tx.Lock(ctx); err != nil {
			return err
		}
		existing, err := tx.List(ctx)
		if err != nil {
			return err
		}
		if len(existing) == 0 {
			result = make([]Source, 0, len(prepared))
			for _, item := range prepared {
				created, err := tx.Insert(ctx, item)
				if err != nil {
					return err
				}
				result = append(result, cloneSource(created))
			}
			return nil
		}
		if len(existing) != len(prepared) {
			return ErrConflict
		}
		byCode := make(map[string]Source, len(existing))
		for _, item := range existing {
			byCode[item.Code] = item
		}
		result = make([]Source, 0, len(prepared))
		for _, item := range prepared {
			current, ok := byCode[item.Code]
			if !ok || !sameSource(current, item) {
				return ErrConflict
			}
			result = append(result, cloneSource(current))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return cloneSources(result), nil
}

func prepareSource(item Source, index int, seen map[string]struct{}, requireTimestamps bool) (Source, error) {
	id, err := coreid.Derive(coreid.Source, "source", item.Code)
	if err != nil {
		return Source{}, &ValidationError{Field: fmt.Sprintf("sources[%d].code", index), Message: "must be a stable Source code"}
	}
	item.ID = id
	item.Config = compactConfig(item.Config)
	if _, duplicate := seen[item.Code]; duplicate {
		return Source{}, ErrConflict
	}
	seen[item.Code] = struct{}{}
	if err := validateSource(item, requireTimestamps); err != nil {
		return Source{}, err
	}
	return cloneSource(item), nil
}

func (s *UseCase) ActiveSnapshot(ctx context.Context) ([]Source, error) {
	items, err := s.store.List(ctx, true)
	if err != nil {
		return nil, err
	}
	active := make([]Source, 0, len(items))
	for _, item := range items {
		if err := validateSource(item, true); err != nil {
			return nil, fmt.Errorf("%w: invalid persisted Source", ErrPersistence)
		}
		if item.Enabled {
			active = append(active, cloneSource(item))
		}
	}
	sortSnapshot(active)
	if len(active) > MaxSources || snapshotEnvelopeSize(active) > MaxSnapshotEnvelopeSize {
		return nil, ErrCapacityExceeded
	}
	return active, nil
}

func sourceFromMutable(input MutableSource) Source {
	config := compactConfig(input.Config)
	return Source{
		Code: input.Code, Name: input.Name, AdapterKey: input.AdapterKey, Enabled: input.Enabled,
		Endpoint: input.Endpoint, AppKey: cloneString(input.AppKey), Config: config,
		Priority: input.Priority, TimeoutSeconds: input.TimeoutSeconds, MaxResults: input.MaxResults,
		DefaultSourceLevel: input.DefaultSourceLevel,
	}
}

func validateProjectedSet(items []Source) error {
	if len(items) > MaxSources {
		return ErrCapacityExceeded
	}
	active := make([]Source, 0, len(items))
	activeWeb := 0
	for _, item := range items {
		if err := validateSource(item, false); err != nil {
			return err
		}
		if item.Enabled {
			active = append(active, item)
			if item.ChannelType == ChannelWebSearch {
				activeWeb++
			}
		}
	}
	if len(active) > MaxSources || activeWeb > 1 || snapshotEnvelopeSize(active) > MaxSnapshotEnvelopeSize {
		return ErrCapacityExceeded
	}
	return nil
}

func validateSource(item Source, requireTimestamps bool) error {
	if !coreid.Is(item.ID, coreid.Source) {
		return &ValidationError{Field: "id", Message: "must be a stable Source ID"}
	}
	if !codePattern.MatchString(item.Code) {
		return &ValidationError{Field: "code", Message: "must match the Source code format"}
	}
	if strings.TrimSpace(item.Name) == "" || utf8.RuneCountInString(item.Name) > 100 {
		return &ValidationError{Field: "name", Message: "must be nonblank and contain at most 100 characters"}
	}
	if item.OwnershipType != OwnershipFixed && item.OwnershipType != OwnershipDynamic {
		return &ValidationError{Field: "ownership_type", Message: "is invalid"}
	}
	if item.ChannelType != ChannelWebSearch && item.ChannelType != ChannelAPI && item.ChannelType != ChannelRSS {
		return &ValidationError{Field: "channel_type", Message: "is invalid"}
	}
	if !validAdapter(item.AdapterKey) {
		return &ValidationError{Field: "adapter_key", Message: "is invalid"}
	}
	if item.OwnershipType == OwnershipDynamic && (item.ChannelType != ChannelRSS || item.AdapterKey != AdapterGenericRSS) {
		return &ValidationError{Field: "adapter_key", Message: "dynamic Sources must use generic_rss"}
	}
	if len(item.Endpoint) > 2048 {
		return &ValidationError{Field: "endpoint", Message: "must contain at most 2048 characters"}
	}
	parsed, err := url.Parse(item.Endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return &ValidationError{Field: "endpoint", Message: "must be an absolute HTTP(S) URL"}
	}
	if item.AppKey != nil && (strings.TrimSpace(*item.AppKey) == "" || utf8.RuneCountInString(*item.AppKey) > 512) {
		return &ValidationError{Field: "app_key", Message: "must be nonblank and contain at most 512 characters when present"}
	}
	if err := validateConfig(item.Config); err != nil {
		return err
	}
	if item.Priority < 1 || item.Priority > 5 {
		return &ValidationError{Field: "priority", Message: "must be between 1 and 5"}
	}
	if item.TimeoutSeconds < 1 || item.TimeoutSeconds > 300 {
		return &ValidationError{Field: "timeout_seconds", Message: "must be between 1 and 300"}
	}
	if item.MaxResults < 1 || item.MaxResults > 100 {
		return &ValidationError{Field: "max_results", Message: "must be between 1 and 100"}
	}
	if !validSourceLevel(item.DefaultSourceLevel) {
		return &ValidationError{Field: "default_source_level", Message: "is invalid"}
	}
	if requireTimestamps && (item.CreatedAt.IsZero() || item.UpdatedAt.IsZero()) {
		return &ValidationError{Field: "created_at", Message: "persisted Source timestamps are required"}
	}
	if requireTimestamps && item.UpdatedAt.Before(item.CreatedAt) {
		return &ValidationError{Field: "updated_at", Message: "must not precede created_at"}
	}
	return nil
}

func validateConfig(raw json.RawMessage) error {
	compact := compactConfig(raw)
	if len(compact) > MaxConfigBytes {
		return &ValidationError{Field: "config", Message: "compact JSON must contain at most 4096 bytes"}
	}
	var value map[string]json.RawMessage
	if len(compact) == 0 || json.Unmarshal(compact, &value) != nil || value == nil {
		return &ValidationError{Field: "config", Message: "must be a JSON object"}
	}
	if levelsRaw, ok := value["source_levels"]; ok {
		var levels map[string]SourceLevel
		if json.Unmarshal(levelsRaw, &levels) != nil {
			return &ValidationError{Field: "config.source_levels", Message: "must be an object"}
		}
		for host, level := range levels {
			if strings.TrimSpace(host) == "" || !validSourceLevel(level) {
				return &ValidationError{Field: "config.source_levels", Message: "contains an invalid host or Source level"}
			}
		}
	}
	if maxRaw, ok := value["max_bytes"]; ok {
		var maximum int
		if json.Unmarshal(maxRaw, &maximum) != nil || maximum < 65_536 || maximum > 10_485_760 {
			return &ValidationError{Field: "config.max_bytes", Message: "must be between 65536 and 10485760"}
		}
	}
	return nil
}

func compactConfig(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var output bytes.Buffer
	if json.Compact(&output, raw) != nil {
		return append(json.RawMessage(nil), raw...)
	}
	return append(json.RawMessage(nil), output.Bytes()...)
}

func snapshotEnvelopeSize(items []Source) int {
	measured := cloneSources(items)
	worstTimestamp := time.Date(9999, 12, 31, 23, 59, 59, 999_999_999, time.UTC)
	for index := range measured {
		if measured[index].CreatedAt.IsZero() {
			measured[index].CreatedAt = worstTimestamp
		}
		if measured[index].UpdatedAt.IsZero() {
			measured[index].UpdatedAt = worstTimestamp
		}
	}
	value := struct {
		RequestID string `json:"request_id"`
		Result    struct {
			Sources []Source `json:"sources"`
		} `json:"result"`
	}{RequestID: strings.Repeat("💧", 128)}
	value.Result.Sources = measured
	encoded, err := json.Marshal(value)
	if err != nil {
		return MaxSnapshotEnvelopeSize + 1
	}
	return len(encoded) + 1
}

func sortSnapshot(items []Source) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ChannelType != items[j].ChannelType {
			return items[i].ChannelType < items[j].ChannelType
		}
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		if items[i].Code != items[j].Code {
			return items[i].Code < items[j].Code
		}
		return items[i].ID < items[j].ID
	})
}

func sourceIndex(items []Source, id string) int {
	for index := range items {
		if items[index].ID == id {
			return index
		}
	}
	return -1
}

func validAdapter(value AdapterKey) bool {
	switch value {
	case AdapterBocha, AdapterTavily, AdapterParallel, AdapterCLS, AdapterEastmoneyFast,
		AdapterEastmoneyStock, AdapterSTCN, AdapterGenericRSS:
		return true
	default:
		return false
	}
}

func validSourceLevel(value SourceLevel) bool {
	switch value {
	case SourceLevelOfficial, SourceLevelWire, SourceLevelMedia, SourceLevelSocial:
		return true
	default:
		return false
	}
}

func cloneSource(input Source) Source {
	input.AppKey = cloneString(input.AppKey)
	input.Config = append(json.RawMessage(nil), input.Config...)
	return input
}

func cloneSources(input []Source) []Source {
	result := make([]Source, len(input))
	for index := range input {
		result[index] = cloneSource(input[index])
	}
	return result
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func sameSource(left, right Source) bool {
	return left.ID == right.ID && left.Code == right.Code && left.Name == right.Name &&
		left.OwnershipType == right.OwnershipType && left.ChannelType == right.ChannelType &&
		left.AdapterKey == right.AdapterKey && left.Enabled == right.Enabled && left.Endpoint == right.Endpoint &&
		sameOptionalString(left.AppKey, right.AppKey) && sameConfig(left.Config, right.Config) &&
		left.Priority == right.Priority && left.TimeoutSeconds == right.TimeoutSeconds &&
		left.MaxResults == right.MaxResults && left.DefaultSourceLevel == right.DefaultSourceLevel &&
		samePostgresTimestamp(left.CreatedAt, right.CreatedAt) && samePostgresTimestamp(left.UpdatedAt, right.UpdatedAt)
}

func sameConfig(left, right json.RawMessage) bool {
	var leftValue, rightValue any
	if json.Unmarshal(left, &leftValue) != nil || json.Unmarshal(right, &rightValue) != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func samePostgresTimestamp(left, right time.Time) bool {
	return left.UTC().Round(time.Microsecond).Equal(right.UTC().Round(time.Microsecond))
}

func sameOptionalString(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
