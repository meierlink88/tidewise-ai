package researchanalysiscontext

import (
	"context"
	"database/sql"
	"fmt"

	entitybiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/entity"
	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/postgres"
)

type contextStore interface {
	ListBundles(context.Context, biz.StoreQuery) (biz.StorePage, error)
	ReferenceClosure(context.Context, biz.ReferenceClosureQuery) (biz.Dictionaries, error)
}

type Repository struct {
	store contextStore
}

func NewRepository(db *sql.DB) Repository {
	return Repository{store: postgres.NewResearchAnalysisContextStore(db)}
}

func (r Repository) ListBundles(
	ctx context.Context,
	query biz.StoreQuery,
) (biz.StorePage, error) {
	return r.store.ListBundles(ctx, query)
}

func (r Repository) ReferenceClosure(
	ctx context.Context,
	query biz.ReferenceClosureQuery,
) (biz.Dictionaries, error) {
	dictionaries, err := r.store.ReferenceClosure(ctx, query)
	if err != nil {
		return biz.Dictionaries{}, err
	}
	for _, item := range dictionaries.EntityTypeDefinitions {
		definition := entitybiz.EntityTypeDefinition{
			TypeKey: item.TypeKey, Version: item.Version, NameZH: item.NameZH, NameEN: item.NameEN,
			BusinessDefinition: item.BusinessDefinition, InclusionCriteria: item.InclusionCriteria,
			ExclusionCriteria: item.ExclusionCriteria, EventLinkAllowed: item.EventLinkAllowed,
			SignalSubjectAllowed: item.SignalSubjectAllowed, DirectTargetMode: item.DirectTargetMode,
			AllowedEventRoles: item.AllowedEventRoles,
			Status:            entitybiz.EntityTypeDefinitionStatus(item.Status),
		}
		if err := definition.Validate(); err != nil {
			return biz.Dictionaries{}, fmt.Errorf("validate persisted Research Entity Type Definition %q version %d: %w", item.TypeKey, item.Version, err)
		}
		if definition.Status != entitybiz.EntityTypeDefinitionActive {
			return biz.Dictionaries{}, fmt.Errorf("validate persisted Research Entity Type Definition %q version %d: status is not active", item.TypeKey, item.Version)
		}
	}
	return dictionaries, nil
}

var _ biz.Store = Repository{}
