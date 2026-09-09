// Report storage is an explicit, operator-owned data cutover, separate from Goose DDL.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	reportdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/report"
)

func run() error {
	cutoff := flag.String("retain-from", "", "explicit RFC3339 published_at cutoff; empty retains all")
	planFile := flag.String("plan", "", "read (apply) or write (dry run) inventory JSON")
	apply := flag.Bool("apply", false, "apply frozen plan; default only inventories")
	backup := flag.String("backup-reference", "", "operator-verified recovery artifact identifier")
	verify := flag.Bool("verify", false, "read-only split-storage release gate")
	flag.Parse()
	if *verify && *apply {
		return fmt.Errorf("--verify and --apply are mutually exclusive")
	}
	if *planFile == "" && !*verify {
		return fmt.Errorf("--plan is required")
	}
	var from time.Time
	var err error
	if *cutoff != "" {
		from, err = time.Parse(time.RFC3339, *cutoff)
		if err != nil {
			return err
		}
	}
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	store, err := reportdata.NewStore(db)
	if err != nil {
		return err
	}
	if *verify {
		return store.VerifyStorage(ctx)
	}
	if *apply {
		raw, e := os.ReadFile(*planFile)
		if e != nil {
			return e
		}
		var plan reportdata.StoragePlan
		if e = json.Unmarshal(raw, &plan); e != nil {
			return e
		}
		if *cutoff != "" && !from.Equal(plan.RetainFrom) {
			return fmt.Errorf("cutoff differs from frozen plan")
		}
		if e = store.ApplyStorage(ctx, plan, *backup); e != nil {
			return e
		}
		fmt.Printf("Report storage applied: retained=%d deleted=%d\n", len(plan.Keep), len(plan.Delete))
		return nil
	}
	plan, err := store.PlanStorage(ctx, from)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(*planFile, append(raw, '\n'), 0600); err != nil {
		return err
	}
	fmt.Printf("Report storage plan: retain=%d delete=%d; no data changed\n", len(plan.Keep), len(plan.Delete))
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Report storage operation failed:", err)
		os.Exit(1)
	}
}
