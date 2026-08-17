package concept

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	conceptbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/concept"
)

func TestGetRejectsPersistedConceptOutsideBizVocabulary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	mock.ExpectQuery(`SELECT`).WithArgs(conceptbiz.ID("CON33333333-3333-4333-8333-333333333333")).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "name", "aliases", "concept_type", "definition", "review_status", "created_at", "updated_at",
		}).AddRow(
			"CON33333333-3333-4333-8333-333333333333", "人工智能", []byte(`["AI"]`), "sector", "错误概念", "candidate", now, now,
		),
	)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Get(context.Background(), conceptbiz.ID("CON33333333-3333-4333-8333-333333333333"))
	if !errors.Is(err, conceptbiz.ErrPersistence) {
		t.Fatalf("Get() error = %v, want ErrPersistence", err)
	}
}
