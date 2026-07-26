package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventtagcatalog"
)

func NewEventTagCatalogRepository(db *sql.DB) eventtagcatalog.Repository {
	return eventTagCatalogRepository{db: db}
}

type eventTagCatalogRepository struct {
	db *sql.DB
}

func (r eventTagCatalogRepository) ListActive(ctx context.Context) ([]eventtagcatalog.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, tag_kind, code, name, is_active
		FROM event_tag_defs
		WHERE is_active
		ORDER BY tag_kind, code, id
	`)
	if err != nil {
		return nil, fmt.Errorf("query active Event Tag Catalog: %w", err)
	}
	defer rows.Close()
	tags := make([]eventtagcatalog.Tag, 0)
	for rows.Next() {
		var tag eventtagcatalog.Tag
		if err := rows.Scan(&tag.ID, &tag.Kind, &tag.Code, &tag.Name, &tag.Active); err != nil {
			return nil, fmt.Errorf("scan active Event Tag Catalog: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read active Event Tag Catalog: %w", err)
	}
	return tags, nil
}
