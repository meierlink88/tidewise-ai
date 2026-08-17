package chainnode

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	chainnodeapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/chainnode"
	chainnodebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/chainnode"
)

type UseCase interface {
	Create(context.Context, chainnodebiz.ChainNode) (chainnodebiz.ChainNode, error)
	List(context.Context, chainnodebiz.ListRequest) (chainnodebiz.Page, error)
	Get(context.Context, chainnodebiz.ID) (chainnodebiz.ChainNode, error)
	Update(context.Context, chainnodebiz.ID, chainnodebiz.Update) (chainnodebiz.ChainNode, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, fmt.Errorf("ChainNode use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) Create(ctx context.Context, request *chainnodeapi.CreateRequest) (*v1.Response[chainnodeapi.ChainNode], error) {
	result, err := s.useCase.Create(ctx, chainnodebiz.ChainNode{
		Name: request.Name, Aliases: request.Aliases, Definition: request.Definition,
		ReviewStatus: chainnodebiz.ReviewStatus(request.ReviewStatus),
	})
	return chainNodeResponse(result, err, v1.StatusCreated)
}

func (s *Service) List(ctx context.Context, request *chainnodeapi.ListRequest) (*v1.Response[chainnodeapi.ChainNodeList], error) {
	pageSize, err := v1.ParseBoundedInt(request.PageSize, 50, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	result, err := s.useCase.List(ctx, chainnodebiz.ListRequest{PageSize: pageSize, Cursor: request.Cursor})
	if err != nil {
		return nil, chainNodeError(err)
	}
	items := make([]chainnodeapi.ChainNode, len(result.Items))
	for index, item := range result.Items {
		items[index] = chainNodeDTO(item)
	}
	return &v1.Response[chainnodeapi.ChainNodeList]{Status: v1.StatusOK, Result: chainnodeapi.ChainNodeList{Items: items, NextCursor: result.NextCursor}}, nil
}

func (s *Service) Get(ctx context.Context, request *chainnodeapi.GetRequest) (*v1.Response[chainnodeapi.ChainNode], error) {
	result, err := s.useCase.Get(ctx, chainnodebiz.ID(request.ChainNodeID))
	return chainNodeResponse(result, err, v1.StatusOK)
}

func (s *Service) Update(ctx context.Context, request *chainnodeapi.UpdateRequest) (*v1.Response[chainnodeapi.ChainNode], error) {
	result, err := s.useCase.Update(ctx, chainnodebiz.ID(request.ChainNodeID), chainnodebiz.Update{
		Name: request.Name, Aliases: request.Aliases, Definition: request.Definition,
		ReviewStatus: chainnodebiz.ReviewStatus(request.ReviewStatus),
	})
	return chainNodeResponse(result, err, v1.StatusOK)
}

func chainNodeResponse(result chainnodebiz.ChainNode, err error, status int) (*v1.Response[chainnodeapi.ChainNode], error) {
	if err != nil {
		return nil, chainNodeError(err)
	}
	return &v1.Response[chainnodeapi.ChainNode]{Status: status, Result: chainNodeDTO(result)}, nil
}

func chainNodeError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, chainnodeapi.ErrorTimeout, "ChainNode operation exceeded its execution budget", nil)
	}
	var validation *chainnodebiz.ValidationError
	if errors.As(err, &validation) {
		if validation.Field == "cursor" {
			return v1.NewPublicError(v1.StatusBadRequest, chainnodeapi.ErrorInvalidRequest, "ChainNode list cursor is invalid", nil)
		}
		return v1.NewPublicError(v1.StatusUnprocessableEntity, chainnodeapi.ErrorInvalid, "ChainNode data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	if errors.Is(err, chainnodebiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, chainnodeapi.ErrorNotFound, "ChainNode was not found", nil)
	}
	if errors.Is(err, chainnodebiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, chainnodeapi.ErrorConflict, "ChainNode identity conflicts with stored data", nil)
	}
	if errors.Is(err, chainnodebiz.ErrPersistence) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, chainnodeapi.ErrorPersistenceFailed, "ChainNode persistence is unavailable", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, chainnodeapi.ErrorFailed, "ChainNode operation failed", nil)
}

func chainNodeDTO(input chainnodebiz.ChainNode) chainnodeapi.ChainNode {
	aliases := append([]string(nil), input.Aliases...)
	if aliases == nil {
		aliases = []string{}
	}
	return chainnodeapi.ChainNode{
		ID: string(input.ID), Name: input.Name, Aliases: aliases, Definition: input.Definition,
		ReviewStatus: string(input.ReviewStatus), CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

var _ chainnodeapi.Service = (*Service)(nil)
