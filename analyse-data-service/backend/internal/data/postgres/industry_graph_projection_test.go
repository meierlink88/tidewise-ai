package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	biz "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industrygraphprojection"
	relationshipimport "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/industryrelationshipimport"
	graphdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/industrygraphprojection"
)

func TestIndustryGraphSnapshotReaderReadsApprovedEndpointClosure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	const packageSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const (
		parentIndustryID = "11111111-1111-4111-8111-111111111111"
		childIndustryID  = "22222222-2222-4222-8222-222222222222"
		conceptID        = "33333333-3333-4333-8333-333333333333"
		chainID          = "44444444-4444-4444-8444-444444444444"
		nodeAID          = "55555555-5555-4555-8555-555555555555"
		nodeBID          = "66666666-6666-4666-8666-666666666666"
		unrelatedID      = "77777777-7777-4777-8777-777777777777"
		globalRelationID = "88888888-8888-4888-8888-888888888888"
	)
	const (
		parentIndustryKey = "industry:test:parent"
		childIndustryKey  = "industry:test:child"
		conceptKey        = "concept:test"
		chainKey          = "industry_chain:test"
		nodeAKey          = "chain_node:test_a"
		nodeBKey          = "chain_node:test_b"
	)

	mock.ExpectBegin()
	mock.ExpectQuery(industryGraphReceiptQuery).
		WithArgs(packageSHA).
		WillReturnRows(sqlmock.NewRows([]string{"package_sha256", "approval_basis"}).
			AddRow(packageSHA, industryGraphReceiptApprovalBasis))
	mock.ExpectQuery(industryGraphNodesQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "entity_key", "entity_type", "canonical_name", "aliases",
		}).
			AddRow(parentIndustryID, parentIndustryKey, "industry", "父行业", `["父行业别名"]`).
			AddRow(childIndustryID, childIndustryKey, "industry", "子行业", `[]`).
			AddRow(conceptID, conceptKey, "concept", "测试概念", `[]`).
			AddRow(chainID, chainKey, "industry_chain", "测试产业链", `["链别名"]`).
			AddRow(nodeAID, nodeAKey, "chain_node", "节点A", `[]`).
			AddRow(nodeBID, nodeBKey, "chain_node", "节点B", `[]`).
			AddRow(unrelatedID, "concept:unrelated", "concept", "未映射概念", `[]`))
	mock.ExpectQuery(industryGraphIndustryHierarchyQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"from_id", "to_id", "from_key", "to_key",
		}).AddRow(childIndustryID, parentIndustryID, childIndustryKey, parentIndustryKey))
	mock.ExpectQuery(industryGraphMappingsQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"from_id", "to_id", "relation_type", "from_key", "to_key", "evidence_note",
		}).
			AddRow(chainID, childIndustryID, "mapped_to_industry", chainKey, childIndustryKey,
				"产业链范围覆盖测试子行业 evidence=artifact://industry").
			AddRow(chainID, conceptID, "mapped_to_concept", chainKey, conceptKey,
				"产业链路线覆盖测试概念 evidence=artifact://concept"))
	mock.ExpectQuery(industryGraphMembershipsQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"chain_id", "node_id", "chain_key", "node_key", "stage", "position", "reason",
		}).
			AddRow(chainID, nodeAID, chainKey, nodeAKey, "upstream", 1, "节点A提供投入。").
			AddRow(chainID, nodeBID, chainKey, nodeBKey, "downstream", 2, "节点B形成产出。"))
	mock.ExpectQuery(industryGraphChainEdgesQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"chain_id", "from_id", "to_id", "chain_key", "from_key",
			"relation_type", "to_key", "mechanism",
		}).
			AddRow(chainID, nodeAID, nodeBID, chainKey, nodeAKey, "input_to", nodeBKey,
				"节点A的产出进入节点B。").
			AddRow(chainID, nodeAID, nodeBID, chainKey, nodeAKey, "is_component_of", nodeBKey,
				"节点A构成节点B。").
			AddRow(chainID, nodeAID, nodeBID, chainKey, nodeAKey, "depends_on", nodeBKey,
				"节点A依赖节点B的能力。"))
	mock.ExpectQuery(industryGraphGlobalNodeHierarchyQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "from_id", "to_id", "mechanism",
		}).AddRow(globalRelationID, nodeAID, nodeBID, "节点A稳定属于节点B。"))
	mock.ExpectCommit()

	reader := NewIndustryGraphSnapshotReader(db)
	projection, err := reader.ReadIndustryGraphSnapshot(context.Background(), packageSHA)
	if err != nil {
		t.Fatalf("ReadIndustryGraphSnapshot() error = %v", err)
	}
	if projection.PackageSHA256 != packageSHA {
		t.Fatalf("package SHA = %q, want %q", projection.PackageSHA256, packageSHA)
	}
	if len(projection.Nodes) != 6 {
		t.Fatalf("node count = %d, want 6", len(projection.Nodes))
	}
	for _, node := range projection.Nodes {
		if node.EntityID == unrelatedID {
			t.Fatal("unrelated approved Concept must be excluded from the endpoint closure")
		}
		if node.Aliases == nil {
			t.Fatalf("node %q aliases are nil", node.EntityID)
		}
	}
	if len(projection.Relationships) != 9 {
		t.Fatalf("relationship count = %d, want 9", len(projection.Relationships))
	}
	if err := biz.ValidateProjection(projection); err != nil {
		t.Fatalf("ValidateProjection() error = %v", err)
	}

	mappedIndustry := relationshipByKey(
		t,
		projection.Relationships,
		chainKey+"|mapped_to_industry|"+childIndustryKey,
	)
	if mappedIndustry.Type != biz.RelationshipTypeMappedToIndustry {
		t.Fatalf("mapping type = %q", mappedIndustry.Type)
	}
	if mappedIndustry.ChainID != chainID {
		t.Fatalf("mapping chain_id = %q, want %q", mappedIndustry.ChainID, chainID)
	}
	if mappedIndustry.Mechanism != "产业链范围覆盖测试子行业" {
		t.Fatalf("mapping mechanism = %q", mappedIndustry.Mechanism)
	}

	membership := relationshipByKey(
		t,
		projection.Relationships,
		chainKey+"|has_node|"+nodeAKey,
	)
	if membership.Position == nil || *membership.Position != 1 {
		t.Fatalf("membership position = %v, want 1", membership.Position)
	}
	if membership.ContextualStage != "upstream" {
		t.Fatalf("membership contextual_stage = %q", membership.ContextualStage)
	}

	globalHierarchy := relationshipByKey(
		t,
		projection.Relationships,
		"chain_node_relation:"+globalRelationID,
	)
	if globalHierarchy.ChainID != "" {
		t.Fatalf("global hierarchy chain_id = %q, want empty", globalHierarchy.ChainID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIndustryGraphSnapshotReaderRequiresExactApprovedReceipt(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	const packageSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	mock.ExpectBegin()
	mock.ExpectQuery(industryGraphReceiptQuery).
		WithArgs(packageSHA).
		WillReturnRows(sqlmock.NewRows([]string{"package_sha256", "approval_basis"}))
	mock.ExpectRollback()

	_, err = NewIndustryGraphSnapshotReader(db).ReadIndustryGraphSnapshot(
		context.Background(),
		packageSHA,
	)
	if err == nil || !strings.Contains(err.Error(), "receipt") ||
		!strings.Contains(err.Error(), "was not found") {
		t.Fatalf("ReadIndustryGraphSnapshot() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIndustryGraphSnapshotReaderSanitizesSecretBearingDriverError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	const packageSHA = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	rawDriverError := errors.New(
		`dial postgres://tidewise:super-secret@localhost/tidewise_local failed while running SELECT * FROM credentials`,
	)
	mock.ExpectBegin()
	mock.ExpectQuery(industryGraphReceiptQuery).
		WithArgs(packageSHA).
		WillReturnError(rawDriverError)
	mock.ExpectRollback()

	_, err = NewIndustryGraphSnapshotReader(db).ReadIndustryGraphSnapshot(
		context.Background(),
		packageSHA,
	)
	if err == nil {
		t.Fatal("ReadIndustryGraphSnapshot() error = nil, want sanitized database error")
	}
	if !errors.Is(err, errIndustryGraphSnapshotDatabase) {
		t.Fatalf("ReadIndustryGraphSnapshot() error = %v, want stable database error", err)
	}
	for _, forbidden := range []string{
		"super-secret",
		"postgres://",
		"SELECT * FROM credentials",
		rawDriverError.Error(),
	} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("sanitized error %q exposes %q", err, forbidden)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestIndustryGraphSnapshotReaderRejectsInvalidPackageSHABeforeDatabaseAccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = NewIndustryGraphSnapshotReader(db).ReadIndustryGraphSnapshot(
		context.Background(),
		"NOT-A-SHA",
	)
	if err == nil || !strings.Contains(err.Error(), "64 lowercase hexadecimal") {
		t.Fatalf("ReadIndustryGraphSnapshot() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBeginIndustryGraphSnapshotTransactionUsesRepeatableReadOnly(t *testing.T) {
	connector := &industryGraphTxOptionsConnector{
		options: make(chan driver.TxOptions, 1),
	}
	db := sql.OpenDB(connector)
	t.Cleanup(func() {
		_ = db.Close()
	})

	tx, err := beginIndustryGraphSnapshotTransaction(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	options := <-connector.options
	if options.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) {
		t.Fatalf("isolation = %v, want repeatable read", options.Isolation)
	}
	if !options.ReadOnly {
		t.Fatal("snapshot transaction must be read-only")
	}
}

func TestIndustryMappingMechanismRemovesOnlyFrozenEvidenceSuffix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "frozen mapping evidence",
			input: "映射原因 evidence=artifact://a,artifact://b",
			want:  "映射原因",
		},
		{
			name:  "no evidence suffix",
			input: "映射原因",
			want:  "映射原因",
		},
		{
			name:  "empty reason is left for Biz validation",
			input: " evidence=artifact://a",
			want:  " evidence=artifact://a",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := industryMappingMechanism(test.input); got != test.want {
				t.Fatalf("industryMappingMechanism() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestIndustryGraphSnapshotReaderMatchesFrozenLocalProjection(t *testing.T) {
	databaseURL := os.Getenv("TIDEWISE_INDUSTRY_GRAPH_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TIDEWISE_INDUSTRY_GRAPH_TEST_DATABASE_URL to run the frozen local graph snapshot integration test")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	const packageSHA = "7c737410ac6af562af19f8b9dad9e8e1c802f8f782625bd360bb2e8f20768608"
	pkg, err := relationshipimport.LoadDirectory(
		"../../../data/industry_relationships/2026-07-27-v1",
		packageSHA,
	)
	if err != nil {
		t.Fatalf("LoadDirectory() error = %v", err)
	}
	baseline, err := graphdata.LoadFrozenV1CSVBaseline(pkg)
	if err != nil {
		t.Fatalf("LoadFrozenV1CSVBaseline() error = %v", err)
	}
	snapshot, err := NewIndustryGraphSnapshotReader(db).ReadIndustryGraphSnapshot(
		context.Background(),
		packageSHA,
	)
	if err != nil {
		t.Fatalf("ReadIndustryGraphSnapshot() error = %v", err)
	}
	if !reflect.DeepEqual(nodesByID(snapshot.Nodes), nodesByID(baseline.Nodes)) {
		t.Fatal("PostgreSQL Industry graph nodes differ from the frozen CSV baseline")
	}
	if !reflect.DeepEqual(
		relationshipsByKey(snapshot.Relationships),
		relationshipsByKey(baseline.Relationships),
	) {
		t.Fatal("PostgreSQL Industry graph relationships differ from the frozen CSV baseline")
	}
}

func relationshipByKey(
	t *testing.T,
	relationships []biz.Relationship,
	key string,
) biz.Relationship {
	t.Helper()
	for _, relationship := range relationships {
		if relationship.RelationKey == key {
			return relationship
		}
	}
	t.Fatalf("relationship %q not found", key)
	return biz.Relationship{}
}

func nodesByID(nodes []biz.Node) map[string]biz.Node {
	result := make(map[string]biz.Node, len(nodes))
	for _, node := range nodes {
		result[node.EntityID] = node
	}
	return result
}

func relationshipsByKey(relationships []biz.Relationship) map[string]biz.Relationship {
	result := make(map[string]biz.Relationship, len(relationships))
	for _, relationship := range relationships {
		result[relationship.RelationKey] = relationship
	}
	return result
}

type industryGraphTxOptionsConnector struct {
	options chan driver.TxOptions
}

func (c *industryGraphTxOptionsConnector) Connect(context.Context) (driver.Conn, error) {
	return &industryGraphTxOptionsConnection{options: c.options}, nil
}

func (*industryGraphTxOptionsConnector) Driver() driver.Driver {
	return industryGraphTxOptionsDriver{}
}

type industryGraphTxOptionsDriver struct{}

func (industryGraphTxOptionsDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("industry graph test driver requires Connector")
}

type industryGraphTxOptionsConnection struct {
	options chan driver.TxOptions
}

func (*industryGraphTxOptionsConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (*industryGraphTxOptionsConnection) Close() error {
	return nil
}

func (*industryGraphTxOptionsConnection) Begin() (driver.Tx, error) {
	return nil, errors.New("legacy begin is not supported")
}

func (c *industryGraphTxOptionsConnection) BeginTx(
	_ context.Context,
	options driver.TxOptions,
) (driver.Tx, error) {
	c.options <- options
	return industryGraphDriverTx{}, nil
}

type industryGraphDriverTx struct{}

func (industryGraphDriverTx) Commit() error {
	return nil
}

func (industryGraphDriverTx) Rollback() error {
	return nil
}

var _ driver.Connector = (*industryGraphTxOptionsConnector)(nil)
var _ driver.ConnBeginTx = (*industryGraphTxOptionsConnection)(nil)
var _ io.Closer = (*industryGraphTxOptionsConnection)(nil)
var _ biz.SnapshotReader = (*IndustryGraphSnapshotReader)(nil)
