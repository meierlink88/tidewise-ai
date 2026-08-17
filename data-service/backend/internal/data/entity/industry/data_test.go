package industry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	industrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industry"
)

func TestGetRejectsPersistedIndustryOutsideBizVocabulary(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	mock.ExpectQuery(`SELECT`).WithArgs(industrybiz.ID("ENT11111111-1111-4111-8111-111111111111")).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "name", "aliases", "classification_system", "industry_code", "parent_industry_id",
			"hierarchy_path_codes", "definition", "review_status", "created_at", "updated_at", "parent_contract_valid",
		}).AddRow(
			"ENT11111111-1111-4111-8111-111111111111", "半导体", []byte(`[]`), "TIDEWISE", "SEMICONDUCTOR", nil,
			[]byte(`["SEMICONDUCTOR"]`), "半导体行业", "reviewed", now, now, true,
		),
	)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Get(context.Background(), industrybiz.ID("ENT11111111-1111-4111-8111-111111111111"))
	if !errors.Is(err, industrybiz.ErrPersistence) {
		t.Fatalf("Get() error = %v, want ErrPersistence", err)
	}
}

func TestGetRejectsPersistedIndustryWithBrokenParentContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Now().UTC()
	mock.ExpectQuery(`SELECT`).WithArgs(industrybiz.ID("ENT11111111-1111-4111-8111-111111111111")).WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "name", "aliases", "classification_system", "industry_code", "parent_industry_id",
			"hierarchy_path_codes", "definition", "review_status", "created_at", "updated_at", "parent_contract_valid",
		}).AddRow(
			"ENT11111111-1111-4111-8111-111111111111", "集成电路", []byte(`[]`), "TIDEWISE", "IC",
			"ENT22222222-2222-4222-8222-222222222222", []byte(`["SEMICONDUCTOR","IC"]`),
			"集成电路行业", "candidate", now, now, false,
		),
	)
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Get(context.Background(), industrybiz.ID("ENT11111111-1111-4111-8111-111111111111"))
	if !errors.Is(err, industrybiz.ErrPersistence) {
		t.Fatalf("Get() error = %v, want ErrPersistence", err)
	}
}
