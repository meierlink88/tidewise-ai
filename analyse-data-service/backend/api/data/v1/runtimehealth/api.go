package runtimehealth

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

const OperationGet = "data.v1.getRuntimeHealth"

func BusinessOperations() []string {
	return []string{OperationGet}
}

type Service interface {
	GetRuntimeHealth(context.Context, *Request) (*v1.Response[Result], error)
}

type Request struct{}

type Result struct {
	CheckedAt string          `json:"checked_at"`
	Services  []ServiceStatus `json:"services"`
}

type ServiceStatus struct {
	Key         string `json:"key"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	CheckedAt   string `json:"checked_at"`
	LatencyMS   *int64 `json:"latency_ms,omitempty"`
	ReasonCode  string `json:"reason_code,omitempty"`
}
