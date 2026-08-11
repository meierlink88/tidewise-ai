package researchanalysiscontext

import (
	"context"
	"testing"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
)

func TestReferenceClosureRejectsMalformedPersistedEntityTypeDefinition(t *testing.T) {
	repository := Repository{store: contextStoreStub{dictionaries: biz.Dictionaries{
		EntityTypeDefinitions: []biz.EntityTypeContext{{
			TypeKey: "product", Version: 1, NameZH: "产品", NameEN: "Product",
			BusinessDefinition: "A marketable product", InclusionCriteria: []string{"identifiable product"},
			ExclusionCriteria: []string{"company"}, EventLinkAllowed: true, SignalSubjectAllowed: true,
			DirectTargetMode: "invalid", AllowedEventRoles: []string{"event_subject"}, Status: "active",
		}},
	}}}
	if _, err := repository.ReferenceClosure(context.Background(), biz.ReferenceClosureQuery{}); err == nil {
		t.Fatal("ReferenceClosure() error = nil, want malformed persisted Entity Type Definition rejection")
	}
}

type contextStoreStub struct{ dictionaries biz.Dictionaries }

func (s contextStoreStub) ListBundles(context.Context, biz.StoreQuery) (biz.StorePage, error) {
	return biz.StorePage{}, nil
}

func (s contextStoreStub) ReferenceClosure(context.Context, biz.ReferenceClosureQuery) (biz.Dictionaries, error) {
	return s.dictionaries, nil
}
