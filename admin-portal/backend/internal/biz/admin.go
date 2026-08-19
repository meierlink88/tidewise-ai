// Package biz contains Admin Portal business orchestration and domain ports.
package biz

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrDataServiceUnavailable = errors.New("data service unavailable")

type Service struct {
	dataClient DataServiceRepo
	dataHealth RuntimeHealthProvider
	now        func() time.Time
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
