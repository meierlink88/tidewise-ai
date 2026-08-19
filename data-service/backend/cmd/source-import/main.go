package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	sourcedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/source"
)

func main() {
	path := flag.String("file", "", "path to a complete AgentOS Source export")
	flag.Parse()
	if *path == "" {
		log.Fatal("-file is required")
	}
	file, err := os.Open(*path)
	if err != nil {
		log.Fatalf("open Source import: %v", err)
	}
	defer file.Close()
	items, err := sourcedata.DecodeImport(file)
	if err != nil {
		log.Fatalf("read Source import: %v", err)
	}

	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	store, err := sourcedata.NewStore(db)
	if err != nil {
		log.Fatalf("create Source store: %v", err)
	}
	useCase, err := sourcebiz.NewUseCase(store)
	if err != nil {
		log.Fatalf("create Source use case: %v", err)
	}
	result, err := useCase.Import(ctx, items)
	if err != nil {
		log.Fatalf("import Sources: %v", err)
	}
	log.Printf("imported or verified %d Sources", len(result))
}
