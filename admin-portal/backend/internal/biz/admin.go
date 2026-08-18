// Package biz contains Admin Portal business orchestration and domain ports.
package biz

import (
	"context"
	"errors"
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
