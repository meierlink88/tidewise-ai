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

type Check struct {
	Status     Status
	ReasonCode ReasonCode
	Latency    time.Duration
}

type Probe interface {
	Check(context.Context) Check
}

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

type Service struct {
	neo4j Probe
	now   func() time.Time
}

func New(neo4j Probe, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{neo4j: neo4j, now: now}
}

func (s *Service) Get(ctx context.Context) Result {
	checkedAt := s.now().UTC()
	neo4j := Check{Status: StatusUnknown, ReasonCode: ReasonNotReady}
	if s.neo4j != nil {
		neo4j = s.neo4j.Check(ctx)
		if !validCheck(neo4j) {
			neo4j = Check{Status: StatusUnknown, ReasonCode: ReasonNotReady}
		}
	}
	return Result{CheckedAt: checkedAt, Services: []ServiceStatus{
		{Key: "data", DisplayName: "Data Service", Status: StatusReady, CheckedAt: checkedAt},
		{Key: "neo4j", DisplayName: "Neo4j", Status: neo4j.Status, CheckedAt: checkedAt, Latency: neo4j.Latency, ReasonCode: neo4j.ReasonCode},
	}}
}

func validCheck(check Check) bool {
	switch check.Status {
	case StatusReady:
		return check.ReasonCode == ReasonNone
	case StatusDegraded, StatusDown, StatusUnknown:
		return check.ReasonCode != ReasonNone
	default:
		return false
	}
}
