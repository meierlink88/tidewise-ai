package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	app "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/config"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/internal/platform/database"
	"github.com/meierlink88/tidewise-ai/backend/internal/repositories"
)

const (
	exitOK       = 0
	exitUsage    = 2
	exitRejected = 2
	exitConflict = 3
	exitDatabase = 4
	exitCLI      = 5
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("event-import", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("input", "", "reviewed outbox JSON file or directory")
	dryRun := flags.Bool("dry-run", false, "validate only; do not connect to PostgreSQL")
	machine := flags.Bool("json", true, "emit one machine-readable JSON result")
	if err := flags.Parse(args); err != nil || *input == "" {
		if err == nil {
			_, _ = fmt.Fprintln(stderr, "-input is required")
		}
		return exitUsage
	}
	packages, err := app.LoadPackages(*input)
	if err != nil {
		return emitFailure(stdout, *machine, exitRejected, err)
	}
	if *dryRun {
		report, err := app.DryRun(context.Background(), packages)
		if err != nil {
			return emitFailure(stdout, *machine, exitRejected, err)
		}
		return emit(stdout, *machine, map[string]any{"ok": true, "mode": "dry-run", "package_id": firstPackageID(report.PackageIDs), "payload_hash": firstHash(report.PayloadHashes), "result": report, "errors": []string{}})
	}

	cfg, err := config.Load()
	if err != nil {
		return emitFailure(stdout, *machine, exitCLI, err)
	}
	timeout := time.Duration(cfg.Database.ConnectTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	db, err := database.Open(ctx, cfg)
	if err != nil {
		return emitFailure(stdout, *machine, exitDatabase, err)
	}
	defer db.Close()
	service := app.NewService(repositories.NewPostgresRepository(db))
	results := make([]app.Result, 0, len(packages))
	for _, pkg := range packages {
		result, importErr := service.Import(ctx, pkg)
		if importErr != nil {
			code := exitDatabase
			if importErr == app.ErrIdempotencyConflict {
				code = exitConflict
			}
			return emitFailure(stdout, *machine, code, importErr)
		}
		results = append(results, result)
	}
	return emit(stdout, *machine, map[string]any{"ok": true, "mode": "import", "package_id": firstPackageID(packageIDs(packages)), "payload_hash": firstHash(resultHashes(packages)), "result": results, "errors": []string{}})
}

func packageIDs(packages []domainimport.Package) []string {
	ids := make([]string, 0, len(packages))
	for _, pkg := range packages {
		ids = append(ids, pkg.PackageID)
	}
	return ids
}

func resultHashes(packages []domainimport.Package) []string {
	hashes := make([]string, 0, len(packages))
	for _, pkg := range packages {
		hash, _ := pkg.CanonicalHash()
		hashes = append(hashes, hashDisplay(hash))
	}
	return hashes
}

func firstPackageID(ids []string) any {
	if len(ids) == 1 {
		return ids[0]
	}
	return ids
}

func firstHash(hashes []string) any {
	if len(hashes) == 1 {
		return hashDisplay(hashes[0])
	}
	return hashes
}

func hashDisplay(hash string) string {
	if len(hash) >= 7 && hash[:7] == "sha256:" {
		return hash
	}
	return "sha256:" + hash
}

func emit(stdout io.Writer, machine bool, value any) int {
	if machine {
		content, err := json.Marshal(value)
		if err != nil {
			return exitCLI
		}
		_, _ = fmt.Fprintln(stdout, string(content))
	} else {
		_, _ = fmt.Fprintln(stdout, "event import completed")
	}
	return exitOK
}

func emitFailure(stdout io.Writer, machine bool, code int, err error) int {
	if machine {
		content, marshalErr := json.Marshal(map[string]any{"ok": false, "error": map[string]any{"code": errorCode(code), "message": err.Error(), "details": []string{}}, "exit_code": code})
		if marshalErr == nil {
			_, _ = fmt.Fprintln(stdout, string(content))
		}
	} else {
		_, _ = fmt.Fprintln(stdout, "event import failed")
	}
	return code
}

func errorCode(exitCode int) string {
	switch exitCode {
	case exitRejected:
		return "invalid_input"
	case exitConflict:
		return "idempotency_conflict"
	case exitDatabase:
		return "database_failure"
	default:
		return "cli_failure"
	}
}
