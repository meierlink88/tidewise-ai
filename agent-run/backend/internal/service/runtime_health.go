package service

import (
	"context"
	"net/http"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/agent-run/backend/api/agentrun/v1"
)

func (s *AgentRunService) GetRuntimeHealth(ctx context.Context, _ *v1.RuntimeHealthRequest) (*v1.RuntimeHealth, error) {
	if s == nil || s.runtimeHealth == nil {
		return nil, v1.NewPublicError(http.StatusInternalServerError, "AGENTRUN_NOT_READY", "Runtime health is unavailable", nil)
	}
	result := s.runtimeHealth.Get(ctx)
	services := make([]v1.RuntimeHealthService, 0, len(result.Services))
	for _, item := range result.Services {
		var latency *int64
		if item.Latency > 0 {
			milliseconds := item.Latency.Milliseconds()
			latency = &milliseconds
		}
		services = append(services, v1.RuntimeHealthService{
			Key: item.Key, DisplayName: item.DisplayName, Status: string(item.Status),
			CheckedAt: item.CheckedAt.UTC().Format(time.RFC3339Nano), LatencyMS: latency,
			ReasonCode: string(item.ReasonCode),
		})
	}
	return &v1.RuntimeHealth{CheckedAt: result.CheckedAt.UTC().Format(time.RFC3339Nano), Services: services}, nil
}
