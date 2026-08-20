package main

import (
	"context"
	"log"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	storylinedomaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/storylinedomain"
)

func main() {
	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load database operation config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	catalog := storylinedomaindata.CurrentCatalog()
	if err := storylinedomaindata.PublishCatalog(ctx, db, catalog); err != nil {
		log.Fatalf("publish StorylineDomain catalog: %v", err)
	}
	log.Printf("published StorylineDomain catalog: %d domains", len(catalog))
}
