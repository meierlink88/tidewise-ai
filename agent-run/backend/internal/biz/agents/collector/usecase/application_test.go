package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type lifecycleLoggerStub struct {
	mu   sync.Mutex
	info []agentrun.AgentLifecycleEvent
	warn []agentrun.AgentLifecycleEvent
	err  []agentrun.AgentLifecycleEvent
}

func (l *lifecycleLoggerStub) Info(event agentrun.AgentLifecycleEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.info = append(l.info, event)
}
func (l *lifecycleLoggerStub) Warn(event agentrun.AgentLifecycleEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warn = append(l.warn, event)
}
func (l *lifecycleLoggerStub) Error(event agentrun.AgentLifecycleEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.err = append(l.err, event)
}

type reconciliationRepositoryStub struct {
	Repository
	mu                  sync.Mutex
	execution           agentrun.Execution
	failures            int
	publicationFailures int
}

func (s *reconciliationRepositoryStub) FailExecutionAndIncompleteInvocations(
	_ context.Context,
	failure agentrun.ExecutionFailure,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures++
	s.execution.Status = agentrun.StatusFailed
	s.execution.ErrorCode = failure.ErrorCode
	return nil
}

func (s *reconciliationRepositoryStub) FailPublicationReconciliation(
	_ context.Context,
	failure agentrun.ExecutionFailure,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publicationFailures++
	s.execution.Status = agentrun.StatusFailed
	s.execution.ErrorCode = failure.ErrorCode
	return nil
}

func (s *reconciliationRepositoryStub) GetExecution(
	context.Context,
	string,
) (agentrun.Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.execution, nil
}

type reconciliationArtifactStoreStub struct {
	ArtifactStore
	repository *reconciliationRepositoryStub
	calls      int
	alwaysFail bool
}

func (s *reconciliationArtifactStoreStub) ReconcilePreparedPublications(
	context.Context,
) error {
	s.calls++
	if s.alwaysFail || s.calls == 1 {
		return errors.New("publication dependency unavailable")
	}
	s.repository.mu.Lock()
	s.repository.execution.Status = agentrun.StatusSucceeded
	s.repository.execution.CandidateCounts = map[string]int{"accepted_artifacts": 1}
	s.repository.mu.Unlock()
	return nil
}

func (*reconciliationArtifactStoreStub) WriteTerminalAudit(
	agentrun.Execution,
) (map[string]string, error) {
	return nil, errors.New("terminal audit unavailable in test")
}

func TestRetryStateWriteIsBounded(t *testing.T) {
	attempts := 0
	err := retryStateWrite(func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary database failure")
		}
		return nil
	})
	if err != nil || attempts != 3 {
		t.Fatalf("eventual success err=%v attempts=%d", err, attempts)
	}

	attempts = 0
	err = retryStateWrite(func(context.Context) error {
		attempts++
		return errors.New("persistent database failure")
	})
	if err == nil || attempts != 3 {
		t.Fatalf("bounded failure err=%v attempts=%d", err, attempts)
	}
}

func TestCollectorLifecycleLogsReflectDurablePublicationAndSkipStates(t *testing.T) {
	logger := &lifecycleLoggerStub{}
	now := time.Date(2026, 7, 31, 9, 0, 1, 0, time.UTC)
	application := &Application{
		events: logger,
		now:    func() time.Time { return now },
	}
	execution := agentrun.Execution{
		ID:            "11111111-1111-4111-8111-111111111111",
		AgentKey:      collector.AgentKey,
		TriggerSource: agentrun.TriggerSchedule,
	}
	application.logPublicationPending(execution, now.Add(-time.Second))
	application.logSkipped(execution)

	if len(logger.warn) != 1 ||
		logger.warn[0].Code != "agent_execution_retry_scheduled" ||
		logger.warn[0].Status != "materializing" ||
		logger.warn[0].Outcome != "publication_reconciliation_pending" {
		t.Fatalf("publication lifecycle event = %#v", logger.warn)
	}
	if len(logger.info) != 1 ||
		logger.info[0].Code != "agent_execution_skipped" ||
		logger.info[0].Status != "skipped" {
		t.Fatalf("skip lifecycle event = %#v", logger.info)
	}
}

func TestPublicationPendingSchedulesInProcessReconciliationToTerminalState(t *testing.T) {
	execution := agentrun.Execution{
		ID:            "11111111-1111-4111-8111-111111111111",
		AgentKey:      collector.AgentKey,
		AgentVersion:  collectorAgentVersion,
		TriggerSource: agentrun.TriggerSchedule,
		Status:        agentrun.StatusMaterializing,
	}
	repository := &reconciliationRepositoryStub{execution: execution}
	artifacts := &reconciliationArtifactStoreStub{repository: repository}
	logger := &lifecycleLoggerStub{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application := &Application{
		store: repository, artifacts: artifacts, events: logger,
		now: func() time.Time {
			return time.Date(2026, 7, 31, 9, 0, 1, 0, time.UTC)
		},
		lifecycleCtx: ctx, reconcileEvery: time.Millisecond,
		reconcileLimit: 3,
	}
	if !application.schedulePublicationReconciliation(
		execution,
		time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC),
	) {
		t.Fatal("publication reconciliation was not scheduled")
	}
	done := make(chan struct{})
	go func() {
		application.active.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publication reconciliation did not complete")
	}
	if artifacts.calls < 2 {
		t.Fatalf("reconciliation calls = %d", artifacts.calls)
	}
	logger.mu.Lock()
	defer logger.mu.Unlock()
	if len(logger.info) != 1 ||
		logger.info[0].Code != "agent_execution_completed" ||
		logger.info[0].Status != string(agentrun.StatusSucceeded) ||
		logger.info[0].Outcome != "publication_reconciled" {
		t.Fatalf("reconciled lifecycle event = %#v", logger.info)
	}
}

func TestPublicationReconciliationExhaustionDurablyFailsExecution(t *testing.T) {
	execution := agentrun.Execution{
		ID:            "11111111-1111-4111-8111-111111111111",
		AgentKey:      collector.AgentKey,
		AgentVersion:  collectorAgentVersion,
		TriggerSource: agentrun.TriggerSchedule,
		Status:        agentrun.StatusMaterializing,
	}
	repository := &reconciliationRepositoryStub{execution: execution}
	artifacts := &reconciliationArtifactStoreStub{
		repository: repository,
		alwaysFail: true,
	}
	logger := &lifecycleLoggerStub{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application := &Application{
		store: repository, artifacts: artifacts, events: logger,
		now: func() time.Time {
			return time.Date(2026, 7, 31, 9, 0, 1, 0, time.UTC)
		},
		lifecycleCtx: ctx, reconcileEvery: time.Millisecond, reconcileLimit: 2,
	}
	if !application.schedulePublicationReconciliation(
		execution,
		time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC),
	) {
		t.Fatal("publication reconciliation was not scheduled")
	}
	done := make(chan struct{})
	go func() {
		application.active.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("failed publication reconciliation did not terminate")
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.failures != 0 ||
		repository.publicationFailures != 1 ||
		repository.execution.Status != agentrun.StatusFailed ||
		repository.execution.ErrorCode != "artifact_publication_reconciliation_exhausted" {
		t.Fatalf(
			"generic failures=%d publication failures=%d execution=%#v",
			repository.failures, repository.publicationFailures, repository.execution,
		)
	}
}
