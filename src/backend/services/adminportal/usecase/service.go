// Package usecase contains Admin Portal business orchestration.
package usecase

import (
	"context"
	"errors"

	"github.com/meierlink88/tidewise-ai/backend/services/adminportal/dataclient"
)

var ErrDataServiceUnavailable = errors.New("data service unavailable")

type Service struct {
	client dataclient.DataServiceClient
}

func NewService(client dataclient.DataServiceClient) *Service {
	return &Service{client: client}
}

func (s *Service) ListRawDocuments(ctx context.Context, query dataclient.RawDocumentListQuery) (dataclient.RawDocumentPage, error) {
	if s == nil || s.client == nil {
		return dataclient.RawDocumentPage{}, ErrDataServiceUnavailable
	}
	return s.client.ListRawDocuments(ctx, query)
}

func (s *Service) ListEvents(ctx context.Context, query dataclient.EventListQuery) (dataclient.EventPage, error) {
	if s == nil || s.client == nil {
		return dataclient.EventPage{}, ErrDataServiceUnavailable
	}
	return s.client.ListEvents(ctx, query)
}
