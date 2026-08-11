package rawdocument

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/rawdocument"
)

func TestStoreRejectsMalformedPersistedDocument(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(`SELECT COUNT\(\*\)`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, contract_version`).WillReturnRows(sqlmock.NewRows([]string{"id", "contract_version", "artifact_id", "source_ref", "ingest_channel", "source_type", "source_name", "source_url", "source_external_id", "title", "content_text", "content_level", "raw_object_uri", "raw_mime_type", "language", "published_at", "collected_at", "content_hash", "ingest_status"}).AddRow("", 2, "artifact", "source", "", "news", "source", "https://example.com", nil, "title", "text", "full", "", "text/plain", "zh", nil, time.Now(), "hash", "collected"))
	store, err := NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(context.Background(), biz.ListFilter{}); err == nil {
		t.Fatal("List() error = nil, want persisted validation failure")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
