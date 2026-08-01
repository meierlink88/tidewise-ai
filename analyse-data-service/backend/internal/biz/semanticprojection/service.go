package semanticprojection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	ProjectionVersion            = "event-semantic-projection.v1"
	EmbeddingModel               = "text-embedding-v4"
	VectorSize                   = 1024
	EntityCollection             = "entity_semantic_v1"
	VariableDefinitionCollection = "variable_definition_semantic_v1"
)

var variablePointNamespace = uuid.MustParse("eeaa872b-7433-5cf7-9001-4cfa62821baa")

type EntitySource struct {
	ID            string
	EntityType    string
	LayerCode     string
	Name          string
	CanonicalName string
	Aliases       []string
	Status        string
}

type VariableSource struct {
	Key                   string
	Version               int
	NameZH                string
	NameEN                string
	BusinessDefinition    string
	Domain                string
	ApplicableEntityTypes []string
	ValueType             string
	AllowedUnits          []string
	AllowedDirections     []string
	Status                string
}

type Snapshot struct {
	Entities  []EntitySource
	Variables []VariableSource
}

type Point struct {
	ID      string         `json:"id"`
	Vector  []float32      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type Source interface {
	Current(context.Context) (Snapshot, error)
}

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type Store interface {
	Replace(context.Context, string, int, []Point) error
}

type Result struct {
	ProjectionVersion string `json:"projection_version"`
	EmbeddingModel    string `json:"embedding_model"`
	EntityCount       int    `json:"entity_count"`
	VariableCount     int    `json:"variable_definition_count"`
}

type Service struct {
	source   Source
	embedder Embedder
	store    Store
}

func New(source Source, embedder Embedder, store Store) (*Service, error) {
	if source == nil || embedder == nil || store == nil {
		return nil, errors.New("semantic projection dependencies are required")
	}
	return &Service{source: source, embedder: embedder, store: store}, nil
}

func (s *Service) Rebuild(ctx context.Context) (Result, error) {
	snapshot, err := s.source.Current(ctx)
	if err != nil {
		return Result{}, err
	}
	entityDocuments, entityPayloads, err := entityProjection(snapshot.Entities)
	if err != nil {
		return Result{}, err
	}
	variableDocuments, variablePayloads, err := variableProjection(snapshot.Variables)
	if err != nil {
		return Result{}, err
	}
	entityVectors, err := s.embedder.Embed(ctx, entityDocuments)
	if err != nil {
		return Result{}, err
	}
	variableVectors, err := s.embedder.Embed(ctx, variableDocuments)
	if err != nil {
		return Result{}, err
	}
	entityPoints, err := buildPoints(entityPayloads, entityVectors)
	if err != nil {
		return Result{}, err
	}
	variablePoints, err := buildPoints(variablePayloads, variableVectors)
	if err != nil {
		return Result{}, err
	}
	if err := s.store.Replace(ctx, EntityCollection, VectorSize, entityPoints); err != nil {
		return Result{}, err
	}
	if err := s.store.Replace(ctx, VariableDefinitionCollection, VectorSize, variablePoints); err != nil {
		return Result{}, err
	}
	return Result{
		ProjectionVersion: ProjectionVersion, EmbeddingModel: EmbeddingModel,
		EntityCount: len(entityPoints), VariableCount: len(variablePoints),
	}, nil
}

func entityProjection(values []EntitySource) ([]string, []map[string]any, error) {
	documents := make([]string, 0, len(values))
	payloads := make([]map[string]any, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		if item.Status != "active" || uuid.Validate(item.ID) != nil || strings.TrimSpace(item.EntityType) == "" ||
			strings.TrimSpace(item.LayerCode) == "" ||
			strings.TrimSpace(item.CanonicalName) == "" {
			return nil, nil, errors.New("entity projection source is invalid")
		}
		if _, exists := seen[item.ID]; exists {
			return nil, nil, errors.New("entity projection identity is duplicated")
		}
		seen[item.ID] = struct{}{}
		aliases := cleanSorted(item.Aliases)
		normalizedNames := cleanSorted(append([]string{NormalizeName(item.ID), NormalizeName(item.Name), NormalizeName(item.CanonicalName)}, normalizeAll(aliases)...))
		description := "formal entity type: " + item.EntityType + "; layer: " + item.LayerCode
		document := strings.Join([]string{
			"实体类型: " + item.EntityType, "规范名称: " + item.CanonicalName,
			"常用名称: " + item.Name, "别名: " + strings.Join(aliases, " / "), "正式上下文: " + description,
		}, "\n")
		payload := map[string]any{
			"source_identity": item.ID, "entity_id": item.ID, "entity_type": item.EntityType,
			"name": item.Name, "canonical_name": item.CanonicalName, "aliases": aliases,
			"normalized_names": normalizedNames, "description": description, "status": item.Status,
			"projection_version": ProjectionVersion, "embedding_model": EmbeddingModel,
			"content_fingerprint": fingerprint(document), "point_id": item.ID,
		}
		documents = append(documents, document)
		payloads = append(payloads, payload)
	}
	return documents, payloads, nil
}

func variableProjection(values []VariableSource) ([]string, []map[string]any, error) {
	documents := make([]string, 0, len(values))
	payloads := make([]map[string]any, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		identity := item.Key + "@" + strconv.Itoa(item.Version)
		if item.Status != "active" || strings.TrimSpace(item.Key) == "" || item.Version <= 0 ||
			strings.TrimSpace(item.BusinessDefinition) == "" {
			return nil, nil, errors.New("Variable Definition projection source is invalid")
		}
		if _, exists := seen[identity]; exists {
			return nil, nil, errors.New("Variable Definition projection identity is duplicated")
		}
		seen[identity] = struct{}{}
		applicable := cleanSorted(item.ApplicableEntityTypes)
		units := cleanSorted(item.AllowedUnits)
		directions := cleanSorted(item.AllowedDirections)
		document := strings.Join([]string{
			"变量: " + item.NameZH + " / " + item.NameEN, "定义: " + item.BusinessDefinition,
			"领域: " + item.Domain, "适用实体类型: " + strings.Join(applicable, " / "),
			"值类型: " + item.ValueType, "允许方向: " + strings.Join(directions, " / "),
			"允许单位: " + strings.Join(units, " / "),
		}, "\n")
		pointID := uuid.NewSHA1(variablePointNamespace, []byte(identity)).String()
		payload := map[string]any{
			"source_identity": identity, "variable_key": item.Key, "variable_version": item.Version,
			"name_zh": item.NameZH, "name_en": item.NameEN, "business_definition": item.BusinessDefinition,
			"domain": item.Domain, "applicable_entity_types": applicable, "value_type": item.ValueType,
			"allowed_units": units, "allowed_directions": directions, "status": item.Status,
			"projection_version": ProjectionVersion, "embedding_model": EmbeddingModel,
			"content_fingerprint": fingerprint(document), "point_id": pointID,
		}
		documents = append(documents, document)
		payloads = append(payloads, payload)
	}
	return documents, payloads, nil
}

func buildPoints(payloads []map[string]any, vectors [][]float32) ([]Point, error) {
	if len(payloads) != len(vectors) {
		return nil, errors.New("embedding response count differs from projection documents")
	}
	result := make([]Point, 0, len(payloads))
	for index, payload := range payloads {
		if len(vectors[index]) != VectorSize {
			return nil, errors.New("embedding vector size differs from projection contract")
		}
		pointID, _ := payload["point_id"].(string)
		delete(payload, "point_id")
		result = append(result, Point{ID: pointID, Vector: vectors[index], Payload: payload})
	}
	return result, nil
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

func normalizeAll(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, NormalizeName(value))
	}
	return result
}

func cleanSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func fingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func CanonicalPayload(point Point) ([]byte, error) { return json.Marshal(point) }
