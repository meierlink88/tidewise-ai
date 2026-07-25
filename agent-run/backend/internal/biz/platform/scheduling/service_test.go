package scheduling

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
)

type scheduleStoreStub struct {
	saved agentrun.AgentSchedule
}

func (s *scheduleStoreStub) GetAgentVersion(_ context.Context, version string) (agentrun.AgentVersion, error) {
	if version != "collector.v1" {
		return agentrun.AgentVersion{}, errors.New("not found")
	}
	return agentrun.AgentVersion{AgentKey: "collector", Version: version}, nil
}

func (s *scheduleStoreStub) PutAgentSchedule(_ context.Context, input agentrun.PutAgentScheduleInput) (agentrun.AgentSchedule, error) {
	s.saved = agentrun.AgentSchedule{
		ID:       "11111111-1111-4111-8111-111111111111",
		AgentKey: input.AgentKey, AgentVersion: input.AgentVersion, Type: input.Type,
		CronExpression: input.CronExpression, DailyTimes: input.DailyTimes,
		InputPayload: input.InputPayload, Enabled: input.Enabled, UpdatedAt: input.UpdatedAt,
	}
	return s.saved, nil
}

func (s *scheduleStoreStub) GetAgentSchedule(context.Context, string) (agentrun.AgentSchedule, error) {
	return s.saved, nil
}

func (*scheduleStoreStub) ListAgentSchedules(context.Context) ([]agentrun.AgentSchedule, error) {
	return nil, nil
}

func (*scheduleStoreStub) ListEnabledAgentSchedules(context.Context) ([]agentrun.AgentSchedule, error) {
	return nil, nil
}

type scheduleRuntimeStub struct {
	validated agentrun.AgentSchedule
	synced    agentrun.AgentSchedule
}

func (r *scheduleRuntimeStub) ValidateCronExpression(string) error {
	return nil
}

func (*scheduleRuntimeStub) Start(context.Context, []agentrun.AgentSchedule) error { return nil }
func (r *scheduleRuntimeStub) Sync(_ context.Context, schedule agentrun.AgentSchedule) error {
	r.synced = schedule
	return nil
}
func (*scheduleRuntimeStub) Shutdown() error { return nil }

type scheduleRunnerStub struct {
	configurationChecks int
}

func (*scheduleRunnerStub) ValidateInput(context.Context, string, json.RawMessage) error { return nil }
func (r *scheduleRunnerStub) ConfigurationReady(context.Context, string) error {
	r.configurationChecks++
	return nil
}
func (*scheduleRunnerStub) Trigger(context.Context, Trigger) error { return nil }

func TestPutOwnsScheduleNormalizationAndEnablementRules(t *testing.T) {
	store := &scheduleStoreStub{}
	runtime := &scheduleRuntimeStub{}
	runner := &scheduleRunnerStub{}
	now := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	service, err := New(
		store,
		map[string]AgentRunner{"collector": runner},
		runtime,
		WithNow(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := service.Put(context.Background(), agentrun.PutAgentScheduleInput{
		AgentKey: " collector ", AgentVersion: " collector.v1 ",
		Type: agentrun.ScheduleDaily, CronExpression: "ignored",
		DailyTimes:   []string{"10:00", " 09:00 ", "10:00"},
		InputPayload: json.RawMessage(`{"prompt":"采集资讯"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.CronExpression != "" || len(saved.DailyTimes) != 2 ||
		saved.DailyTimes[0] != "09:00" || saved.DailyTimes[1] != "10:00" ||
		!saved.UpdatedAt.Equal(now) {
		t.Fatalf("normalized Schedule = %#v", saved)
	}
	if runner.configurationChecks != 0 {
		t.Fatal("disabled Schedule checked external configuration")
	}
	enabled := true
	if _, err := service.Patch(context.Background(), "collector", agentrun.PatchAgentScheduleInput{
		Enabled: &enabled,
	}); err != nil {
		t.Fatal(err)
	}
	if runner.configurationChecks != 1 {
		t.Fatalf("enabled validation checks=%d", runner.configurationChecks)
	}
}

func TestPutRejectsNonObjectAndOversizedInputInBiz(t *testing.T) {
	service, err := New(
		&scheduleStoreStub{},
		map[string]AgentRunner{"collector": &scheduleRunnerStub{}},
		&scheduleRuntimeStub{},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []json.RawMessage{
		json.RawMessage(`[]`),
		json.RawMessage(make([]byte, 64*1024+1)),
	} {
		_, err := service.Put(context.Background(), agentrun.PutAgentScheduleInput{
			AgentKey: "collector", AgentVersion: "collector.v1",
			Type: agentrun.ScheduleCron, CronExpression: "*/5 * * * *",
			InputPayload: payload,
		})
		if !errors.Is(err, agentrun.ErrInvalidSchedule) {
			t.Fatalf("error = %v, want ErrInvalidSchedule", err)
		}
	}
}

func TestReadyTracksScheduleRuntimeLifecycle(t *testing.T) {
	service, err := New(
		&scheduleStoreStub{},
		map[string]AgentRunner{"collector": &scheduleRunnerStub{}},
		&scheduleRuntimeStub{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Ready(context.Background()); err == nil {
		t.Fatal("Schedule Service reported ready before runtime start")
	}
	if err := service.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := service.Ready(context.Background()); err != nil {
		t.Fatalf("Schedule Service readiness after start: %v", err)
	}
	if err := service.Shutdown(); err != nil {
		t.Fatal(err)
	}
	if err := service.Ready(context.Background()); err == nil {
		t.Fatal("Schedule Service remained ready after shutdown")
	}
}
