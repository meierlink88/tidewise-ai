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
	ReasonCollectionUnhealthy  ReasonCode = "collection_unhealthy"
	ReasonAuthenticationFailed ReasonCode = "authentication_failed"
	ReasonInvalidResponse      ReasonCode = "invalid_response"
)

type Check struct {
	Status     Status
	ReasonCode ReasonCode
	Latency    time.Duration
}

type Readiness interface {
	Ready(context.Context) error
}

type Probe interface {
	Check(context.Context) Check
}

type ServiceStatus struct {
	Key, DisplayName string
	Status           Status
	CheckedAt        time.Time
	Latency          time.Duration
	ReasonCode       ReasonCode
}

type Result struct {
	CheckedAt time.Time
	Services  []ServiceStatus
}

type Service struct {
	readiness Readiness
	qdrant    Probe
	now       func() time.Time
}

func New(readiness Readiness, qdrant Probe, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{readiness: readiness, qdrant: qdrant, now: now}
}

func (s *Service) Get(ctx context.Context) Result {
	checkedAt := s.now().UTC()
	agentrun := Check{Status: StatusDegraded, ReasonCode: ReasonNotReady}
	if s.readiness != nil && s.readiness.Ready(ctx) == nil {
		agentrun = Check{Status: StatusReady}
	}
	qdrant := Check{Status: StatusUnknown, ReasonCode: ReasonNotReady}
	if s.qdrant != nil {
		qdrant = s.qdrant.Check(ctx)
		if !valid(qdrant) {
			qdrant = Check{Status: StatusUnknown, ReasonCode: ReasonInvalidResponse}
		}
	}
	return Result{CheckedAt: checkedAt, Services: []ServiceStatus{
		{Key: "agentrun", DisplayName: "AgentRun", Status: agentrun.Status, CheckedAt: checkedAt, ReasonCode: agentrun.ReasonCode},
		{Key: "qdrant", DisplayName: "Qdrant", Status: qdrant.Status, CheckedAt: checkedAt, Latency: qdrant.Latency, ReasonCode: qdrant.ReasonCode},
	}}
}

func valid(check Check) bool {
	switch check.Status {
	case StatusReady:
		return check.ReasonCode == ReasonNone
	case StatusDegraded, StatusDown, StatusUnknown:
		return check.ReasonCode != ReasonNone
	default:
		return false
	}
}
