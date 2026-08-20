package storyline

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	geopoliticrivalrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/geopoliticrivalry"
	macroeconomicdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/macroeconomic"
	eventdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/event"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	postgresfixture "github.com/meierlink88/tidewise-ai/data-service/backend/internal/testsupport/postgres"
)

func TestStoreCreatesAndGetsGeopoliticalStoryline(t *testing.T) {
	db := openStorylineTestDatabase(t)
	ctx := context.Background()
	rivalryStore, err := geopoliticrivalrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalry, err := rivalryStore.Create(ctx, geopoliticrivalrydata.CreateInput{
		Name: "中美科技竞争", NameEn: "US-China technology competition",
		RivalryType: geopoliticrivalrydata.RivalryTypeGeopolitical,
		Description: "围绕关键技术与供应链的长期竞争。", CoreActors: "中国、美国",
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	checkedAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	created, err := store.Create(ctx, CreateInput{
		Name:                   "先进制程设备限制升级",
		Type:                   StorylineTypeGeopolitical,
		RivalryID:              &rivalry.ID,
		Summary:                "出口管制与国产替代沿关键设备环节持续演进。",
		CurrentStage:           "技术封锁强化期",
		Confidence:             0.82,
		DataAlignmentStatus:    DataAlignmentAccumulating,
		DataAlignmentScore:     0.74,
		DataAlignmentReason:    "政策与企业数据正在累积，但产能兑现仍需观察。",
		LastAlignmentCheckedAt: checkedAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !coreid.Is(created.ID, coreid.Storyline) || created.Status != StatusEmerging ||
		created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("Create() = %+v", created)
	}
	got, err := store.Get(ctx, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %+v, %v; want %+v", got, err, created)
	}
}

func TestStoreSupportsAllAnchorsListsAndUpdatesStorylines(t *testing.T) {
	db := openStorylineTestDatabase(t)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalryStore, err := geopoliticrivalrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalry, err := rivalryStore.Create(ctx, geopoliticrivalrydata.CreateInput{
		Name: "全球技术竞争", NameEn: "Global technology competition",
		RivalryType: geopoliticrivalrydata.RivalryTypeGeopolitical,
		Description: "全球关键技术竞争。", CoreActors: "主要经济体",
	})
	if err != nil {
		t.Fatal(err)
	}
	macroStore, err := macroeconomicdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	macro, err := macroStore.Create(ctx, macroeconomicdata.CreateInput{
		Name: "全球货币政策", NameEn: "Global monetary policy",
		MacroType: macroeconomicdata.MacroTypeMonetary, Description: "全球货币政策框架。",
	})
	if err != nil {
		t.Fatal(err)
	}
	const industryChainID = "ICH11111111-1111-4111-8111-111111111111"
	if _, err := db.ExecContext(ctx, `INSERT INTO industry_chain (
    id, name, aliases, scope, target_output, end_use, observable_variables,
    geography, as_of_date, review_status
) VALUES ($1, '先进制程产业链', '{}', '先进制程设备与材料', '先进制程芯片',
    '高性能计算', ARRAY['capacity'], 'global', CURRENT_DATE, 'approved')`, industryChainID); err != nil {
		t.Fatal(err)
	}
	checkedAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	inputs := []CreateInput{
		validStorylineInput("Delta", StorylineTypeGeopolitical, checkedAt, &rivalry.ID, nil, nil),
		validStorylineInput("Alpha", StorylineTypeMacro, checkedAt, nil, &macro.ID, nil),
		validStorylineInput("Shared", StorylineTypeIndustry, checkedAt, nil, nil, stringPointer(industryChainID)),
		validStorylineInput("Shared", StorylineTypeCorporate, checkedAt, nil, nil, nil),
	}
	inputs[3].Status = StatusActive
	created := make([]Storyline, 0, len(inputs))
	for _, input := range inputs {
		item, err := store.Create(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		created = append(created, item)
	}

	all, err := store.List(ctx, Filter{})
	tiedIDs := []string{created[2].ID, created[3].ID}
	sort.Strings(tiedIDs)
	wantOrder := []string{created[1].ID, created[0].ID, tiedIDs[0], tiedIDs[1]}
	if err != nil || len(all) != len(wantOrder) {
		t.Fatalf("List() = %+v, %v", all, err)
	}
	for index, wantID := range wantOrder {
		if all[index].ID != wantID {
			t.Fatalf("List()[%d].ID = %q, want %q; full list = %+v", index, all[index].ID, wantID, all)
		}
	}
	byType, err := store.List(ctx, Filter{Type: storylineTypePointer(StorylineTypeIndustry)})
	if err != nil || len(byType) != 1 || byType[0].ID != created[2].ID {
		t.Fatalf("List(type) = %+v, %v", byType, err)
	}
	byStatus, err := store.List(ctx, Filter{Status: statusPointer(StatusActive)})
	if err != nil || len(byStatus) != 1 || byStatus[0].ID != created[3].ID {
		t.Fatalf("List(status) = %+v, %v", byStatus, err)
	}

	time.Sleep(time.Millisecond)
	updated, err := store.Update(ctx, UpdateInput{
		ID: created[0].ID, Name: "全球先进制程竞争", Type: StorylineTypeGeopolitical,
		RivalryID: &rivalry.ID, Summary: "更新后的叙事摘要。", CurrentStage: "竞争深化期",
		Status: StatusDormant, Confidence: 0.76, DataAlignmentStatus: DataAlignmentLagging,
		DataAlignmentScore: 0.48, DataAlignmentReason: "产业数据暂时落后于政策信号。",
		LastAlignmentCheckedAt: checkedAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created[0].ID || !updated.CreatedAt.Equal(created[0].CreatedAt) ||
		!updated.UpdatedAt.After(created[0].UpdatedAt) || updated.Status != StatusDormant {
		t.Fatalf("Update() = %+v; created = %+v", updated, created[0])
	}
}

func TestStoreLinksEventsAndDerivesOccurredAtBounds(t *testing.T) {
	db := openStorylineTestDatabase(t)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalryStore, err := geopoliticrivalrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalry, err := rivalryStore.Create(ctx, geopoliticrivalrydata.CreateInput{
		Name: "区域竞争", NameEn: "Regional competition",
		RivalryType: geopoliticrivalrydata.RivalryTypeGeopolitical,
		Description: "区域竞争叙事蓝图。", CoreActors: "甲方、乙方",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkedAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	storyline, err := store.Create(ctx, validStorylineInput(
		"事件演进", StorylineTypeGeopolitical, checkedAt, &rivalry.ID, nil, nil,
	))
	if err != nil {
		t.Fatal(err)
	}
	emptyStoryline, err := store.Create(ctx, validStorylineInput(
		"尚无事件", StorylineTypeGeopolitical, checkedAt, &rivalry.ID, nil, nil,
	))
	if err != nil {
		t.Fatal(err)
	}
	unknownOccurrenceStoryline, err := store.Create(ctx, validStorylineInput(
		"事件发生时间未知", StorylineTypeGeopolitical, checkedAt, &rivalry.ID, nil, nil,
	))
	if err != nil {
		t.Fatal(err)
	}

	eventStore, err := eventdata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	eventUseCase, err := eventbiz.NewUseCase(eventStore)
	if err != nil {
		t.Fatal(err)
	}
	firstAt := time.Date(2026, time.July, 1, 8, 0, 0, 0, time.UTC)
	lastAt := time.Date(2026, time.August, 15, 9, 0, 0, 0, time.UTC)
	announcedOnlyAt := time.Date(2026, time.June, 1, 7, 0, 0, 0, time.UTC)
	firstEvent := createStorylineTestEvent(t, db, eventUseCase, "first", &firstAt, nil)
	lastEvent := createStorylineTestEvent(t, db, eventUseCase, "last", &lastAt, nil)
	announcedOnlyEvent := createStorylineTestEvent(t, db, eventUseCase, "announced-only", nil, &announcedOnlyAt)

	links := make([]StorylineEventLink, 0, 3)
	for _, eventID := range []string{lastEvent, announcedOnlyEvent, firstEvent} {
		link, err := store.LinkEvent(ctx, storyline.ID, eventID)
		if err != nil {
			t.Fatal(err)
		}
		if !coreid.Is(link.ID, coreid.StorylineEventLink) || link.StorylineID != storyline.ID ||
			link.EventID != eventID || link.CreatedAt.IsZero() {
			t.Fatalf("LinkEvent() = %+v", link)
		}
		links = append(links, link)
	}
	if _, err := store.LinkEvent(ctx, storyline.ID, firstEvent); !errors.Is(err, ErrConflict) {
		t.Fatalf("LinkEvent(duplicate) error = %v, want ErrConflict", err)
	}
	if _, err := store.LinkEvent(ctx, unknownOccurrenceStoryline.ID, announcedOnlyEvent); err != nil {
		t.Fatalf("LinkEvent(same Event to another Storyline) error = %v", err)
	}
	gotLinks, err := store.ListEventLinks(ctx, storyline.ID)
	if err != nil || len(gotLinks) != 3 {
		t.Fatalf("ListEventLinks() = %+v, %v", gotLinks, err)
	}
	for index, link := range gotLinks {
		if !reflect.DeepEqual(link, links[index]) {
			t.Fatalf("ListEventLinks()[%d] = %+v, want %+v", index, link, links[index])
		}
	}
	bounds, err := store.EventOccurredAtBounds(ctx, storyline.ID)
	if err != nil || bounds.First == nil || bounds.Last == nil ||
		!bounds.First.Equal(firstAt) || !bounds.Last.Equal(lastAt) {
		t.Fatalf("EventOccurredAtBounds() = %+v, %v", bounds, err)
	}
	emptyBounds, err := store.EventOccurredAtBounds(ctx, emptyStoryline.ID)
	if err != nil || emptyBounds.First != nil || emptyBounds.Last != nil {
		t.Fatalf("EventOccurredAtBounds(empty) = %+v, %v", emptyBounds, err)
	}
	unknownOccurrenceBounds, err := store.EventOccurredAtBounds(ctx, unknownOccurrenceStoryline.ID)
	if err != nil || unknownOccurrenceBounds.First != nil || unknownOccurrenceBounds.Last != nil {
		t.Fatalf("EventOccurredAtBounds(all occurred_at null) = %+v, %v", unknownOccurrenceBounds, err)
	}
}

func TestStoreRejectsInvalidStorylineContracts(t *testing.T) {
	db := openStorylineTestDatabase(t)
	ctx := context.Background()
	if _, err := NewStore(nil); err == nil {
		t.Fatal("NewStore(nil) error = nil")
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalryStore, err := geopoliticrivalrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalry, err := rivalryStore.Create(ctx, geopoliticrivalrydata.CreateInput{
		Name: "合同测试竞争", NameEn: "Contract test competition",
		RivalryType: geopoliticrivalrydata.RivalryTypeGeopolitical,
		Description: "用于验证 Storyline 合同。", CoreActors: "甲方、乙方",
	})
	if err != nil {
		t.Fatal(err)
	}
	checkedAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	valid := validStorylineInput(
		"有效故事线", StorylineTypeGeopolitical, checkedAt, &rivalry.ID, nil, nil,
	)
	tests := []struct {
		name   string
		mutate func(*CreateInput)
	}{
		{name: "blank name", mutate: func(input *CreateInput) { input.Name = " \t" }},
		{name: "long name", mutate: func(input *CreateInput) { input.Name = strings.Repeat("界", 201) }},
		{name: "unknown type", mutate: func(input *CreateInput) { input.Type = "UNKNOWN" }},
		{name: "missing anchor", mutate: func(input *CreateInput) { input.RivalryID = nil }},
		{name: "extra anchor", mutate: func(input *CreateInput) { input.MacroEconomicID = &rivalry.ID }},
		{name: "wrong anchor identity", mutate: func(input *CreateInput) {
			wrong := "MAC11111111-1111-4111-8111-111111111111"
			input.RivalryID = &wrong
		}},
		{name: "blank summary", mutate: func(input *CreateInput) { input.Summary = "" }},
		{name: "blank stage", mutate: func(input *CreateInput) { input.CurrentStage = "" }},
		{name: "long stage", mutate: func(input *CreateInput) { input.CurrentStage = strings.Repeat("阶", 51) }},
		{name: "unknown status", mutate: func(input *CreateInput) { input.Status = "UNKNOWN" }},
		{name: "negative confidence", mutate: func(input *CreateInput) { input.Confidence = -0.01 }},
		{name: "confidence one", mutate: func(input *CreateInput) { input.Confidence = 1 }},
		{name: "nan confidence", mutate: func(input *CreateInput) { input.Confidence = math.NaN() }},
		{name: "unknown alignment", mutate: func(input *CreateInput) { input.DataAlignmentStatus = "UNKNOWN" }},
		{name: "score over one", mutate: func(input *CreateInput) { input.DataAlignmentScore = 1.01 }},
		{name: "blank alignment reason", mutate: func(input *CreateInput) { input.DataAlignmentReason = "" }},
		{name: "missing alignment time", mutate: func(input *CreateInput) { input.LastAlignmentCheckedAt = time.Time{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if _, err := store.Create(ctx, input); !errors.Is(err, ErrInvalidStoryline) {
				t.Fatalf("Create() error = %v, want ErrInvalidStoryline", err)
			}
		})
	}

	missingRivalry, err := coreid.New(coreid.GeopoliticRivalry)
	if err != nil {
		t.Fatal(err)
	}
	missingInput := valid
	missingInput.RivalryID = &missingRivalry
	if _, err := store.Create(ctx, missingInput); !errors.Is(err, ErrInvalidStoryline) {
		t.Fatalf("Create(missing anchor) error = %v, want ErrInvalidStoryline", err)
	}
	if _, err := store.Get(ctx, "not-a-storyline"); !errors.Is(err, ErrInvalidStoryline) {
		t.Fatalf("Get(invalid ID) error = %v, want ErrInvalidStoryline", err)
	}
	missingStoryline, err := coreid.New(coreid.Storyline)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, missingStoryline); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}
	invalidType := StorylineType("UNKNOWN")
	if _, err := store.List(ctx, Filter{Type: &invalidType}); !errors.Is(err, ErrInvalidStoryline) {
		t.Fatalf("List(invalid type) error = %v, want ErrInvalidStoryline", err)
	}
	invalidStatus := Status("UNKNOWN")
	if _, err := store.List(ctx, Filter{Status: &invalidStatus}); !errors.Is(err, ErrInvalidStoryline) {
		t.Fatalf("List(invalid status) error = %v, want ErrInvalidStoryline", err)
	}
	update := UpdateInput{
		ID: missingStoryline, Name: valid.Name, Type: valid.Type, RivalryID: valid.RivalryID,
		Summary: valid.Summary, CurrentStage: valid.CurrentStage, Status: StatusEmerging,
		Confidence: valid.Confidence, DataAlignmentStatus: valid.DataAlignmentStatus,
		DataAlignmentScore: valid.DataAlignmentScore, DataAlignmentReason: valid.DataAlignmentReason,
		LastAlignmentCheckedAt: valid.LastAlignmentCheckedAt,
	}
	if _, err := store.Update(ctx, update); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update(missing) error = %v, want ErrNotFound", err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := store.List(canceled, Filter{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("List(canceled) error = %v, want context.Canceled", err)
	}
}

func TestStoreRejectsInvalidEventLinkContracts(t *testing.T) {
	db := openStorylineTestDatabase(t)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	missingStoryline, err := coreid.New(coreid.Storyline)
	if err != nil {
		t.Fatal(err)
	}
	missingEvent, err := coreid.New(coreid.Event)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.LinkEvent(ctx, "invalid", missingEvent); !errors.Is(err, ErrInvalidStorylineEventLink) {
		t.Fatalf("LinkEvent(invalid Storyline) error = %v", err)
	}
	if _, err := store.LinkEvent(ctx, missingStoryline, "invalid"); !errors.Is(err, ErrInvalidStorylineEventLink) {
		t.Fatalf("LinkEvent(invalid Event) error = %v", err)
	}
	if _, err := store.LinkEvent(ctx, missingStoryline, missingEvent); !errors.Is(err, ErrInvalidStorylineEventLink) {
		t.Fatalf("LinkEvent(missing endpoints) error = %v", err)
	}
	if _, err := store.ListEventLinks(ctx, missingStoryline); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListEventLinks(missing) error = %v, want ErrNotFound", err)
	}
	if _, err := store.EventOccurredAtBounds(ctx, missingStoryline); !errors.Is(err, ErrNotFound) {
		t.Fatalf("EventOccurredAtBounds(missing) error = %v, want ErrNotFound", err)
	}
}

func TestStoreFailsClosedOnUnknownPersistedStatus(t *testing.T) {
	db := openStorylineTestDatabase(t)
	ctx := context.Background()
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalryStore, err := geopoliticrivalrydata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	rivalry, err := rivalryStore.Create(ctx, geopoliticrivalrydata.CreateInput{
		Name: "脏数据竞争", NameEn: "Dirty-data competition",
		RivalryType: geopoliticrivalrydata.RivalryTypeGeopolitical,
		Description: "用于验证读取时拒绝未知状态。", CoreActors: "甲方、乙方",
	})
	if err != nil {
		t.Fatal(err)
	}
	storyline, err := store.Create(ctx, validStorylineInput(
		"脏数据故事线", StorylineTypeGeopolitical,
		time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC), &rivalry.ID, nil, nil,
	))
	if err != nil {
		t.Fatal(err)
	}
	statements := []struct {
		query string
		args  []any
	}{
		{query: `ALTER TABLE storylines ALTER COLUMN status DROP DEFAULT`},
		{query: `ALTER TABLE storylines ALTER COLUMN status TYPE TEXT USING status::text`},
		{query: `UPDATE storylines SET status = 'UNKNOWN' WHERE id = $1`, args: []any{storyline.ID}},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Get(ctx, storyline.ID); !errors.Is(err, ErrPersistence) {
		t.Fatalf("Get(unknown persisted status) error = %v, want ErrPersistence", err)
	}
}

func validStorylineInput(
	name string,
	storylineType StorylineType,
	checkedAt time.Time,
	rivalryID, macroEconomicID, industryChainID *string,
) CreateInput {
	return CreateInput{
		Name: name, Type: storylineType, RivalryID: rivalryID, MacroEconomicID: macroEconomicID,
		IndustryChainID: industryChainID,
		Summary:         "有效叙事摘要。", CurrentStage: "发展阶段", Confidence: 0.70,
		DataAlignmentStatus: DataAlignmentAligned, DataAlignmentScore: 0.80,
		DataAlignmentReason: "事实与叙事保持一致。", LastAlignmentCheckedAt: checkedAt,
	}
}

func createStorylineTestEvent(
	t *testing.T,
	db *sql.DB,
	useCase *eventbiz.UseCase,
	key string,
	occurredAt, announcedAt *time.Time,
) string {
	t.Helper()
	evidenceStore, err := evidencedata.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	evidenceUseCase, err := evidencebiz.NewUseCase(evidenceStore)
	if err != nil {
		t.Fatal(err)
	}
	publishedAt := time.Date(2026, time.August, 20, 1, 0, 0, 0, time.UTC)
	raw, err := evidenceUseCase.PublishRawEvidence(context.Background(), evidencebiz.RawEvidence{
		PublicationKey: "storyline-" + key, SourceID: "SRC_storyline_test", SourceName: "Storyline Test",
		SourceLevel: evidencebiz.SourceLevelWire, SourceURL: "https://example.test/storyline/" + key,
		IsOriginal: true, RawText: "Storyline event " + key, PublishedAt: &publishedAt,
		CollectedAt: publishedAt.Add(time.Minute), Keywords: []string{"storyline"},
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := evidenceUseCase.PublishEvidence(context.Background(), raw.ID, []evidencebiz.Evidence{{
		Summary:  "Storyline event " + key,
		Semantic: evidencebiz.Semantic{Who: stringPointer("测试主体"), What: "发生测试事件"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	created, err := useCase.Create(context.Background(), eventbiz.CreateInput{
		Title: "Storyline event " + key, Summary: "Storyline event summary " + key,
		Semantic: eventbiz.Semantic{}, Modality: eventbiz.ModalityFact,
		OccurredAt: occurredAt, AnnouncedAt: announcedAt,
		Evidence: []eventbiz.EvidenceLinkInput{{EvidenceID: evidence.IDs[0], ContributionWeight: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Event.ID
}

func stringPointer(value string) *string { return &value }

func storylineTypePointer(value StorylineType) *StorylineType { return &value }

func statusPointer(value Status) *Status { return &value }

func openStorylineTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	migrationDir, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return postgresfixture.OpenIsolated(t, "tw_storyline", migrationDir, 0)
}
