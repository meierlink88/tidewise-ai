package source

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	sourceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/source"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

type UseCase interface {
	CreateDynamic(context.Context, sourcebiz.MutableSource) (sourcebiz.Source, error)
	List(context.Context) ([]sourcebiz.Source, error)
	Update(context.Context, string, sourcebiz.MutableSource) (sourcebiz.Source, error)
	Delete(context.Context, string) error
	ActiveSnapshot(context.Context) ([]sourcebiz.Source, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Source use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) Create(ctx context.Context, request *sourceapi.CreateRequest) (*v1.Response[sourceapi.Source], error) {
	result, err := s.useCase.CreateDynamic(ctx, mutableFromCreate(request))
	return sourceResponse(result, err, v1.StatusCreated)
}

func (s *Service) List(ctx context.Context) (*v1.Response[sourceapi.SourceList], error) {
	result, err := s.useCase.List(ctx)
	if err != nil {
		return nil, sourceError(err, false)
	}
	return &v1.Response[sourceapi.SourceList]{Status: v1.StatusOK, Result: sourceapi.SourceList{Sources: sourceDTOs(result)}}, nil
}

func (s *Service) Update(ctx context.Context, request *sourceapi.UpdateRequest) (*v1.Response[sourceapi.Source], error) {
	result, err := s.useCase.Update(ctx, request.SourceID, mutableFromUpdate(request))
	return sourceResponse(result, err, v1.StatusOK)
}

func (s *Service) Delete(ctx context.Context, request *sourceapi.DeleteRequest) (*v1.Response[sourceapi.DeleteResult], error) {
	if err := s.useCase.Delete(ctx, request.SourceID); err != nil {
		return nil, sourceError(err, false)
	}
	return &v1.Response[sourceapi.DeleteResult]{Status: v1.StatusOK, Result: sourceapi.DeleteResult{ID: request.SourceID}}, nil
}

func (s *Service) Snapshot(ctx context.Context) (*v1.Response[sourceapi.SourceSnapshot], error) {
	result, err := s.useCase.ActiveSnapshot(ctx)
	if err != nil {
		return nil, sourceError(err, true)
	}
	return &v1.Response[sourceapi.SourceSnapshot]{Status: v1.StatusOK, Result: sourceapi.SourceSnapshot{Sources: sourceDTOs(result)}}, nil
}

func mutableFromCreate(request *sourceapi.CreateRequest) sourcebiz.MutableSource {
	return sourcebiz.MutableSource{
		Code: request.Code, Name: request.Name, Enabled: request.Enabled, Endpoint: request.Endpoint,
		AppKey: request.AppKey, Config: append(json.RawMessage(nil), request.Config...), Priority: request.Priority,
		TimeoutSeconds: request.TimeoutSeconds, MaxResults: request.MaxResults,
		DefaultSourceLevel: sourcebiz.SourceLevel(request.DefaultSourceLevel),
	}
}

func mutableFromUpdate(request *sourceapi.UpdateRequest) sourcebiz.MutableSource {
	return sourcebiz.MutableSource{
		Name: request.Name, AdapterKey: sourcebiz.AdapterKey(request.AdapterKey), Enabled: request.Enabled,
		Endpoint: request.Endpoint, AppKey: request.AppKey, Config: append(json.RawMessage(nil), request.Config...),
		Priority: request.Priority, TimeoutSeconds: request.TimeoutSeconds, MaxResults: request.MaxResults,
		DefaultSourceLevel: sourcebiz.SourceLevel(request.DefaultSourceLevel),
	}
}

func sourceResponse(result sourcebiz.Source, err error, status int) (*v1.Response[sourceapi.Source], error) {
	if err != nil {
		return nil, sourceError(err, false)
	}
	return &v1.Response[sourceapi.Source]{Status: status, Result: sourceDTO(result)}, nil
}

func sourceError(err error, snapshot bool) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, sourceapi.ErrorTimeout, "Source operation exceeded its execution budget", nil)
	}
	if snapshot && (errors.Is(err, sourcebiz.ErrCapacityExceeded) || errors.Is(err, sourcebiz.ErrPersistence)) {
		return v1.NewPublicError(v1.StatusServiceUnavailable, sourceapi.ErrorSnapshotFailed, "Complete Source snapshot is unavailable", nil)
	}
	var validation *sourcebiz.ValidationError
	if errors.As(err, &validation) {
		return v1.NewPublicError(v1.StatusUnprocessableEntity, sourceapi.ErrorInvalid, "Source data is invalid", map[string]any{"field": validation.Field, "message": validation.Message})
	}
	if errors.Is(err, sourcebiz.ErrNotFound) {
		return v1.NewPublicError(v1.StatusNotFound, sourceapi.ErrorNotFound, "Source was not found", nil)
	}
	if errors.Is(err, sourcebiz.ErrFixedDeleteForbidden) {
		return v1.NewPublicError(v1.StatusConflict, sourceapi.ErrorFixedDeleteForbidden, "Fixed Source cannot be deleted", nil)
	}
	if errors.Is(err, sourcebiz.ErrCapacityExceeded) {
		return v1.NewPublicError(v1.StatusConflict, sourceapi.ErrorCapacityExceeded, "Source capacity would be exceeded", nil)
	}
	if errors.Is(err, sourcebiz.ErrConflict) {
		return v1.NewPublicError(v1.StatusConflict, sourceapi.ErrorConflict, "Source conflicts with stored state", nil)
	}
	if errors.Is(err, sourcebiz.ErrPersistence) {
		return v1.NewPublicError(v1.StatusInternalServerError, sourceapi.ErrorFailed, "Source operation failed", nil)
	}
	return v1.NewPublicError(v1.StatusInternalServerError, sourceapi.ErrorFailed, "Source operation failed", nil)
}

func sourceDTOs(input []sourcebiz.Source) []sourceapi.Source {
	result := make([]sourceapi.Source, len(input))
	for index := range input {
		result[index] = sourceDTO(input[index])
	}
	return result
}

func sourceDTO(input sourcebiz.Source) sourceapi.Source {
	return sourceapi.Source{
		ID: input.ID, Code: input.Code, Name: input.Name, OwnershipType: string(input.OwnershipType),
		ChannelType: string(input.ChannelType), AdapterKey: string(input.AdapterKey), Enabled: input.Enabled,
		Endpoint: input.Endpoint, AppKey: cloneString(input.AppKey), Config: append(json.RawMessage(nil), input.Config...),
		Priority: input.Priority, TimeoutSeconds: input.TimeoutSeconds, MaxResults: input.MaxResults,
		DefaultSourceLevel: string(input.DefaultSourceLevel), CreatedAt: input.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: input.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ sourceapi.Service = (*Service)(nil)
