package main

import (
	"context"
	"log"
	"time"

	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	sourcedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/source"
)

func main() {
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
	result, err := useCase.PublishDynamicCatalog(ctx, sourcebiz.CurrentResearchRSSManifest())
	if err != nil {
		log.Fatalf("publish research RSS Sources: %v", err)
	}
	log.Printf("published or verified %d research RSS Sources", len(result))
}
