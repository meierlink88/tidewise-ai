package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

type Credential struct {
	Value string
}

type RawResponse struct {
	ContentType string
	Content     []byte
	CollectedAt time.Time
}

type RawDocumentCandidate struct {
	ID               string
	SourceID         string
	IngestChannel    string
	SourceType       string
	SourceName       string
	SourceURL        string
	SourceExternalID string
	Title            string
	ContentText      string
	RawObjectURI     string
	RawMIMEType      string
	Language         string
	PublishedAt      *time.Time
	CollectedAt      time.Time
	ContentHash      string
	IngestStatus     domain.IngestStatus
}

type Connector interface {
	Fetch(context.Context, domain.SourceCatalog, Credential) (RawResponse, error)
}

type Parser interface {
	Parse(context.Context, domain.SourceCatalog, RawResponse) ([]RawDocumentCandidate, error)
}

type SourceRegistry struct {
	repository repositories.SourceCatalogRepository
}

func NewSourceRegistry(repository repositories.SourceCatalogRepository) SourceRegistry {
	return SourceRegistry{repository: repository}
}

func (r SourceRegistry) ActiveSources(ctx context.Context, filter repositories.SourceCatalogFilter) ([]domain.SourceCatalog, error) {
	return r.repository.ActiveSources(ctx, filter)
}

type Registry struct {
	connectors map[string]Connector
	parsers    map[string]Parser
}

func NewRegistry() *Registry {
	return &Registry{
		connectors: map[string]Connector{},
		parsers:    map[string]Parser{},
	}
}

func (r *Registry) RegisterConnector(key string, connector Connector) {
	r.connectors[key] = connector
}

func (r *Registry) RegisterParser(key string, parser Parser) {
	r.parsers[key] = parser
}

func (r *Registry) Connector(key string) (Connector, error) {
	connector, ok := r.connectors[key]
	if !ok {
		return nil, fmt.Errorf("connector %q is not registered", key)
	}
	return connector, nil
}

func (r *Registry) Parser(key string) (Parser, error) {
	parser, ok := r.parsers[key]
	if !ok {
		return nil, fmt.Errorf("parser %q is not registered", key)
	}
	return parser, nil
}

type EnvCredentialResolver struct{}

func (EnvCredentialResolver) Resolve(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	if !strings.HasPrefix(ref, "env:") {
		return "", fmt.Errorf("unsupported credential ref")
	}

	name := strings.TrimPrefix(ref, "env:")
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("credential ref is not available")
	}

	return value, nil
}

type RateLimitPolicy struct {
	RequestsPerMinute int
}

type RateLimiter struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
	now      func() time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		lastSeen: map[string]time.Time{},
		now:      time.Now,
	}
}

func (l *RateLimiter) Allow(providerKey string, policy RateLimitPolicy) error {
	if policy.RequestsPerMinute <= 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	minInterval := time.Minute / time.Duration(policy.RequestsPerMinute)
	if previous, ok := l.lastSeen[providerKey]; ok && now.Sub(previous) < minInterval {
		return fmt.Errorf("provider %q is rate limited", providerKey)
	}
	l.lastSeen[providerKey] = now

	return nil
}

type RawObject struct {
	Name        string
	ContentType string
	Content     []byte
}

type LocalRawObjectStore struct {
	Root string
}

func (s LocalRawObjectStore) Save(_ context.Context, object RawObject) (string, error) {
	if s.Root == "" {
		return "", fmt.Errorf("raw object store root is required")
	}
	if len(object.Content) == 0 {
		return "", fmt.Errorf("raw object content is required")
	}
	if err := os.MkdirAll(s.Root, 0o700); err != nil {
		return "", fmt.Errorf("create raw object store: %w", err)
	}

	sum := sha256.Sum256(object.Content)
	name := strings.TrimSpace(object.Name)
	if name == "" {
		name = hex.EncodeToString(sum[:])
	}
	path := filepath.Join(s.Root, name)
	if err := os.WriteFile(path, object.Content, 0o600); err != nil {
		return "", fmt.Errorf("write raw object: %w", err)
	}

	return "file://" + path, nil
}

type RawDocumentWriter struct {
	repository repositories.RawDocumentRepository
}

func NewRawDocumentWriter(repository repositories.RawDocumentRepository) RawDocumentWriter {
	return RawDocumentWriter{repository: repository}
}

func (w RawDocumentWriter) Write(ctx context.Context, candidate RawDocumentCandidate) (repositories.RawDocumentWriteResult, error) {
	doc := domain.RawDocument{
		ID:               candidate.ID,
		SourceID:         candidate.SourceID,
		IngestChannel:    candidate.IngestChannel,
		SourceType:       candidate.SourceType,
		SourceName:       candidate.SourceName,
		SourceURL:        candidate.SourceURL,
		SourceExternalID: candidate.SourceExternalID,
		Title:            strings.TrimSpace(candidate.Title),
		ContentText:      strings.TrimSpace(candidate.ContentText),
		RawObjectURI:     candidate.RawObjectURI,
		RawMIMEType:      candidate.RawMIMEType,
		Language:         candidate.Language,
		PublishedAt:      candidate.PublishedAt,
		CollectedAt:      candidate.CollectedAt,
		ContentHash:      candidate.ContentHash,
		IngestStatus:     candidate.IngestStatus,
	}
	if doc.ContentHash == "" {
		doc.ContentHash = contentHash(doc.Title, doc.ContentText)
	}
	if doc.IngestStatus == "" {
		doc.IngestStatus = domain.IngestStatusCollected
	}

	return w.repository.UpsertRawDocument(ctx, doc)
}

func contentHash(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		hash.Write([]byte(value))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
