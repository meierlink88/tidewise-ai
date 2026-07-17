package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/internal/domain"
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
