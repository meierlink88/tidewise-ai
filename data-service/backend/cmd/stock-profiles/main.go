package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	stockdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/stock"
)

func run() error {
	file := flag.String("file", "", "provided v4 profile JSON")
	check := flag.Bool("check-only", false, "validate without database")
	flag.Parse()
	if *file == "" {
		return biz.ErrInvalid
	}
	f, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer f.Close()
	if *check {
		batch, err := biz.DecodeProfiles(f)
		if err != nil {
			return err
		}
		fmt.Printf("validated %d profiles\n", len(batch.Items))
		return nil
	}
	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		return err
	}
	defer db.Close()
	repo, err := stockdata.NewStore(db)
	if err != nil {
		return err
	}
	u, err := biz.NewUseCase(repo)
	if err != nil {
		return err
	}
	count, err := u.InitializeProfiles(ctx, f)
	if err == nil {
		fmt.Printf("published or verified %d profiles\n", count)
	}
	return err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "stock initialization failed; verify catalog, target and database readiness")
		os.Exit(1)
	}
}
