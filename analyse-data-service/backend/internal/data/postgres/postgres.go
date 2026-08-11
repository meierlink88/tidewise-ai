package postgres

import (
	"database/sql"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/adminquery"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/eventsemantics"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchreasoningtreeimport"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchthemeimport"
)

type repository struct {
	db *sql.DB
}

func newRepository(db *sql.DB) repository {
	return repository{db: db}
}

func NewEventSemanticsStore(db *sql.DB) eventsemantics.Store {
	return newRepository(db)
}

func NewResearchThemeImportStore(db *sql.DB) researchthemeimport.Store {
	return newRepository(db)
}

func NewResearchReasoningTreeImportStore(db *sql.DB) researchreasoningtreeimport.Store {
	return newRepository(db)
}

func NewResearchPublicationStore(db *sql.DB) researchpublication.Store {
	return newRepository(db)
}

func NewIndustryRelationshipImportStore(db *sql.DB) industryrelationshipimport.Store {
	return newRepository(db)
}

func NewResearchRepository(db *sql.DB) research.Repository {
	return newRepository(db)
}

func NewAdminQueryRepository(db *sql.DB) adminquery.Repository {
	return newRepository(db)
}

func NewBenchmarkObservationRepository(db *sql.DB) BenchmarkObservationRepository {
	return newRepository(db)
}

func NewIndustryChainRepository(db *sql.DB) IndustryChainRepository {
	return newRepository(db)
}

type rawDocumentScanner interface {
	Scan(dest ...any) error
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullablePositiveInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}
