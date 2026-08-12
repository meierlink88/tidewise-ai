package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data"
)

const auditContractVersion = "uat-evidence-receipt-cleanup-audit.v1"

type protectedTableAudit struct {
	RowCount        int64             `json:"row_count"`
	Fingerprint     string            `json:"fingerprint"`
	RowFingerprints map[string]string `json:"row_fingerprints"`
}

type cleanupAuditReport struct {
	ContractVersion string                         `json:"contract_version"`
	Objects         map[string]bool                `json:"objects"`
	ProtectedTables map[string]protectedTableAudit `json:"protected_tables"`
}

func main() {
	if err := execute(os.Stdout); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func execute(output io.Writer) error {
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return fmt.Errorf("load Data configuration: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open PostgreSQL for Evidence receipt cleanup audit: %w", err)
	}
	defer database.Close()
	return run(ctx, database, output)
}

func run(ctx context.Context, database *sql.DB, output io.Writer) error {
	if database == nil || output == nil {
		return fmt.Errorf("Evidence receipt cleanup audit dependencies are required")
	}
	report := cleanupAuditReport{
		ContractVersion: auditContractVersion,
		Objects:         make(map[string]bool, 3),
		ProtectedTables: make(map[string]protectedTableAudit, 2),
	}
	objects := map[string]string{
		"raw_evidence_publication_receipts":             "SELECT to_regclass('public.raw_evidence_publication_receipts') IS NOT NULL",
		"evidence_publication_receipts":                 "SELECT to_regclass('public.evidence_publication_receipts') IS NOT NULL",
		"prevent_evidence_publication_receipt_mutation": "SELECT to_regprocedure('public.prevent_evidence_publication_receipt_mutation()') IS NOT NULL",
	}
	for name, query := range objects {
		var exists bool
		if err := database.QueryRowContext(ctx, query).Scan(&exists); err != nil {
			return fmt.Errorf("audit Evidence receipt object %s: %w", name, err)
		}
		report.Objects[name] = exists
	}
	for table, identityColumn := range map[string]string{"raw_evidences": "raw_evidence_id", "evidences": "evidence_id"} {
		value, err := auditProtectedTable(ctx, database, table, identityColumn)
		if err != nil {
			return fmt.Errorf("audit protected Evidence table %s: %w", table, err)
		}
		report.ProtectedTables[table] = value
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func auditProtectedTable(ctx context.Context, database *sql.DB, table, identityColumn string) (protectedTableAudit, error) {
	query := fmt.Sprintf("SELECT %s::text, md5(to_jsonb(source_row)::text) FROM %s source_row ORDER BY %s", identityColumn, table, identityColumn)
	rows, err := database.QueryContext(ctx, query)
	if err != nil {
		return protectedTableAudit{}, err
	}
	defer rows.Close()
	hash := sha256.New()
	rowFingerprints := make(map[string]string)
	var count int64
	for rows.Next() {
		var id, rowFingerprint string
		if err := rows.Scan(&id, &rowFingerprint); err != nil {
			return protectedTableAudit{}, err
		}
		if _, exists := rowFingerprints[id]; exists {
			return protectedTableAudit{}, fmt.Errorf("duplicate persisted identity %s", id)
		}
		if _, err := io.WriteString(hash, id+":"+rowFingerprint+"\n"); err != nil {
			return protectedTableAudit{}, err
		}
		rowFingerprints[id] = rowFingerprint
		count++
	}
	if err := rows.Err(); err != nil {
		return protectedTableAudit{}, err
	}
	return protectedTableAudit{RowCount: count, Fingerprint: hex.EncodeToString(hash.Sum(nil)), RowFingerprints: rowFingerprints}, nil
}
