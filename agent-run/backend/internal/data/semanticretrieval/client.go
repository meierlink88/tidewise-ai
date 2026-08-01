package semanticretrieval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/google/uuid"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

const (
	EntityCollection   = "entity_semantic_v1"
	VariableCollection = "variable_definition_semantic_v1"
	EmbeddingModel     = "text-embedding-v4"
	VectorSize         = 1024
)

type Config struct {
	QdrantURL        string
	QdrantAPIKey     string
	Embedder         embedding.Embedder
	EntityCollection string
	VectorSize       int
	Timeout          time.Duration
	MaxResponseBytes int64
	HTTPClient       *http.Client
}

type Client struct {
	qdrantURL, qdrantAPIKey string
	embedder                embedding.Embedder
	entityCollection        string
	vectorSize              int
	http                    *http.Client
	maxResponseBytes        int64
}

func New(config Config) (*Client, error) {
	if config.Embedder == nil || config.EntityCollection != EntityCollection ||
		config.VectorSize != VectorSize || config.Timeout <= 0 || config.MaxResponseBytes <= 0 {
		return nil, errors.New("semantic retrieval fixed contract is invalid")
	}
	qdrantURL, err := endpoint(config.QdrantURL)
	if err != nil {
		return nil, errors.New("Qdrant URL is invalid")
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.Timeout}
	}
	return &Client{
		qdrantURL: qdrantURL, qdrantAPIKey: config.QdrantAPIKey,
		embedder: config.Embedder, entityCollection: config.EntityCollection,
		vectorSize: config.VectorSize, http: httpClient, maxResponseBytes: config.MaxResponseBytes,
	}, nil
}

func endpoint(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid endpoint")
	}
	return parsed.String(), nil
}

func (c *Client) ExactEntities(ctx context.Context, lookups []eventsemantic.EntityLookup) ([]eventsemantic.EntityCandidateSet, error) {
	result := emptySets(lookups)
	if len(lookups) == 0 {
		return result, nil
	}
	normalized := make([]string, 0, len(lookups))
	for _, lookup := range lookups {
		normalized = append(normalized, NormalizeName(lookup.Mention))
	}
	payload, _ := json.Marshal(map[string]any{
		"filter": map[string]any{"must": []any{
			map[string]any{"key": "status", "match": map[string]any{"value": "active"}},
			map[string]any{"key": "normalized_names", "match": map[string]any{"any": normalized}},
		}},
		"limit": 10000, "with_payload": true, "with_vector": false,
	})
	var response struct {
		Result *struct {
			Points *[]qdrantPoint `json:"points"`
		} `json:"result"`
	}
	if err := c.do(ctx, c.qdrantURL+"/collections/"+url.PathEscape(c.entityCollection)+"/points/scroll", c.qdrantAPIKey, payload, &response); err != nil {
		return nil, err
	}
	if response.Result == nil || response.Result.Points == nil {
		return nil, retrievalError("qdrant_response_invalid", false)
	}
	for _, point := range *response.Result.Points {
		if !point.valid("") {
			return nil, retrievalError("qdrant_response_invalid", false)
		}
	}
	for _, lookup := range lookups {
		key := NormalizeName(lookup.Mention)
		for _, point := range *response.Result.Points {
			if point.Payload.EntityType == lookup.PredictedEntityType && contains(point.Payload.NormalizedNames, key) {
				resultSet(result, lookup.CandidateKey, point.candidate())
			}
		}
	}
	return result, nil
}

func (c *Client) SearchEntities(ctx context.Context, lookups []eventsemantic.EntityLookup, topK int) ([]eventsemantic.EntityCandidateSet, error) {
	result := emptySets(lookups)
	if len(lookups) == 0 {
		return result, nil
	}
	if topK <= 0 || topK > 20 {
		return nil, errors.New("semantic retrieval topK is invalid")
	}
	texts := make([]string, 0, len(lookups))
	for _, lookup := range lookups {
		texts = append(texts, lookup.Mention)
	}
	vectors, err := c.embedder.EmbedStrings(ctx, texts)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, retrievalError("embedding_unavailable", true)
	}
	if len(vectors) != len(texts) {
		return nil, retrievalError("embedding_response_invalid", false)
	}
	for _, vector := range vectors {
		if len(vector) != c.vectorSize {
			return nil, retrievalError("embedding_response_invalid", false)
		}
	}
	searches := make([]any, 0, len(lookups))
	for index, lookup := range lookups {
		searches = append(searches, map[string]any{
			"query": vectors[index],
			"filter": map[string]any{"must": []any{
				map[string]any{"key": "entity_type", "match": map[string]any{"value": lookup.PredictedEntityType}},
				map[string]any{"key": "status", "match": map[string]any{"value": "active"}},
			}},
			"limit": topK, "with_payload": true, "with_vector": false,
		})
	}
	payload, _ := json.Marshal(map[string]any{"searches": searches})
	var response struct {
		Result *[]struct {
			Points *[]qdrantPoint `json:"points"`
		} `json:"result"`
	}
	if err := c.do(ctx, c.qdrantURL+"/collections/"+url.PathEscape(c.entityCollection)+"/points/query/batch", c.qdrantAPIKey, payload, &response); err != nil {
		return nil, err
	}
	if response.Result == nil || len(*response.Result) != len(lookups) {
		return nil, retrievalError("qdrant_response_invalid", false)
	}
	for index, search := range *response.Result {
		if search.Points == nil {
			return nil, retrievalError("qdrant_response_invalid", false)
		}
		for _, point := range *search.Points {
			if !point.valid(lookups[index].PredictedEntityType) {
				return nil, retrievalError("qdrant_response_invalid", false)
			}
			result[index].Candidates = append(result[index].Candidates, point.candidate())
		}
	}
	return result, nil
}

type qdrantPoint struct {
	ID      any     `json:"id"`
	Score   float64 `json:"score"`
	Payload struct {
		EntityID        string   `json:"entity_id"`
		EntityType      string   `json:"entity_type"`
		Name            string   `json:"name"`
		CanonicalName   string   `json:"canonical_name"`
		Aliases         []string `json:"aliases"`
		NormalizedNames []string `json:"normalized_names"`
		Description     string   `json:"description"`
		Status          string   `json:"status"`
	} `json:"payload"`
}

func (p qdrantPoint) valid(expectedEntityType string) bool {
	pointID, ok := p.ID.(string)
	if !ok {
		return false
	}
	parsed, err := uuid.Parse(pointID)
	if err != nil || parsed.String() != pointID || p.Payload.EntityID != pointID ||
		strings.TrimSpace(p.Payload.EntityType) == "" || strings.TrimSpace(p.Payload.Name) == "" ||
		strings.TrimSpace(p.Payload.CanonicalName) == "" || p.Payload.Status != "active" ||
		len(p.Payload.NormalizedNames) == 0 ||
		(expectedEntityType != "" && p.Payload.EntityType != expectedEntityType) {
		return false
	}
	for _, normalized := range p.Payload.NormalizedNames {
		if strings.TrimSpace(normalized) == "" || NormalizeName(normalized) != normalized {
			return false
		}
	}
	return true
}

func (p qdrantPoint) candidate() eventsemantic.EntityCandidate {
	return eventsemantic.EntityCandidate{
		Entity: eventsemantic.Entity{
			EntityID: p.Payload.EntityID, EntityType: p.Payload.EntityType, Name: p.Payload.Name,
			CanonicalName: p.Payload.CanonicalName, Aliases: p.Payload.Aliases,
			Description: p.Payload.Description, Status: p.Payload.Status,
		},
		Score: p.Score,
	}
}

func (c *Client) do(ctx context.Context, endpoint, apiKey string, payload []byte, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return retrievalError("semantic_retrieval_request_invalid", false)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if apiKey != "" {
		request.Header.Set("api-key", apiKey)
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return retrievalError("semantic_retrieval_unavailable", true)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil || int64(len(body)) > c.maxResponseBytes || response.StatusCode < 200 || response.StatusCode >= 300 {
		return retrievalError("semantic_retrieval_unavailable", response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return retrievalError("semantic_retrieval_response_invalid", false)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return retrievalError("semantic_retrieval_response_invalid", false)
	}
	return nil
}

func retrievalError(code string, retryable bool) error {
	return &eventsemantic.RemoteError{Code: code, Summary: "semantic retrieval failed", Retryable: retryable}
}

func emptySets(lookups []eventsemantic.EntityLookup) []eventsemantic.EntityCandidateSet {
	result := make([]eventsemantic.EntityCandidateSet, 0, len(lookups))
	for _, lookup := range lookups {
		result = append(result, eventsemantic.EntityCandidateSet{CandidateKey: lookup.CandidateKey})
	}
	return result
}

func resultSet(sets []eventsemantic.EntityCandidateSet, key string, candidate eventsemantic.EntityCandidate) {
	for index := range sets {
		if sets[index].CandidateKey == key {
			sets[index].Candidates = append(sets[index].Candidates, candidate)
			return
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func NormalizeName(value string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsSpace(character) || unicode.IsPunct(character) {
			continue
		}
		builder.WriteRune(character)
	}
	return builder.String()
}

var _ eventsemantic.SemanticRetriever = (*Client)(nil)

func (c Config) String() string {
	return fmt.Sprintf("qdrant=%s entity_collection=%s vector_size=%d", c.QdrantURL, c.EntityCollection, c.VectorSize)
}
