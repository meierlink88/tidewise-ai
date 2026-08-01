package semanticprojection

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	projection "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/semanticprojection"
)

type PostgresSource struct{ database *sql.DB }

func NewPostgresSource(database *sql.DB) (*PostgresSource, error) {
	if database == nil {
		return nil, errors.New("semantic projection PostgreSQL is required")
	}
	return &PostgresSource{database: database}, nil
}

func (s *PostgresSource) Current(ctx context.Context) (projection.Snapshot, error) {
	result := projection.Snapshot{}
	rows, err := s.database.QueryContext(ctx, `
		SELECT id, entity_type, layer_code, name, canonical_name, array_to_json(aliases)::text, status
		FROM entity_nodes entity
		WHERE entity.status = 'active'
		  AND EXISTS (
		      SELECT 1
		      FROM entity_type_definitions definition
		      WHERE definition.type_key = entity.entity_type
		        AND definition.status = 'active'
		        AND definition.event_link_allowed
		  )
		ORDER BY id
	`)
	if err != nil {
		return result, fmt.Errorf("read active Entity projection source: %w", err)
	}
	for rows.Next() {
		var item projection.EntitySource
		var aliasesJSON string
		if err := rows.Scan(&item.ID, &item.EntityType, &item.LayerCode, &item.Name, &item.CanonicalName, &aliasesJSON, &item.Status); err != nil {
			rows.Close()
			return result, err
		}
		if err := decodeStringArray(aliasesJSON, &item.Aliases); err != nil {
			rows.Close()
			return result, err
		}
		result.Entities = append(result.Entities, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, fmt.Errorf("iterate active Entity projection source: %w", err)
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	rows, err = s.database.QueryContext(ctx, `
		SELECT DISTINCT ON (definition.variable_key)
		       definition.variable_key, definition.version, definition.name_zh, definition.name_en,
		       definition.business_definition, definition.domain, definition.value_type,
		       array_to_json(definition.allowed_units)::text,
		       array_to_json(definition.allowed_directions)::text, definition.status,
		       array_to_json(COALESCE((
		           SELECT array_agg(applicable.entity_type ORDER BY applicable.entity_type)
		           FROM variable_definition_entity_types applicable
		           WHERE applicable.variable_key = definition.variable_key
		             AND applicable.variable_version = definition.version
		       ), ARRAY[]::text[]))::text
		FROM variable_definitions definition
		WHERE definition.status = 'active'
		ORDER BY definition.variable_key, definition.version DESC
	`)
	if err != nil {
		return result, fmt.Errorf("read active Variable Definition projection source: %w", err)
	}
	for rows.Next() {
		var item projection.VariableSource
		var unitsJSON, directionsJSON, applicableTypesJSON string
		if err := rows.Scan(
			&item.Key, &item.Version, &item.NameZH, &item.NameEN, &item.BusinessDefinition,
			&item.Domain, &item.ValueType, &unitsJSON, &directionsJSON,
			&item.Status, &applicableTypesJSON,
		); err != nil {
			rows.Close()
			return result, err
		}
		if err := decodeStringArray(unitsJSON, &item.AllowedUnits); err != nil {
			rows.Close()
			return result, err
		}
		if err := decodeStringArray(directionsJSON, &item.AllowedDirections); err != nil {
			rows.Close()
			return result, err
		}
		if err := decodeStringArray(applicableTypesJSON, &item.ApplicableEntityTypes); err != nil {
			rows.Close()
			return result, err
		}
		result.Variables = append(result.Variables, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, fmt.Errorf("iterate active Variable Definition projection source: %w", err)
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	return result, nil
}

func decodeStringArray(raw string, target *[]string) error {
	if err := json.Unmarshal([]byte(raw), target); err != nil || *target == nil {
		return errors.New("semantic projection source array is invalid")
	}
	return nil
}

type HTTPConfig struct {
	Endpoint         string
	APIKey           string
	Timeout          time.Duration
	MaxResponseBytes int64
	BatchSize        int
	HTTPClient       *http.Client
}

type OpenAIEmbedder struct {
	endpoint, apiKey string
	http             *http.Client
	maxResponseBytes int64
	batchSize        int
}

func NewOpenAIEmbedder(config HTTPConfig) (*OpenAIEmbedder, error) {
	endpoint, err := validateEndpoint(config.Endpoint)
	if err != nil || config.Timeout <= 0 || config.MaxResponseBytes <= 0 || config.BatchSize <= 0 {
		return nil, errors.New("embedding projection configuration is invalid")
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}
	return &OpenAIEmbedder{
		endpoint: endpoint, apiKey: config.APIKey, http: client,
		maxResponseBytes: config.MaxResponseBytes, batchSize: config.BatchSize,
	}, nil
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, documents []string) ([][]float32, error) {
	result := make([][]float32, 0, len(documents))
	for start := 0; start < len(documents); start += e.batchSize {
		end := min(start+e.batchSize, len(documents))
		payload, _ := json.Marshal(map[string]any{
			"model": projection.EmbeddingModel, "input": documents[start:end],
			"dimensions": projection.VectorSize, "encoding_format": "float",
		})
		var response struct {
			Data []struct {
				Index     int       `json:"index"`
				Embedding []float32 `json:"embedding"`
			} `json:"data"`
		}
		if err := postJSON(ctx, e.http, e.endpoint+"/embeddings", e.apiKey, e.maxResponseBytes, payload, &response); err != nil {
			return nil, err
		}
		sort.Slice(response.Data, func(i, j int) bool { return response.Data[i].Index < response.Data[j].Index })
		if len(response.Data) != end-start {
			return nil, errors.New("embedding response count is invalid")
		}
		for index, item := range response.Data {
			if item.Index != index || len(item.Embedding) != projection.VectorSize {
				return nil, errors.New("embedding response vector contract is invalid")
			}
			result = append(result, item.Embedding)
		}
	}
	return result, nil
}

type QdrantStore struct {
	endpoint, apiKey   string
	http               *http.Client
	maxResponseBytes   int64
	batchSize          int
	entityCollection   string
	variableCollection string
}

func NewQdrantStore(config HTTPConfig) (*QdrantStore, error) {
	endpoint, err := validateEndpoint(config.Endpoint)
	if err != nil || config.Timeout <= 0 || config.MaxResponseBytes <= 0 || config.BatchSize <= 0 {
		return nil, errors.New("Qdrant projection configuration is invalid")
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: config.Timeout}
	}
	return &QdrantStore{
		endpoint: endpoint, apiKey: config.APIKey, http: client,
		maxResponseBytes: config.MaxResponseBytes, batchSize: config.BatchSize,
		entityCollection:   projection.EntityCollection,
		variableCollection: projection.VariableDefinitionCollection,
	}, nil
}

func (s *QdrantStore) Replace(ctx context.Context, collection string, vectorSize int, points []projection.Point) error {
	if (collection != projection.EntityCollection && collection != projection.VariableDefinitionCollection) || vectorSize != projection.VectorSize {
		return errors.New("Qdrant replacement target is outside the semantic projection contract")
	}
	physicalCollection := s.entityCollection
	if collection == projection.VariableDefinitionCollection {
		physicalCollection = s.variableCollection
	}
	collectionURL := s.endpoint + "/collections/" + url.PathEscape(physicalCollection)
	if err := requestJSON(ctx, s.http, http.MethodDelete, collectionURL, s.apiKey, s.maxResponseBytes, nil, nil, http.StatusNotFound); err != nil {
		return err
	}
	createPayload, _ := json.Marshal(map[string]any{"vectors": map[string]any{"size": vectorSize, "distance": "Cosine"}})
	if err := requestJSON(ctx, s.http, http.MethodPut, collectionURL, s.apiKey, s.maxResponseBytes, createPayload, nil); err != nil {
		return err
	}
	indexes := []string{"status"}
	if collection == projection.EntityCollection {
		indexes = append(indexes, "entity_type", "normalized_names")
	} else {
		indexes = append(indexes, "applicable_entity_types", "variable_key")
	}
	for _, field := range indexes {
		payload, _ := json.Marshal(map[string]any{"field_name": field, "field_schema": "keyword"})
		if err := requestJSON(ctx, s.http, http.MethodPut, collectionURL+"/index", s.apiKey, s.maxResponseBytes, payload, nil); err != nil {
			return err
		}
	}
	for start := 0; start < len(points); start += s.batchSize {
		end := min(start+s.batchSize, len(points))
		payload, _ := json.Marshal(map[string]any{"points": points[start:end]})
		if err := requestJSON(ctx, s.http, http.MethodPut, collectionURL+"/points?wait=true", s.apiKey, s.maxResponseBytes, payload, nil); err != nil {
			return err
		}
	}
	return nil
}

func validateEndpoint(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(raw), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("endpoint is invalid")
	}
	return parsed.String(), nil
}

func postJSON(ctx context.Context, client *http.Client, endpoint, apiKey string, maxBytes int64, payload []byte, target any) error {
	return requestJSON(ctx, client, http.MethodPost, endpoint, apiKey, maxBytes, payload, target)
}

func requestJSON(
	ctx context.Context,
	client *http.Client,
	method, endpoint, apiKey string,
	maxBytes int64,
	payload []byte,
	target any,
	allowedStatuses ...int,
) error {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payload))
	if err != nil {
		return errors.New("semantic projection request is invalid")
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		request.Header.Set("api-key", apiKey)
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	response, err := client.Do(request)
	if err != nil {
		return errors.New("semantic projection endpoint is unavailable")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil || int64(len(body)) > maxBytes {
		return errors.New("semantic projection response is unavailable")
	}
	allowed := response.StatusCode >= 200 && response.StatusCode < 300
	for _, status := range allowedStatuses {
		allowed = allowed || response.StatusCode == status
	}
	if !allowed {
		return fmt.Errorf("semantic projection endpoint returned HTTP %d", response.StatusCode)
	}
	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			return errors.New("semantic projection response is invalid")
		}
	}
	return nil
}

var _ projection.Source = (*PostgresSource)(nil)
var _ projection.Embedder = (*OpenAIEmbedder)(nil)
var _ projection.Store = (*QdrantStore)(nil)
