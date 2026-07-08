package repositories

import (
	"context"
	"fmt"
	"sync"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
)

type SourceCatalogFilter struct {
	ProviderKey   string
	IngestChannel string
}

type SourceCatalogRepository interface {
	ActiveSources(context.Context, SourceCatalogFilter) ([]domain.SourceCatalog, error)
}

type RawDocumentRepository interface {
	UpsertRawDocument(context.Context, domain.RawDocument) (RawDocumentWriteResult, error)
	UpdateRawDocumentStatus(context.Context, string, domain.IngestStatus) error
}

type RawDocumentWriteResult struct {
	Document    domain.RawDocument
	Created     bool
	DuplicateOf string
}

type InMemoryRepository struct {
	mu        sync.Mutex
	sources   []domain.SourceCatalog
	documents map[string]domain.RawDocument
}

func NewInMemoryRepository(sources []domain.SourceCatalog) *InMemoryRepository {
	copiedSources := make([]domain.SourceCatalog, len(sources))
	copy(copiedSources, sources)

	return &InMemoryRepository{
		sources:   copiedSources,
		documents: map[string]domain.RawDocument{},
	}
}

func (r *InMemoryRepository) SeedSource(_ context.Context, source domain.SourceCatalog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, existing := range r.sources {
		if existing.ID == source.ID {
			r.sources[index] = source
			return nil
		}
	}
	r.sources = append(r.sources, source)
	return nil
}

func (r *InMemoryRepository) ActiveSources(_ context.Context, filter SourceCatalogFilter) ([]domain.SourceCatalog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []domain.SourceCatalog
	for _, source := range r.sources {
		if source.Status != domain.SourceCatalogStatusActive {
			continue
		}
		if filter.ProviderKey != "" && source.ProviderKey != filter.ProviderKey {
			continue
		}
		if filter.IngestChannel != "" && source.IngestChannel != filter.IngestChannel {
			continue
		}
		result = append(result, source)
	}

	return result, nil
}

func (r *InMemoryRepository) UpsertRawDocument(_ context.Context, doc domain.RawDocument) (RawDocumentWriteResult, error) {
	if err := doc.Validate(); err != nil {
		return RawDocumentWriteResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.findDuplicate(doc); ok {
		return RawDocumentWriteResult{
			Document:    existing,
			Created:     false,
			DuplicateOf: existing.ID,
		}, nil
	}

	r.documents[doc.ID] = doc
	return RawDocumentWriteResult{
		Document: doc,
		Created:  true,
	}, nil
}

func (r *InMemoryRepository) UpdateRawDocumentStatus(_ context.Context, id string, status domain.IngestStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, ok := r.documents[id]
	if !ok {
		return fmt.Errorf("raw document %q not found", id)
	}
	doc.IngestStatus = status
	r.documents[id] = doc

	return nil
}

func (r *InMemoryRepository) RawDocument(id string) (domain.RawDocument, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, ok := r.documents[id]
	return doc, ok
}

func (r *InMemoryRepository) RawDocumentCount(_ context.Context, sourceID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, doc := range r.documents {
		if sourceID == "" || doc.SourceID == sourceID {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryRepository) findDuplicate(doc domain.RawDocument) (domain.RawDocument, bool) {
	for _, existing := range r.documents {
		if existing.SourceID != doc.SourceID {
			continue
		}
		if doc.SourceExternalID != "" && existing.SourceExternalID == doc.SourceExternalID {
			return existing, true
		}
		if existing.ContentHash == doc.ContentHash {
			return existing, true
		}
	}

	return domain.RawDocument{}, false
}
