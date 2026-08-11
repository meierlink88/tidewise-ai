package runtimehealth

import (
	"context"
	"time"
)

type Status string

const (
	StatusReady    Status = "ready"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
	StatusUnknown  Status = "unknown"
)

type ReasonCode string

const (
	ReasonNone                 ReasonCode = ""
	ReasonTimeout              ReasonCode = "timeout"
	ReasonUnreachable          ReasonCode = "unreachable"
	ReasonNotReady             ReasonCode = "not_ready"
	ReasonAuthenticationFailed ReasonCode = "authentication_failed"
)

type ServiceStatus struct {
	Key         string
	DisplayName string
	Status      Status
	CheckedAt   time.Time
	Latency     time.Duration
	ReasonCode  ReasonCode
}

type Result struct {
	CheckedAt time.Time
	Services  []ServiceStatus
}

type Service struct{ now func() time.Time }

func New(now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{now: now}
}

func (s *Service) Get(ctx context.Context) Result {
	checkedAt := s.now().UTC()
	_ = ctx
	return Result{CheckedAt: checkedAt, Services: []ServiceStatus{
		{Key: "data", DisplayName: "Data Service", Status: StatusReady, CheckedAt: checkedAt},
	}}
}
