// Package biz contains Admin Portal business orchestration and domain ports.
package biz

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

var ErrDataServiceUnavailable = errors.New("data service unavailable")
var ErrRawEvidenceNotFound = errors.New("raw evidence not found")

type Service struct {
	dataClient               DataServiceRepo
	dataHealth               RuntimeHealthProvider
	rawEvidencePublicBaseURL string
	now                      func() time.Time
}

func WithRawEvidencePublicBaseURL(baseURL string) Option {
	return func(service *Service) {
		service.rawEvidencePublicBaseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	}
}

type Option func(*Service)

func WithRuntimeHealthProvider(dataProvider RuntimeHealthProvider) Option {
	return func(service *Service) {
		service.dataHealth = dataProvider
	}
}

func NewService(dataClient DataServiceRepo, options ...Option) *Service {
	service := &Service{dataClient: dataClient, now: time.Now}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (s *Service) ListEvents(ctx context.Context, query EventListQuery) (EventPage, error) {
	if s == nil || s.dataClient == nil {
		return EventPage{}, ErrDataServiceUnavailable
	}
	return s.dataClient.ListEvents(ctx, query)
}

func (s *Service) ListEvidences(ctx context.Context, query EvidenceListQuery) (EvidencePage, error) {
	if s == nil || s.dataClient == nil {
		return EvidencePage{}, ErrDataServiceUnavailable
	}
	return s.dataClient.ListEvidences(ctx, query)
}

func (s *Service) GetCollectionDocument(ctx context.Context, rawEvidenceID string) (CollectionDocument, error) {
	if s == nil || s.dataClient == nil || s.rawEvidencePublicBaseURL == "" {
		return CollectionDocument{}, ErrDataServiceUnavailable
	}
	document, err := s.dataClient.GetRawEvidenceDocument(ctx, rawEvidenceID)
	if err != nil {
		return CollectionDocument{}, err
	}
	path := strings.TrimSpace(document.RawText)
	matches := collectionDocumentPathPattern.FindStringSubmatch(path)
	if len(matches) != 2 {
		return CollectionDocument{}, nil
	}
	if _, err := time.Parse("2006/01/02", matches[1]); err != nil {
		return CollectionDocument{}, nil
	}
	return CollectionDocument{Available: true, URL: s.rawEvidencePublicBaseURL + path}, nil
}

var collectionDocumentPathPattern = regexp.MustCompile(`^/raw-evidence/documents/([0-9]{4}/[0-9]{2}/[0-9]{2})/[0-9a-f]{64}\.md$`)

func (s *Service) ListEvidenceCategories(ctx context.Context) ([]EvidenceCategory, error) {
	if s == nil || s.dataClient == nil {
		return nil, ErrDataServiceUnavailable
	}
	return s.dataClient.ListEvidenceCategories(ctx)
}

func (s *Service) ListSources(ctx context.Context, query SourceListQuery) (SourcePage, error) {
	if s == nil || s.dataClient == nil {
		return SourcePage{}, ErrDataServiceUnavailable
	}
	sources, err := s.dataClient.ListSources(ctx)
	if err != nil {
		return SourcePage{}, err
	}
	items := make([]Source, 0, len(sources))
	needle := strings.ToLower(strings.TrimSpace(query.Text))
	for _, source := range sources {
		if needle != "" && !strings.Contains(strings.ToLower(source.Name), needle) && !strings.Contains(strings.ToLower(source.Code), needle) {
			continue
		}
		if query.OwnershipType != "" && source.OwnershipType != query.OwnershipType {
			continue
		}
		if query.ChannelType != "" && source.ChannelType != query.ChannelType {
			continue
		}
		if query.Enabled != nil && source.Enabled != *query.Enabled {
			continue
		}
		if query.Priority != nil && source.Priority != *query.Priority {
			continue
		}
		if query.DefaultSourceLevel != "" && source.DefaultSourceLevel != query.DefaultSourceLevel {
			continue
		}
		if query.UpdatedFrom != nil && source.UpdatedAt.Before(*query.UpdatedFrom) {
			continue
		}
		if query.UpdatedTo != nil && source.UpdatedAt.After(*query.UpdatedTo) {
			continue
		}
		items = append(items, source)
	}
	total := len(items)
	start := (query.Page - 1) * query.PageSize
	if start >= total {
		items = []Source{}
	} else {
		end := start + query.PageSize
		if end > total {
			end = total
		}
		items = items[start:end]
	}
	return SourcePage{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}
