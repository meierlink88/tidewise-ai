package runtimehealth

import (
	"context"
	"errors"
	"net/http"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	runtimehealthapi "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1/runtimehealth"
	runtimehealthbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
)

type Provider interface {
	Get(context.Context) runtimehealthbiz.Result
}

type Service struct{ provider Provider }

func NewService(provider Provider) (*Service, error) {
	if provider == nil {
		return nil, errors.New("Runtime Health provider is required")
	}
	return &Service{provider: provider}, nil
}

func (s *Service) GetRuntimeHealth(ctx context.Context, _ *runtimehealthapi.Request) (*v1.Response[runtimehealthapi.Result], error) {
	if s == nil || s.provider == nil {
		return nil, v1.NewPublicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "runtime health service is unavailable", nil)
	}
	result := s.provider.Get(ctx)
	services := make([]runtimehealthapi.ServiceStatus, 0, len(result.Services))
	for _, item := range result.Services {
		var latency *int64
		if item.Latency > 0 {
			milliseconds := item.Latency.Milliseconds()
			latency = &milliseconds
		}
		services = append(services, runtimehealthapi.ServiceStatus{
			Key: item.Key, DisplayName: item.DisplayName, Status: string(item.Status),
			CheckedAt: item.CheckedAt.UTC().Format(time.RFC3339Nano), LatencyMS: latency,
			ReasonCode: string(item.ReasonCode),
		})
	}
	return &v1.Response[runtimehealthapi.Result]{Status: http.StatusOK, Result: runtimehealthapi.Result{
		CheckedAt: result.CheckedAt.UTC().Format(time.RFC3339Nano), Services: services,
	}}, nil
}

var _ runtimehealthapi.Service = (*Service)(nil)
