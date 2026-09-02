package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/dbmigration"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
)

const preV74EvidenceRecoveryConfirmation = "issue-374-uat-pre-v74-evidence-clear"

type cliOptions struct {
	apply bool
}

func main() {
	options, err := parseCLIOptions(os.Args[1:])
	if err != nil {
		log.Fatalf("parse pre-v74 Evidence recovery options: %v", err)
	}
	if err := validateApplyConfirmation(options, os.Getenv); err != nil {
		log.Fatalf("validate pre-v74 Evidence recovery options: %v", err)
	}

	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := data.OpenPostgres(context.Background(), cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	locker := dbmigration.NewPostgresAdvisoryLocker(db, cfg.Migration.LockKey)
	if err := locker.Lock(context.Background()); err != nil {
		log.Fatalf("lock Data migrations: %v", err)
	}
	report, recoveryErr := evidence.RecoverPreV74Evidence(context.Background(), db, options.apply)
	unlockErr := locker.Unlock(context.Background())
	if err := errors.Join(recoveryErr, unlockErr); err != nil {
		log.Fatalf("run pre-v74 Evidence recovery: %v", err)
	}
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("encode pre-v74 Evidence recovery report: %v", err)
	}
	fmt.Fprintln(os.Stdout, string(content))
}

func parseCLIOptions(args []string) (cliOptions, error) {
	flags := flag.NewFlagSet("pre-v74-evidence-recovery", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	apply := flags.Bool("apply", false, "clear the incompatible pre-v74 Evidence dataset")
	if err := flags.Parse(args); err != nil {
		return cliOptions{}, err
	}
	if flags.NArg() != 0 {
		return cliOptions{}, fmt.Errorf("unexpected positional arguments")
	}
	return cliOptions{apply: *apply}, nil
}

func validateApplyConfirmation(options cliOptions, getenv func(string) string) error {
	if !options.apply {
		return nil
	}
	if getenv("TIDEWISE_PRE_V74_EVIDENCE_RECOVERY_CONFIRMED") != preV74EvidenceRecoveryConfirmation {
		return fmt.Errorf("pre-v74 Evidence recovery confirmation is missing")
	}
	return nil
}
