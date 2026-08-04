package service

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/admin-portal/backend/api/admin/v1"
)

func (s *AdminService) GetRuntimeHealth(ctx context.Context, _ *v1.EmptyRequest) (*v1.RuntimeHealth, error) {
	if s == nil || s.admin == nil {
		return nil, v1.ErrInvalidRequest
	}
	result := s.admin.GetRuntimeHealth(ctx)
	services := make([]v1.RuntimeHealthService, 0, len(result.Services))
	for _, item := range result.Services {
		services = append(services, v1.RuntimeHealthService{
			Key: string(item.Key), DisplayName: item.DisplayName, Status: string(item.Status),
			CheckedAt: item.CheckedAt, LatencyMS: item.LatencyMS, ReasonCode: string(item.ReasonCode),
		})
	}
	return &v1.RuntimeHealth{Status: string(result.Status), CheckedAt: result.CheckedAt, Services: services}, nil
}
