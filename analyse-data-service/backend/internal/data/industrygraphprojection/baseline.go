package industrygraphprojection

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	graphbiz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
)

const (
	nodesProjectionPath         = "neo4j/nodes.csv"
	relationshipsProjectionPath = "neo4j/relationships.csv"

	FrozenV1NodesCSVSHA256         = "2710ca446884fbdb2d54a52f730d2d5c4bf991608074197b3096859e039e3ea7"
	FrozenV1RelationshipsCSVSHA256 = "50a3d59e277724e4c4071e3ea545466a2c503b3c05dffb1337cbd68991b18e56"

	maxProjectionCSVBytes = 256 << 20
)

var (
	nodeCSVHeader = []string{
		"entity_id:ID",
		"entity_key",
		"canonical_name",
		"aliases",
		":LABEL",
	}
	relationshipCSVHeader = []string{
		":START_ID",
		":END_ID",
		":TYPE",
		"chain_id",
		"relation_key",
		"contextual_stage",
		"position",
		"mechanism",
	}
)

func LoadFrozenV1CSVBaseline(
	pkg industryrelationshipimport.Package,
) (graphbiz.Projection, error) {
	if !graphbiz.ValidPackageSHA256(pkg.Manifest.PackageSHA256) {
		return graphbiz.Projection{}, errors.New(
			"relationship package manifest has an invalid package SHA-256",
		)
	}
	if pkg.Manifest.PackageSHA256 != graphbiz.FrozenV1PackageSHA256 {
		return graphbiz.Projection{}, fmt.Errorf(
			"relationship package SHA-256 %s, want frozen V1 package SHA-256 %s",
			pkg.Manifest.PackageSHA256,
			graphbiz.FrozenV1PackageSHA256,
		)
	}

	nodeContent, err := readProjectionCSV(
		pkg,
		nodesProjectionPath,
		FrozenV1NodesCSVSHA256,
	)
	if err != nil {
		return graphbiz.Projection{}, fmt.Errorf("read Industry graph nodes baseline: %w", err)
	}
	nodes, err := parseNodeCSV(nodeContent)
	if err != nil {
		return graphbiz.Projection{}, fmt.Errorf("parse Industry graph nodes baseline: %w", err)
	}

	relationshipContent, err := readProjectionCSV(
		pkg,
		relationshipsProjectionPath,
		FrozenV1RelationshipsCSVSHA256,
	)
	if err != nil {
		return graphbiz.Projection{}, fmt.Errorf(
			"read Industry graph relationships baseline: %w",
			err,
		)
	}
	relationships, err := parseRelationshipCSV(relationshipContent)
	if err != nil {
		return graphbiz.Projection{}, fmt.Errorf(
			"parse Industry graph relationships baseline: %w",
			err,
		)
	}

	projection := graphbiz.Projection{
		PackageSHA256: pkg.Manifest.PackageSHA256,
		Nodes:         nodes,
		Relationships: relationships,
	}
	if err := graphbiz.ValidateFrozenV1Projection(projection); err != nil {
		return graphbiz.Projection{}, fmt.Errorf(
			"validate Industry graph CSV baseline: %w",
			err,
		)
	}
	return projection, nil
}

func readProjectionCSV(
	pkg industryrelationshipimport.Package,
	expectedPath string,
	expectedSHA256 string,
) ([]byte, error) {
	if strings.TrimSpace(pkg.Directory) == "" {
		return nil, errors.New("relationship package directory is required")
	}
	descriptor, ok := pkg.Manifest.ProjectionFiles[expectedPath]
	if !ok {
		return nil, fmt.Errorf("relationship manifest is missing projection file %q", expectedPath)
	}
	if descriptor.Path != expectedPath {
		return nil, fmt.Errorf(
			"projection file %q path is %q, want %q",
			expectedPath,
			descriptor.Path,
			expectedPath,
		)
	}
	if descriptor.SHA256 != expectedSHA256 {
		return nil, fmt.Errorf(
			"projection file %q SHA-256 %q, want frozen V1 SHA-256 %s",
			expectedPath,
			descriptor.SHA256,
			expectedSHA256,
		)
	}

	packageDirectory, err := filepath.Abs(pkg.Directory)
	if err != nil {
		return nil, fmt.Errorf("resolve relationship package directory: %w", err)
	}
	target := filepath.Join(packageDirectory, filepath.FromSlash(expectedPath))
	relative, err := filepath.Rel(packageDirectory, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("projection file %q escapes the relationship package", expectedPath)
	}
	content, err := readLimitedProjectionFile(target)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != expectedSHA256 {
		return nil, fmt.Errorf("projection file %q SHA-256 mismatch", expectedPath)
	}
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("projection file %q is not valid UTF-8", expectedPath)
	}
	return content, nil
}

func readLimitedProjectionFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxProjectionCSVBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxProjectionCSVBytes {
		return nil, fmt.Errorf("projection CSV exceeds %d bytes", maxProjectionCSVBytes)
	}
	return content, nil
}

func parseNodeCSV(content []byte) ([]graphbiz.Node, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if !reflect.DeepEqual(header, nodeCSVHeader) {
		return nil, fmt.Errorf("header = %q, want %q", header, nodeCSVHeader)
	}
	reader.FieldsPerRecord = len(nodeCSVHeader)

	nodes := make([]graphbiz.Node, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNumber, err)
		}
		entityType, err := parseNodeLabel(record[4])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNumber, err)
		}
		aliases, err := parseAliases(record[3])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNumber, err)
		}
		nodes = append(nodes, graphbiz.Node{
			EntityID:      record[0],
			EntityKey:     record[1],
			CanonicalName: record[2],
			Aliases:       aliases,
			EntityType:    entityType,
		})
	}
	return nodes, nil
}

func parseNodeLabel(value string) (graphbiz.EntityType, error) {
	switch value {
	case "Industry":
		return graphbiz.EntityTypeIndustry, nil
	case "Concept":
		return graphbiz.EntityTypeConcept, nil
	case "IndustryChain":
		return graphbiz.EntityTypeIndustryChain, nil
	case "ChainNode":
		return graphbiz.EntityTypeChainNode, nil
	default:
		return "", fmt.Errorf("unsupported node label %q", value)
	}
}

func parseAliases(value string) ([]string, error) {
	aliases := make([]string, 0)
	if value == "" {
		return aliases, nil
	}
	seen := make(map[string]struct{})
	for _, alias := range strings.Split(value, "|") {
		if alias == "" || strings.TrimSpace(alias) != alias {
			return nil, fmt.Errorf("aliases %q contain a blank or non-canonical alias", value)
		}
		if _, duplicate := seen[alias]; duplicate {
			return nil, fmt.Errorf("aliases %q contain duplicate alias %q", value, alias)
		}
		seen[alias] = struct{}{}
		aliases = append(aliases, alias)
	}
	return aliases, nil
}

func parseRelationshipCSV(content []byte) ([]graphbiz.Relationship, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if !reflect.DeepEqual(header, relationshipCSVHeader) {
		return nil, fmt.Errorf("header = %q, want %q", header, relationshipCSVHeader)
	}
	reader.FieldsPerRecord = len(relationshipCSVHeader)

	relationships := make([]graphbiz.Relationship, 0)
	for rowNumber := 2; ; rowNumber++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNumber, err)
		}
		relationshipType, err := parseRelationshipType(record[2])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNumber, err)
		}
		position, err := parseRelationshipPosition(relationshipType, record[6])
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowNumber, err)
		}
		if relationshipType == graphbiz.RelationshipTypeHasNode {
			switch record[5] {
			case "upstream", "midstream", "downstream":
			default:
				return nil, fmt.Errorf(
					"row %d: HAS_NODE has invalid contextual_stage %q",
					rowNumber,
					record[5],
				)
			}
		} else if record[5] != "" {
			return nil, fmt.Errorf(
				"row %d: %s must not contain contextual_stage",
				rowNumber,
				relationshipType,
			)
		}
		relationships = append(relationships, graphbiz.Relationship{
			FromEntityID:    record[0],
			ToEntityID:      record[1],
			Type:            relationshipType,
			ChainID:         record[3],
			RelationKey:     record[4],
			ContextualStage: record[5],
			Position:        position,
			Mechanism:       record[7],
		})
	}
	return relationships, nil
}

func parseRelationshipType(value string) (graphbiz.RelationshipType, error) {
	switch graphbiz.RelationshipType(value) {
	case graphbiz.RelationshipTypeMappedToIndustry,
		graphbiz.RelationshipTypeMappedToConcept,
		graphbiz.RelationshipTypeHasNode,
		graphbiz.RelationshipTypeInputTo,
		graphbiz.RelationshipTypeIsComponentOf,
		graphbiz.RelationshipTypeDependsOn,
		graphbiz.RelationshipTypeIsSubcategoryOf:
		return graphbiz.RelationshipType(value), nil
	default:
		return "", fmt.Errorf("unsupported relationship type %q", value)
	}
}

func parseRelationshipPosition(
	relationshipType graphbiz.RelationshipType,
	value string,
) (*int, error) {
	if relationshipType != graphbiz.RelationshipTypeHasNode {
		if value != "" {
			return nil, fmt.Errorf("%s must not contain position", relationshipType)
		}
		return nil, nil
	}
	position, err := strconv.Atoi(value)
	if err != nil || position <= 0 || strconv.Itoa(position) != value {
		return nil, fmt.Errorf(
			"HAS_NODE position %q must be a canonical positive integer",
			value,
		)
	}
	return &position, nil
}
