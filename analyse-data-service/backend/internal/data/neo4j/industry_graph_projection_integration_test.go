package neo4j

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v6/neo4j"

	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
)

const neo4jIntegrationOptIn = "TIDEWISE_NEO4J_INTEGRATION_TEST"

func TestIndustryGraphProjectionNeo4jIntegration(t *testing.T) {
	config := neo4jIntegrationConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	store, err := Open(ctx, config)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		closeContext, closeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer closeCancel()
		if err := store.Close(closeContext); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	counts := readIntegrationDatabaseCounts(t, ctx, store)
	if counts.nodes != 0 || counts.relationships != 0 {
		t.Fatalf(
			"refusing to mutate non-empty Neo4j integration database: nodes=%d relationships=%d",
			counts.nodes,
			counts.relationships,
		)
	}
	t.Cleanup(func() {
		cleanupIntegrationNamespace(t, store)
	})

	if err := store.ensureSchema(ctx); err != nil {
		t.Fatalf("ensureSchema() error = %v", err)
	}

	first := integrationProjection(strings.Repeat("a", 64), true)
	firstState, err := store.ReplaceIndustryGraph(ctx, biz.Namespace, first)
	if err != nil {
		t.Fatalf("first ReplaceIndustryGraph() error = %v", err)
	}
	assertIntegrationProjectionState(t, firstState, first)
	inspectedFirst, err := store.InspectIndustryGraph(ctx, biz.Namespace)
	if err != nil {
		t.Fatalf("first InspectIndustryGraph() error = %v", err)
	}
	assertIntegrationProjectionState(t, inspectedFirst, first)

	second := integrationProjection(strings.Repeat("b", 64), false)
	secondState, err := store.ReplaceIndustryGraph(ctx, biz.Namespace, second)
	if err != nil {
		t.Fatalf("second ReplaceIndustryGraph() error = %v", err)
	}
	assertIntegrationProjectionState(t, secondState, second)
	inspectedSecond, err := store.InspectIndustryGraph(ctx, biz.Namespace)
	if err != nil {
		t.Fatalf("second InspectIndustryGraph() error = %v", err)
	}
	assertIntegrationProjectionState(t, inspectedSecond, second)
	for _, node := range inspectedSecond.Projection.Nodes {
		if node.EntityID == "node-stale" {
			t.Fatal("second replacement retained stale node")
		}
	}

	missingAliases := integrationProjection(strings.Repeat("c", 64), false)
	if err := biz.ValidateProjection(missingAliases); err != nil {
		t.Fatalf("missing-aliases fault input is not structurally valid: %v", err)
	}
	omission := &aliasesOmission{}
	faultyStore := &Store{
		driver: missingAliasesDriver{
			Driver:   store.driver,
			omission: omission,
		},
		database:  store.database,
		batchSize: store.batchSize,
	}
	_, err = faultyStore.ReplaceIndustryGraph(ctx, biz.Namespace, missingAliases)
	if err == nil || !strings.Contains(err.Error(), "integrity_violations=1") {
		t.Fatalf(
			"missing-aliases ReplaceIndustryGraph() error = %v, want one readback integrity violation",
			err,
		)
	}
	if !omission.applied.Load() {
		t.Fatal("missing-aliases fault was not applied")
	}

	afterRollback, err := store.InspectIndustryGraph(ctx, biz.Namespace)
	if err != nil {
		t.Fatalf("InspectIndustryGraph() after rollback error = %v", err)
	}
	assertIntegrationProjectionState(t, afterRollback, second)
}

type integrationDatabaseCounts struct {
	nodes         int64
	relationships int64
}

func neo4jIntegrationConfig(t *testing.T) Config {
	t.Helper()
	if os.Getenv(neo4jIntegrationOptIn) != "1" {
		t.Skip("set TIDEWISE_NEO4J_INTEGRATION_TEST=1 to run the Neo4j Adapter integration test")
	}
	values := map[string]string{
		"URI":      os.Getenv("TIDEWISE_TEST_NEO4J_URI"),
		"Username": os.Getenv("TIDEWISE_TEST_NEO4J_USERNAME"),
		"Password": os.Getenv("TIDEWISE_TEST_NEO4J_PASSWORD"),
		"Database": os.Getenv("TIDEWISE_TEST_NEO4J_DATABASE"),
	}
	for name, value := range values {
		if value == "" {
			t.Skipf("set TIDEWISE_TEST_NEO4J_%s to run the Neo4j Adapter integration test", strings.ToUpper(name))
		}
	}
	if strings.EqualFold(strings.TrimSpace(values["Database"]), "system") {
		t.Fatal("refusing to run the Neo4j Adapter integration test against the system database")
	}
	return Config{
		URI:       values["URI"],
		Username:  values["Username"],
		Password:  values["Password"],
		Database:  values["Database"],
		BatchSize: 2,
	}
}

func readIntegrationDatabaseCounts(
	t *testing.T,
	ctx context.Context,
	store *Store,
) integrationDatabaseCounts {
	t.Helper()
	session := store.driver.NewSession(ctx, driver.SessionConfig{
		AccessMode:   driver.AccessModeRead,
		DatabaseName: store.database,
	})
	defer func() {
		if err := session.Close(ctx); err != nil {
			t.Errorf("close Neo4j preflight session: %v", err)
		}
	}()
	counts, err := driver.ExecuteRead(
		ctx,
		session,
		func(tx driver.ManagedTransaction) (integrationDatabaseCounts, error) {
			nodes, err := integrationCount(ctx, tx, "MATCH (n) RETURN count(n) AS amount", nil)
			if err != nil {
				return integrationDatabaseCounts{}, err
			}
			relationships, err := integrationCount(
				ctx,
				tx,
				"MATCH ()-[r]->() RETURN count(r) AS amount",
				nil,
			)
			if err != nil {
				return integrationDatabaseCounts{}, err
			}
			return integrationDatabaseCounts{
				nodes:         nodes,
				relationships: relationships,
			}, nil
		},
	)
	if err != nil {
		t.Fatalf("read Neo4j integration database counts: %v", err)
	}
	return counts
}

func integrationCount(
	ctx context.Context,
	runner cypherRunner,
	query string,
	params map[string]any,
) (int64, error) {
	result, err := runner.Run(ctx, query, params)
	if err != nil {
		return 0, err
	}
	record, err := result.Single(ctx)
	if err != nil {
		return 0, err
	}
	value, ok := record.Get("amount")
	if !ok {
		return 0, errMissingIntegrationCount
	}
	amount, ok := value.(int64)
	if !ok || amount < 0 {
		return 0, errInvalidIntegrationCount
	}
	return amount, nil
}

var (
	errMissingIntegrationCount = errors.New("Neo4j count result is missing")
	errInvalidIntegrationCount = errors.New("Neo4j count result is invalid")
)

func cleanupIntegrationNamespace(t *testing.T, store *Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	session := store.driver.NewSession(ctx, driver.SessionConfig{
		AccessMode:   driver.AccessModeWrite,
		DatabaseName: store.database,
	})
	defer func() {
		if err := session.Close(ctx); err != nil {
			t.Errorf("close Neo4j cleanup session: %v", err)
		}
	}()
	runner := autoCommitRunner{session: session}
	params := map[string]any{"namespace": biz.Namespace}
	for _, query := range []string{
		deleteProjectionRelationshipsQuery,
		deleteProjectionNodesQuery,
		deleteProjectionMetadataQuery,
	} {
		if err := runAndConsume(ctx, runner, query, params); err != nil {
			t.Errorf("clean Neo4j integration namespace: %v", err)
			return
		}
	}
	nodes, err := integrationCount(
		ctx,
		runner,
		"MATCH (n {projection_namespace: $namespace}) RETURN count(n) AS amount",
		params,
	)
	if err != nil {
		t.Errorf("verify cleaned Neo4j integration nodes: %v", err)
		return
	}
	relationships, err := integrationCount(
		ctx,
		runner,
		"MATCH ()-[r]->() WHERE r.projection_namespace = $namespace RETURN count(r) AS amount",
		params,
	)
	if err != nil {
		t.Errorf("verify cleaned Neo4j integration relationships: %v", err)
		return
	}
	if nodes != 0 || relationships != 0 {
		t.Errorf(
			"Neo4j integration namespace cleanup left nodes=%d relationships=%d",
			nodes,
			relationships,
		)
	}
}

func integrationProjection(packageSHA string, includeStale bool) biz.Projection {
	positionOne := 1
	positionTwo := 2
	nodes := []biz.Node{
		{
			EntityID: "chain", EntityKey: "industry_chain:integration",
			EntityType: biz.EntityTypeIndustryChain, CanonicalName: "集成测试产业链",
			Aliases: []string{},
		},
		{
			EntityID: "node-a", EntityKey: "chain_node:integration_a",
			EntityType: biz.EntityTypeChainNode, CanonicalName: "集成测试节点甲",
			Aliases: []string{},
		},
		{
			EntityID: "node-b", EntityKey: "chain_node:integration_b",
			EntityType: biz.EntityTypeChainNode, CanonicalName: "集成测试节点乙",
			Aliases: []string{},
		},
	}
	relationships := []biz.Relationship{
		{
			FromEntityID: "chain", ToEntityID: "node-a",
			Type: biz.RelationshipTypeHasNode, ChainID: "chain",
			RelationKey:     "industry_chain:integration|has_node|chain_node:integration_a",
			ContextualStage: "upstream", Position: &positionOne,
			Mechanism: "节点甲属于测试产业链上游",
		},
		{
			FromEntityID: "chain", ToEntityID: "node-b",
			Type: biz.RelationshipTypeHasNode, ChainID: "chain",
			RelationKey:     "industry_chain:integration|has_node|chain_node:integration_b",
			ContextualStage: "downstream", Position: &positionTwo,
			Mechanism: "节点乙属于测试产业链下游",
		},
		{
			FromEntityID: "node-a", ToEntityID: "node-b",
			Type: biz.RelationshipTypeInputTo, ChainID: "chain",
			RelationKey: "industry_chain:integration|chain_node:integration_a|input_to|chain_node:integration_b",
			Mechanism:   "节点甲的产出进入节点乙",
		},
	}
	if includeStale {
		positionThree := 3
		nodes = append(nodes, biz.Node{
			EntityID: "node-stale", EntityKey: "chain_node:integration_stale",
			EntityType: biz.EntityTypeChainNode, CanonicalName: "待删除测试节点",
			Aliases: []string{},
		})
		relationships[1].ContextualStage = "midstream"
		relationships = append(relationships,
			biz.Relationship{
				FromEntityID: "chain", ToEntityID: "node-stale",
				Type: biz.RelationshipTypeHasNode, ChainID: "chain",
				RelationKey:     "industry_chain:integration|has_node|chain_node:integration_stale",
				ContextualStage: "downstream", Position: &positionThree,
				Mechanism: "待删除节点属于测试产业链下游",
			},
			biz.Relationship{
				FromEntityID: "node-b", ToEntityID: "node-stale",
				Type: biz.RelationshipTypeInputTo, ChainID: "chain",
				RelationKey: "industry_chain:integration|chain_node:integration_b|input_to|chain_node:integration_stale",
				Mechanism:   "节点乙的产出进入待删除节点",
			},
		)
	}
	return biz.Projection{
		PackageSHA256: packageSHA,
		Nodes:         nodes,
		Relationships: relationships,
	}
}

func assertIntegrationProjectionState(
	t *testing.T,
	state biz.ProjectionState,
	expected biz.Projection,
) {
	t.Helper()
	if state.ContractVersion != biz.ContractVersion ||
		state.PackageSHA256 != expected.PackageSHA256 ||
		state.IntegrityViolationCount != 0 ||
		!biz.ProjectionsEqual(state.Projection, expected) {
		t.Fatalf("projection state = %#v, want exact valid projection", state)
	}
}

type aliasesOmission struct {
	applied atomic.Bool
}

type missingAliasesDriver struct {
	driver.Driver
	omission *aliasesOmission
}

func (d missingAliasesDriver) NewSession(
	ctx context.Context,
	config driver.SessionConfig,
) driver.Session {
	return missingAliasesSession{
		Session:  d.Driver.NewSession(ctx, config),
		omission: d.omission,
	}
}

type missingAliasesSession struct {
	driver.Session
	omission *aliasesOmission
}

func (s missingAliasesSession) BeginTransaction(
	ctx context.Context,
	configurers ...func(*driver.TransactionConfig),
) (driver.ExplicitTransaction, error) {
	tx, err := s.Session.BeginTransaction(ctx, configurers...)
	if err != nil {
		return nil, err
	}
	return missingAliasesTransaction{
		ExplicitTransaction: tx,
		omission:            s.omission,
	}, nil
}

type missingAliasesTransaction struct {
	driver.ExplicitTransaction
	omission *aliasesOmission
}

func (tx missingAliasesTransaction) Run(
	ctx context.Context,
	query string,
	params map[string]any,
) (driver.Result, error) {
	if strings.Contains(query, "CREATE (n:TidewiseEntity:") &&
		tx.omission.applied.CompareAndSwap(false, true) {
		params = cloneParamsWithoutFirstAliases(params)
	}
	return tx.ExplicitTransaction.Run(ctx, query, params)
}

func cloneParamsWithoutFirstAliases(params map[string]any) map[string]any {
	clonedParams := make(map[string]any, len(params))
	for key, value := range params {
		clonedParams[key] = value
	}
	rows, ok := params["rows"].([]map[string]any)
	if !ok || len(rows) == 0 {
		return clonedParams
	}
	clonedRows := append([]map[string]any(nil), rows...)
	first := make(map[string]any, len(rows[0]))
	for key, value := range rows[0] {
		if key != "aliases" {
			first[key] = value
		}
	}
	clonedRows[0] = first
	clonedParams["rows"] = clonedRows
	return clonedParams
}
