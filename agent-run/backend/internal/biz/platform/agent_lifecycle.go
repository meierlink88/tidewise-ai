package agentrun

import "time"

type AgentLifecycleEvent struct {
	Code          string
	AgentKey      string
	AgentVersion  string
	RuntimeMode   string
	ExecutionID   string
	WorkItemID    string
	TriggerSource string
	Status        string
	Outcome       string
	Stage         string
	ErrorCode     string
	Attempt       int
	MaxAttempts   int
	Duration      time.Duration
	Counts        map[string]int
}

type AgentLifecycleLogger interface {
	Info(AgentLifecycleEvent)
	Warn(AgentLifecycleEvent)
	Error(AgentLifecycleEvent)
}

type DiscardAgentLifecycleLogger struct{}

func (DiscardAgentLifecycleLogger) Info(AgentLifecycleEvent)  {}
func (DiscardAgentLifecycleLogger) Warn(AgentLifecycleEvent)  {}
func (DiscardAgentLifecycleLogger) Error(AgentLifecycleEvent) {}
