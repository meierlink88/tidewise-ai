package neo4j

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v6/neo4j"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
)

const (
	defaultBatchSize = 500

	deleteProjectionRelationshipsQuery = `
MATCH ()-[r]->()
WHERE r.projection_namespace = $namespace
DELETE r`
	deleteProjectionNodesQuery = `
MATCH (n:TidewiseEntity {projection_namespace: $namespace})
DETACH DELETE n`
	deleteProjectionMetadataQuery = `
MATCH (m:TidewiseProjection {projection_namespace: $namespace})
DELETE m`
	readProjectionMetadataQuery = `
OPTIONAL MATCH (m:TidewiseProjection {projection_namespace: $namespace})
RETURN m.projection_contract_version AS contract_version,
       m.source_package_sha256 AS package_sha256,
       m.node_count AS node_count,
       m.relationship_count AS relationship_count,
       m.node_fingerprint AS node_fingerprint,
       m.relationship_fingerprint AS relationship_fingerprint,
       m.projection_namespace IS NOT NULL AS metadata_exists
LIMIT 1`
	readProjectionNodesQuery = `
MATCH (n:TidewiseEntity {projection_namespace: $namespace})
RETURN n.entity_id AS entity_id,
       n.entity_key AS entity_key,
       n.entity_type AS entity_type,
       n.canonical_name AS canonical_name,
       coalesce(n.aliases, []) AS aliases,
       n.aliases IS NOT NULL AS has_aliases,
       labels(n) AS labels,
       n.status AS status,
       n.projection_namespace AS projection_namespace,
       n.projection_contract_version AS projection_contract_version,
       n.source_package_sha256 AS source_package_sha256
ORDER BY entity_id, entity_key`
	readProjectionRelationshipsQuery = `
MATCH (from:TidewiseEntity {projection_namespace: $namespace})
      -[r]->
      (to:TidewiseEntity {projection_namespace: $namespace})
WHERE r.projection_namespace = $namespace
RETURN from.entity_id AS from_entity_id,
       to.entity_id AS to_entity_id,
       type(r) AS relationship_type,
       r.chain_id AS chain_id,
       r.relation_key AS relation_key,
       r.contextual_stage AS contextual_stage,
       r.position AS position,
       r.mechanism AS mechanism,
       r.status AS status,
       r.projection_namespace AS projection_namespace,
       r.projection_contract_version AS projection_contract_version,
       r.source_package_sha256 AS source_package_sha256
ORDER BY relation_key, relationship_type, from_entity_id, to_entity_id`
	readProjectionRelationshipIntegrityQuery = `
MATCH (from)-[r]->(to)
WHERE from.projection_namespace = $namespace
   OR to.projection_namespace = $namespace
   OR r.projection_namespace = $namespace
WITH from, r, to
WHERE NOT (
  from:TidewiseEntity
  AND to:TidewiseEntity
  AND from.projection_namespace = $namespace
  AND to.projection_namespace = $namespace
  AND r.projection_namespace = $namespace
)
RETURN count(*) AS violation_count`
	readProjectionNodeIntegrityQuery = `
MATCH (n)
WHERE n.projection_namespace = $namespace
  AND NOT n:TidewiseEntity
  AND NOT n:TidewiseProjection
RETURN count(*) AS violation_count`
	readProjectionMetadataIntegrityQuery = `
MATCH (m:TidewiseProjection {projection_namespace: $namespace})
WITH count(m) AS amount
RETURN CASE WHEN amount <= 1 THEN 0 ELSE amount - 1 END AS violation_count`
	readProjectionSchemaQuery = `
SHOW CONSTRAINTS
YIELD name, type, entityType, labelsOrTypes, properties
WHERE name IN $names
RETURN name, type, entityType, labelsOrTypes, properties
ORDER BY name`
	createProjectionMetadataQuery = `
CREATE (:TidewiseProjection {
  projection_namespace: $namespace,
  projection_contract_version: $contract_version,
  source_package_sha256: $package_sha256,
  node_count: $node_count,
  relationship_count: $relationship_count,
  node_fingerprint: $node_fingerprint,
  relationship_fingerprint: $relationship_fingerprint,
  projected_at: datetime()
})`
)

var entityTypeOrder = []biz.EntityType{
	biz.EntityTypeIndustry,
	biz.EntityTypeConcept,
	biz.EntityTypeIndustryChain,
	biz.EntityTypeChainNode,
}

var relationshipTypeOrder = []biz.RelationshipType{
	biz.RelationshipTypeMappedToIndustry,
	biz.RelationshipTypeMappedToConcept,
	biz.RelationshipTypeHasNode,
	biz.RelationshipTypeInputTo,
	biz.RelationshipTypeIsComponentOf,
	biz.RelationshipTypeDependsOn,
	biz.RelationshipTypeIsSubcategoryOf,
}

type Config struct {
	URI       string
	Username  string
	Password  string
	Database  string
	BatchSize int
}

type Store struct {
	driver    driver.Driver
	database  string
	batchSize int
}

type constraintDefinition struct {
	Name        string
	EntityType  string
	LabelOrType string
	Properties  []string
	Query       string
}

type cypherRunner interface {
	Run(context.Context, string, map[string]any) (driver.Result, error)
}

type autoCommitRunner struct {
	session driver.Session
}

func (r autoCommitRunner) Run(
	ctx context.Context,
	query string,
	params map[string]any,
) (driver.Result, error) {
	return r.session.Run(ctx, query, params)
}

func Open(ctx context.Context, config Config) (*Store, error) {
	if strings.TrimSpace(config.URI) == "" ||
		strings.TrimSpace(config.Username) == "" ||
		config.Password == "" ||
		strings.TrimSpace(config.Database) == "" {
		return nil, errors.New("local Neo4j URI, username, password and database are required")
	}
	value, err := driver.NewDriver(
		config.URI,
		driver.BasicAuth(config.Username, config.Password, ""),
	)
	if err != nil {
		return nil, errors.New("configure local Neo4j driver failed")
	}
	if err := value.VerifyConnectivity(ctx); err != nil {
		_ = value.Close(ctx)
		return nil, errors.New("verify local Neo4j connectivity failed")
	}
	batchSize := config.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	return &Store{driver: value, database: config.Database, batchSize: batchSize}, nil
}

func (s *Store) Close(ctx context.Context) error {
	if s == nil || s.driver == nil {
		return nil
	}
	if err := s.driver.Close(ctx); err != nil {
		return errors.New("close local Neo4j driver failed")
	}
	return nil
}

func (s *Store) InspectIndustryGraph(
	ctx context.Context,
	namespace string,
) (biz.ProjectionState, error) {
	if err := s.validate(namespace); err != nil {
		return biz.ProjectionState{}, err
	}
	session := s.driver.NewSession(ctx, driver.SessionConfig{
		AccessMode:   driver.AccessModeRead,
		DatabaseName: s.database,
		FetchSize:    1000,
	})
	defer func() { _ = session.Close(ctx) }()
	schemaMatchesContract, err := schemaMatches(ctx, autoCommitRunner{session: session})
	if err != nil {
		return biz.ProjectionState{}, errors.New("inspect local Neo4j Industry graph schema failed")
	}
	state, err := driver.ExecuteRead(
		ctx,
		session,
		func(tx driver.ManagedTransaction) (biz.ProjectionState, error) {
			return readProjection(ctx, tx, namespace)
		},
	)
	if err != nil {
		return biz.ProjectionState{}, errors.New("inspect local Neo4j Industry graph failed")
	}
	if !schemaMatchesContract {
		state.IntegrityViolationCount++
	}
	return state, nil
}

func (s *Store) ReplaceIndustryGraph(
	ctx context.Context,
	namespace string,
	projection biz.Projection,
) (biz.ProjectionState, error) {
	if err := s.validate(namespace); err != nil {
		return biz.ProjectionState{}, err
	}
	if err := biz.ValidateProjection(projection); err != nil {
		return biz.ProjectionState{}, fmt.Errorf("validate Industry graph before Neo4j replacement: %w", err)
	}
	if err := s.ensureSchema(ctx); err != nil {
		return biz.ProjectionState{}, err
	}

	session := s.driver.NewSession(ctx, driver.SessionConfig{
		AccessMode:   driver.AccessModeWrite,
		DatabaseName: s.database,
		FetchSize:    1000,
	})
	defer func() { _ = session.Close(ctx) }()
	tx, err := session.BeginTransaction(ctx, func(config *driver.TransactionConfig) {
		config.Timeout = 10 * time.Minute
		config.Metadata = map[string]any{
			"operation": "industry-graph-replace",
			"namespace": namespace,
		}
	})
	if err != nil {
		return biz.ProjectionState{}, errors.New("begin local Neo4j Industry graph transaction failed")
	}
	defer func() {
		cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = tx.Close(cleanupContext)
	}()

	params := map[string]any{"namespace": namespace}
	if err := runAndConsume(ctx, tx, deleteProjectionMetadataQuery, params); err != nil {
		return biz.ProjectionState{}, errors.New("replace local Neo4j Industry graph failed")
	}
	if err := runAndConsume(ctx, tx, deleteProjectionRelationshipsQuery, params); err != nil {
		return biz.ProjectionState{}, errors.New("replace local Neo4j Industry graph failed")
	}
	if err := runAndConsume(ctx, tx, deleteProjectionNodesQuery, params); err != nil {
		return biz.ProjectionState{}, errors.New("replace local Neo4j Industry graph failed")
	}
	nodeQueries := nodeCreateQueries()
	for _, entityType := range entityTypeOrder {
		rows := projectionNodeRows(projection, entityType)
		if err := writeBatches(ctx, tx, nodeQueries[entityType], namespace, rows, s.batchSize); err != nil {
			return biz.ProjectionState{}, errors.New("replace local Neo4j Industry graph failed")
		}
	}
	relationshipQueries := relationshipCreateQueries()
	for _, relationshipType := range relationshipTypeOrder {
		rows := projectionRelationshipRows(projection, relationshipType)
		if err := writeBatches(
			ctx,
			tx,
			relationshipQueries[relationshipType],
			namespace,
			rows,
			s.batchSize,
		); err != nil {
			return biz.ProjectionState{}, errors.New("replace local Neo4j Industry graph failed")
		}
	}
	summary := biz.SummarizeProjection(projection)
	if err := runAndConsume(ctx, tx, createProjectionMetadataQuery, map[string]any{
		"namespace":                namespace,
		"contract_version":         biz.ContractVersion,
		"package_sha256":           projection.PackageSHA256,
		"node_count":               summary.NodeCount,
		"relationship_count":       summary.RelationshipCount,
		"node_fingerprint":         summary.NodeFingerprint,
		"relationship_fingerprint": summary.RelationshipFingerprint,
	}); err != nil {
		return biz.ProjectionState{}, errors.New("replace local Neo4j Industry graph failed")
	}

	state, err := readProjection(ctx, tx, namespace)
	if err != nil {
		return biz.ProjectionState{}, errors.New("verify local Neo4j Industry graph transaction failed")
	}
	if state.ContractVersion != biz.ContractVersion ||
		state.PackageSHA256 != projection.PackageSHA256 ||
		state.IntegrityViolationCount != 0 ||
		!biz.ProjectionsEqual(state.Projection, projection) {
		actualSummary := biz.SummarizeProjection(state.Projection)
		return biz.ProjectionState{}, fmt.Errorf(
			"local Neo4j Industry graph transaction verification failed: contract_match=%t package_match=%t integrity_violations=%d nodes=%d/%d relationships=%d/%d node_fingerprint_match=%t relationship_fingerprint_match=%t",
			state.ContractVersion == biz.ContractVersion,
			state.PackageSHA256 == projection.PackageSHA256,
			state.IntegrityViolationCount,
			actualSummary.NodeCount,
			summary.NodeCount,
			actualSummary.RelationshipCount,
			summary.RelationshipCount,
			actualSummary.NodeFingerprint == summary.NodeFingerprint,
			actualSummary.RelationshipFingerprint == summary.RelationshipFingerprint,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		return biz.ProjectionState{}, errors.New("commit local Neo4j Industry graph transaction failed")
	}
	return state, nil
}

func (s *Store) validate(namespace string) error {
	if s == nil || s.driver == nil {
		return errors.New("local Neo4j Industry graph store is not configured")
	}
	if namespace != biz.Namespace {
		return errors.New("unsupported Neo4j Industry graph namespace")
	}
	if strings.TrimSpace(s.database) == "" {
		return errors.New("local Neo4j database is required")
	}
	return nil
}

func (s *Store) ensureSchema(ctx context.Context) error {
	session := s.driver.NewSession(ctx, driver.SessionConfig{
		AccessMode:   driver.AccessModeWrite,
		DatabaseName: s.database,
	})
	defer func() { _ = session.Close(ctx) }()
	runner := autoCommitRunner{session: session}
	for _, definition := range constraintDefinitions() {
		if err := runAndConsume(ctx, runner, definition.Query, nil); err != nil {
			return errors.New("ensure local Neo4j Industry graph schema failed")
		}
	}
	if err := verifySchema(ctx, runner); err != nil {
		return errors.New("verify local Neo4j Industry graph schema failed")
	}
	return nil
}

func constraintDefinitions() []constraintDefinition {
	definitions := []constraintDefinition{
		{
			Name: "tidewise_industry_entity_identity", EntityType: "NODE",
			LabelOrType: "TidewiseEntity",
			Properties:  []string{"projection_namespace", "entity_id"},
			Query: `CREATE CONSTRAINT tidewise_industry_entity_identity IF NOT EXISTS
FOR (n:TidewiseEntity)
REQUIRE (n.projection_namespace, n.entity_id) IS UNIQUE`,
		},
		{
			Name: "tidewise_industry_projection_namespace", EntityType: "NODE",
			LabelOrType: "TidewiseProjection",
			Properties:  []string{"projection_namespace"},
			Query: `CREATE CONSTRAINT tidewise_industry_projection_namespace IF NOT EXISTS
FOR (m:TidewiseProjection)
REQUIRE m.projection_namespace IS UNIQUE`,
		},
	}
	names := map[biz.RelationshipType]string{
		biz.RelationshipTypeMappedToIndustry: "mapped_to_industry",
		biz.RelationshipTypeMappedToConcept:  "mapped_to_concept",
		biz.RelationshipTypeHasNode:          "has_node",
		biz.RelationshipTypeInputTo:          "input_to",
		biz.RelationshipTypeIsComponentOf:    "is_component_of",
		biz.RelationshipTypeDependsOn:        "depends_on",
		biz.RelationshipTypeIsSubcategoryOf:  "is_subcategory_of",
	}
	for _, relationshipType := range relationshipTypeOrder {
		name := "tidewise_industry_" + names[relationshipType] + "_identity"
		definitions = append(definitions, constraintDefinition{
			Name: name, EntityType: "RELATIONSHIP",
			LabelOrType: string(relationshipType),
			Properties:  []string{"projection_namespace", "relation_key"},
			Query: fmt.Sprintf(
				`CREATE CONSTRAINT %s IF NOT EXISTS
FOR ()-[r:%s]-()
REQUIRE (r.projection_namespace, r.relation_key) IS UNIQUE`,
				name,
				relationshipType,
			),
		})
	}
	return definitions
}

func verifySchema(ctx context.Context, runner cypherRunner) error {
	matches, err := schemaMatches(ctx, runner)
	if err != nil {
		return err
	}
	if !matches {
		return errors.New("Neo4j Industry graph constraints differ from the contract")
	}
	return nil
}

func schemaMatches(ctx context.Context, runner cypherRunner) (bool, error) {
	definitions := constraintDefinitions()
	names := make([]string, 0, len(definitions))
	expected := make(map[string]constraintDefinition, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.Name)
		expected[definition.Name] = definition
	}
	rows, err := collectRows(ctx, runner, readProjectionSchemaQuery, map[string]any{"names": names})
	if err != nil {
		return false, err
	}
	if len(rows) != len(definitions) {
		return false, nil
	}
	for _, row := range rows {
		name, okName := requiredString(row["name"])
		constraintType, okType := requiredString(row["type"])
		entityType, okEntityType := requiredString(row["entityType"])
		labelsOrTypes, okLabels := stringSlice(row["labelsOrTypes"])
		properties, okProperties := stringSlice(row["properties"])
		definition, exists := expected[name]
		if !okName || !okType || !okEntityType || !okLabels || !okProperties || !exists {
			return false, nil
		}
		validUniquenessType := constraintType == "UNIQUENESS" ||
			(definition.EntityType == "RELATIONSHIP" && constraintType == "RELATIONSHIP_UNIQUENESS")
		if !validUniquenessType ||
			entityType != definition.EntityType ||
			len(labelsOrTypes) != 1 ||
			labelsOrTypes[0] != definition.LabelOrType ||
			!equalStrings(properties, definition.Properties) {
			return false, nil
		}
	}
	return true, nil
}

func nodeCreateQueries() map[biz.EntityType]string {
	return map[biz.EntityType]string{
		biz.EntityTypeIndustry:      createNodeQuery("Industry"),
		biz.EntityTypeConcept:       createNodeQuery("Concept"),
		biz.EntityTypeIndustryChain: createNodeQuery("IndustryChain"),
		biz.EntityTypeChainNode:     createNodeQuery("ChainNode"),
	}
}

func createNodeQuery(label string) string {
	return fmt.Sprintf(`
UNWIND $rows AS row
CREATE (n:TidewiseEntity:%s)
SET n = row
RETURN count(n) AS created`, label)
}

func relationshipCreateQueries() map[biz.RelationshipType]string {
	result := make(map[biz.RelationshipType]string, len(relationshipTypeOrder))
	for _, relationshipType := range relationshipTypeOrder {
		result[relationshipType] = createRelationshipQuery(relationshipType)
	}
	return result
}

func createRelationshipQuery(relationshipType biz.RelationshipType) string {
	return fmt.Sprintf(`
UNWIND $rows AS row
MATCH (from:TidewiseEntity {
  projection_namespace: $namespace,
  entity_id: row.from_entity_id
})
MATCH (to:TidewiseEntity {
  projection_namespace: $namespace,
  entity_id: row.to_entity_id
})
CREATE (from)-[r:%s]->(to)
SET r = row.properties
RETURN count(r) AS created`, relationshipType)
}

func projectionNodeRows(projection biz.Projection, entityType biz.EntityType) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, node := range projection.Nodes {
		if node.EntityType != entityType {
			continue
		}
		rows = append(rows, map[string]any{
			"entity_id":                   node.EntityID,
			"entity_key":                  node.EntityKey,
			"entity_type":                 string(node.EntityType),
			"canonical_name":              node.CanonicalName,
			"aliases":                     node.Aliases,
			"status":                      "active",
			"projection_namespace":        biz.Namespace,
			"projection_contract_version": biz.ContractVersion,
			"source_package_sha256":       projection.PackageSHA256,
		})
	}
	return rows
}

func projectionRelationshipRows(
	projection biz.Projection,
	relationshipType biz.RelationshipType,
) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, relationship := range projection.Relationships {
		if relationship.Type != relationshipType {
			continue
		}
		properties := map[string]any{
			"relation_key":                relationship.RelationKey,
			"mechanism":                   relationship.Mechanism,
			"status":                      "active",
			"projection_namespace":        biz.Namespace,
			"projection_contract_version": biz.ContractVersion,
			"source_package_sha256":       projection.PackageSHA256,
		}
		if relationship.ChainID != "" {
			properties["chain_id"] = relationship.ChainID
		}
		if relationship.ContextualStage != "" {
			properties["contextual_stage"] = relationship.ContextualStage
		}
		if relationship.Position != nil {
			properties["position"] = *relationship.Position
		}
		rows = append(rows, map[string]any{
			"from_entity_id": relationship.FromEntityID,
			"to_entity_id":   relationship.ToEntityID,
			"properties":     properties,
		})
	}
	return rows
}

func writeBatches(
	ctx context.Context,
	runner cypherRunner,
	query string,
	namespace string,
	rows []map[string]any,
	batchSize int,
) error {
	for start := 0; start < len(rows); start += batchSize {
		end := start + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		created, err := runAndCount(ctx, runner, query, map[string]any{
			"namespace": namespace,
			"rows":      rows[start:end],
		})
		if err != nil {
			return err
		}
		if created != end-start {
			return fmt.Errorf("created %d graph rows, want %d", created, end-start)
		}
	}
	return nil
}

func runAndCount(
	ctx context.Context,
	runner cypherRunner,
	query string,
	params map[string]any,
) (int, error) {
	result, err := runner.Run(ctx, query, params)
	if err != nil {
		return 0, err
	}
	record, err := result.Single(ctx)
	if err != nil {
		return 0, err
	}
	value, ok := record.Get("created")
	if !ok {
		return 0, errors.New("Neo4j write did not return created count")
	}
	count, ok := value.(int64)
	if !ok || count < 0 || int64(int(count)) != count {
		return 0, errors.New("Neo4j write returned an invalid created count")
	}
	return int(count), nil
}

func runAndConsume(
	ctx context.Context,
	runner cypherRunner,
	query string,
	params map[string]any,
) error {
	result, err := runner.Run(ctx, query, params)
	if err != nil {
		return err
	}
	_, err = result.Consume(ctx)
	return err
}

func readProjection(
	ctx context.Context,
	runner cypherRunner,
	namespace string,
) (biz.ProjectionState, error) {
	params := map[string]any{"namespace": namespace}
	metadata, err := collectRows(ctx, runner, readProjectionMetadataQuery, params)
	if err != nil {
		return biz.ProjectionState{}, err
	}
	var contractVersion, packageSHA, metadataNodeFingerprint, metadataRelationshipFingerprint string
	var metadataNodeCount, metadataRelationshipCount *int
	metadataExists := false
	if len(metadata) == 1 {
		contractVersion, _ = optionalString(metadata[0]["contract_version"])
		packageSHA, _ = optionalString(metadata[0]["package_sha256"])
		metadataNodeCount, _ = optionalInt(metadata[0]["node_count"])
		metadataRelationshipCount, _ = optionalInt(metadata[0]["relationship_count"])
		metadataNodeFingerprint, _ = optionalString(metadata[0]["node_fingerprint"])
		metadataRelationshipFingerprint, _ = optionalString(metadata[0]["relationship_fingerprint"])
		metadataExists, _ = optionalBool(metadata[0]["metadata_exists"])
	}
	integrityViolationCount := 0

	nodeRows, err := collectRows(ctx, runner, readProjectionNodesQuery, params)
	if err != nil {
		return biz.ProjectionState{}, err
	}
	nodes := make([]biz.Node, 0, len(nodeRows))
	for _, row := range nodeRows {
		entityID, okID := requiredString(row["entity_id"])
		entityKey, okKey := requiredString(row["entity_key"])
		entityType, okType := requiredString(row["entity_type"])
		canonicalName, okName := requiredString(row["canonical_name"])
		aliases, okAliases := stringSlice(row["aliases"])
		hasAliases, okHasAliases := optionalBool(row["has_aliases"])
		labels, okLabels := stringSlice(row["labels"])
		status, okStatus := optionalString(row["status"])
		projectionNamespace, okNamespace := optionalString(row["projection_namespace"])
		projectionContract, okContract := optionalString(row["projection_contract_version"])
		sourcePackageSHA, okSourceSHA := optionalString(row["source_package_sha256"])
		if !okID || !okKey || !okType || !okName || !okAliases || !okLabels || !okHasAliases {
			return biz.ProjectionState{}, errors.New("invalid Neo4j Industry graph node row")
		}
		if !okStatus || !okNamespace || !okContract || !okSourceSHA ||
			status != "active" ||
			projectionNamespace != namespace ||
			projectionContract != contractVersion ||
			sourcePackageSHA != packageSHA ||
			!hasAliases ||
			!validNodeLabels(biz.EntityType(entityType), labels) {
			integrityViolationCount++
		}
		nodes = append(nodes, biz.Node{
			EntityID: entityID, EntityKey: entityKey,
			EntityType: biz.EntityType(entityType), CanonicalName: canonicalName,
			Aliases: aliases,
		})
	}

	relationshipRows, err := collectRows(ctx, runner, readProjectionRelationshipsQuery, params)
	if err != nil {
		return biz.ProjectionState{}, err
	}
	relationships := make([]biz.Relationship, 0, len(relationshipRows))
	for _, row := range relationshipRows {
		fromEntityID, okFrom := requiredString(row["from_entity_id"])
		toEntityID, okTo := requiredString(row["to_entity_id"])
		relationshipType, okType := requiredString(row["relationship_type"])
		relationKey, okKey := requiredString(row["relation_key"])
		mechanism, okMechanism := requiredString(row["mechanism"])
		if !okFrom || !okTo || !okType || !okKey || !okMechanism {
			return biz.ProjectionState{}, errors.New("invalid Neo4j Industry graph relationship row")
		}
		chainID, _ := optionalString(row["chain_id"])
		contextualStage, _ := optionalString(row["contextual_stage"])
		status, okStatus := optionalString(row["status"])
		projectionNamespace, okNamespace := optionalString(row["projection_namespace"])
		projectionContract, okContract := optionalString(row["projection_contract_version"])
		sourcePackageSHA, okSourceSHA := optionalString(row["source_package_sha256"])
		position, okPosition := optionalInt(row["position"])
		if !okPosition {
			return biz.ProjectionState{}, errors.New("invalid Neo4j Industry graph relationship position")
		}
		if !okStatus || !okNamespace || !okContract || !okSourceSHA ||
			status != "active" ||
			projectionNamespace != namespace ||
			projectionContract != contractVersion ||
			sourcePackageSHA != packageSHA {
			integrityViolationCount++
		}
		relationships = append(relationships, biz.Relationship{
			FromEntityID: fromEntityID, ToEntityID: toEntityID,
			Type: biz.RelationshipType(relationshipType), ChainID: chainID,
			RelationKey: relationKey, ContextualStage: contextualStage,
			Position: position, Mechanism: mechanism,
		})
	}
	for _, query := range []string{
		readProjectionNodeIntegrityQuery,
		readProjectionRelationshipIntegrityQuery,
		readProjectionMetadataIntegrityQuery,
	} {
		violations, err := readViolationCount(ctx, runner, query, params)
		if err != nil {
			return biz.ProjectionState{}, err
		}
		integrityViolationCount += violations
	}
	projection := biz.Projection{
		PackageSHA256: packageSHA,
		Nodes:         nodes,
		Relationships: relationships,
	}
	summary := biz.SummarizeProjection(projection)
	if metadataExists {
		if metadataNodeCount == nil ||
			metadataRelationshipCount == nil ||
			*metadataNodeCount != summary.NodeCount ||
			*metadataRelationshipCount != summary.RelationshipCount ||
			metadataNodeFingerprint != summary.NodeFingerprint ||
			metadataRelationshipFingerprint != summary.RelationshipFingerprint {
			integrityViolationCount++
		}
	} else if len(nodes) != 0 || len(relationships) != 0 {
		integrityViolationCount++
	}
	return biz.ProjectionState{
		Projection: projection, ContractVersion: contractVersion, PackageSHA256: packageSHA,
		IntegrityViolationCount: integrityViolationCount,
	}, nil
}

func readViolationCount(
	ctx context.Context,
	runner cypherRunner,
	query string,
	params map[string]any,
) (int, error) {
	rows, err := collectRows(ctx, runner, query, params)
	if err != nil {
		return 0, err
	}
	if len(rows) != 1 {
		return 0, errors.New("Neo4j integrity query returned an invalid row count")
	}
	value, ok := rows[0]["violation_count"].(int64)
	if !ok || value < 0 || int64(int(value)) != value {
		return 0, errors.New("Neo4j integrity query returned an invalid count")
	}
	return int(value), nil
}

func collectRows(
	ctx context.Context,
	runner cypherRunner,
	query string,
	params map[string]any,
) ([]map[string]any, error) {
	result, err := runner.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}
	records, err := result.Collect(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(records))
	for _, record := range records {
		rows = append(rows, record.AsMap())
	}
	return rows, nil
}

func requiredString(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok && strings.TrimSpace(text) != ""
}

func optionalString(value any) (string, bool) {
	if value == nil {
		return "", true
	}
	text, ok := value.(string)
	return text, ok
}

func stringSlice(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return typed, true
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			result = append(result, text)
		}
		return result, true
	default:
		return nil, false
	}
}

func validNodeLabels(entityType biz.EntityType, labels []string) bool {
	expectedSpecific := map[biz.EntityType]string{
		biz.EntityTypeIndustry:      "Industry",
		biz.EntityTypeConcept:       "Concept",
		biz.EntityTypeIndustryChain: "IndustryChain",
		biz.EntityTypeChainNode:     "ChainNode",
	}[entityType]
	if expectedSpecific == "" || len(labels) != 2 {
		return false
	}
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		seen[label] = struct{}{}
	}
	_, base := seen["TidewiseEntity"]
	_, specific := seen[expectedSpecific]
	return base && specific
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func optionalInt(value any) (*int, bool) {
	if value == nil {
		return nil, true
	}
	switch typed := value.(type) {
	case int:
		return &typed, true
	case int64:
		result := int(typed)
		return &result, int64(result) == typed
	default:
		return nil, false
	}
}

func optionalBool(value any) (bool, bool) {
	if value == nil {
		return false, true
	}
	result, ok := value.(bool)
	return result, ok
}
