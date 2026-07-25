package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresListRawDocumentsProjectsContentLevelInScannerOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	collectedAt := time.Date(2026, 7, 17, 1, 0, 0, 0, time.UTC)
	mock.ExpectQuery("(?s)SELECT COUNT\\(\\*\\).*FROM raw_documents").WithArgs("", "", "").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("(?s)SELECT id, contract_version, artifact_id, source_ref, ingest_channel, source_type, source_name, source_url,.*source_external_id, title, content_text, content_level, raw_object_uri, raw_mime_type,.*language, published_at, collected_at, content_hash, ingest_status.*FROM raw_documents").WithArgs("", "", "", 50, 0).WillReturnRows(sqlmock.NewRows([]string{
		"id", "contract_version", "artifact_id", "source_ref", "ingest_channel", "source_type", "source_name", "source_url",
		"source_external_id", "title", "content_text", "content_level", "raw_object_uri", "raw_mime_type",
		"language", "published_at", "collected_at", "content_hash", "ingest_status",
	}).AddRow(
		"11111111-1111-5111-8111-111111111111", 2, "artifact-1", "source:reuters:world", "", "news", "Example", "https://example.test/feed",
		"story-1", "Title", "Body", "full", "s3://raw/story-1", "text/html",
		"en", nil, collectedAt, "hash", "collected",
	))

	page, err := NewPostgresRepository(db).ListRawDocuments(context.Background(), RawDocumentListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ContentLevel != "full" || page.Items[0].RawObjectURI != "s3://raw/story-1" {
		t.Fatalf("raw document projection shifted: %#v", page.Items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
