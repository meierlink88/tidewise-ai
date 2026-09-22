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
	file := flag.String("file", "", "stock catalog JSON path (required)")
	check := flag.Bool("check-only", false, "validate without connecting to database")
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
		items, err := biz.DecodeCatalog(f)
		if err == nil {
			fmt.Printf("validated %d stocks\n", len(items))
		}
		return err
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
	count, err := u.Initialize(ctx, f)
	if err == nil {
		fmt.Printf("initialized or verified %d stocks\n", count)
	}
	return err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "stock initialization failed; verify catalog, target and database readiness")
		os.Exit(1)
	}
}
