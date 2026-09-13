// Command storyline-domain-backfill publishes existing domain memberships between
// schema migrations 92 and 93. It never reads a seed catalog or rewrites storylines.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
)

func main() {
	apply := flag.Bool("apply", false, "commit the verified backfill; default rolls back a verification run")
	flag.Parse()
	if err := run(*apply); err != nil {
		log.Fatal(err)
	}
}

func run(apply bool) error {
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return errors.New("load Data operation configuration failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return errors.New("open Data database failed")
	}
	defer db.Close()
	result, err := data.BackfillStorylineDomains(ctx, db, apply)
	if err != nil {
		return err
	}
	log.Printf("verified domain memberships: geopolitical=%d macroeconomic=%d committed=%t", result.Geopolitical, result.Macroeconomic, apply)
	return nil
}
