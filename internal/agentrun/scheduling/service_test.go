package scheduling_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/persistence/postgres"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/scheduling"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/testsupport"
	"github.com/jonboulle/clockwork"
)

type recordingAgentRunner struct {
	triggers chan scheduling.Trigger
}

func (r *recordingAgentRunner) ValidateInput(context.Context, string, json.RawMessage) error {
	return nil
}

func (r *recordingAgentRunner) ConfigurationReady(context.Context, string) error {
	return nil
}

func (r *recordingAgentRunner) Trigger(_ context.Context, trigger scheduling.Trigger) error {
	r.triggers <- trigger
	return nil
}

func TestEnabledScheduleTriggersRegisteredAgentAtCronTime(t *testing.T) {
	databaseURL := os.Getenv("AGENTRUN_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AGENTRUN_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	isolatedURL, cleanup, err := testsupport.IsolatedPostgresDatabase(ctx, databaseURL, "scheduling_cron_test")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	database, err := postgres.Open(ctx, isolatedURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := postgres.Migrate(ctx, database); err != nil {
		t.Fatal(err)
	}
	store := postgres.New(database)
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 7, 24, 9, 59, 0, 0, location)
	schedule, err := store.PutAgentSchedule(ctx, agentrun.PutAgentScheduleInput{
		AgentKey:       "collector",
		AgentVersion:   "collector.v1",
		Type:           agentrun.ScheduleCron,
		CronExpression: "0 10 * * *",
		InputPayload:   json.RawMessage(`{"prompt":"采集上午资讯"}`),
		Enabled:        true,
		UpdatedAt:      start,
	})
	if err != nil {
		t.Fatal(err)
	}
	runner := &recordingAgentRunner{triggers: make(chan scheduling.Trigger, 1)}
	clock := clockwork.NewFakeClockAt(start)
	service, err := scheduling.New(
		store,
		location,
		map[string]scheduling.AgentRunner{"collector": runner},
		scheduling.WithClock(clock),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Shutdown() })
	if err := service.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := clock.BlockUntilContext(ctx, 1); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Minute)

	select {
	case trigger := <-runner.triggers:
		if trigger.ScheduleID != schedule.ID || trigger.AgentKey != "collector" ||
			trigger.AgentVersion != "collector.v1" ||
			string(trigger.InputPayload) != `{"prompt": "采集上午资讯"}` ||
			!trigger.TriggeredAt.Equal(start.Add(time.Minute)) {
			t.Fatalf("trigger = %#v", trigger)
		}
	case <-time.After(time.Second):
		t.Fatal("enabled Agent Schedule did not trigger")
	}

	if err := service.Shutdown(); err != nil {
		t.Fatal(err)
	}
	lateStart := start.Add(24*time.Hour + 2*time.Minute)
	lateClock := clockwork.NewFakeClockAt(lateStart)
	lateRunner := &recordingAgentRunner{triggers: make(chan scheduling.Trigger, 1)}
	restarted, err := scheduling.New(
		store,
		location,
		map[string]scheduling.AgentRunner{"collector": lateRunner},
		scheduling.WithClock(lateClock),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Shutdown() })
	if err := restarted.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := lateClock.BlockUntilContext(ctx, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case trigger := <-lateRunner.triggers:
		t.Fatalf("restart backfilled a missed Schedule trigger: %#v", trigger)
	case <-time.After(50 * time.Millisecond):
	}
	lateClock.Advance(23*time.Hour + 59*time.Minute)
	select {
	case trigger := <-lateRunner.triggers:
		if !trigger.TriggeredAt.Equal(lateStart.Add(23*time.Hour + 59*time.Minute)) {
			t.Fatalf("next scheduled trigger = %#v", trigger)
		}
	case <-time.After(time.Second):
		t.Fatal("restarted Schedule did not wait for its next normal tick")
	}
}
