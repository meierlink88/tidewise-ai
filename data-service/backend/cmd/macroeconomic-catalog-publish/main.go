package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	macroeconomicdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/macroeconomic"
)

func main() {
	catalogPath := flag.String("file", "/app/initdata/macroeconomic-storylines-v1.json", "path to the macroeconomic domain and storyline initialization package")
	flag.Parse()

	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load database operation config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	catalog, err := macroeconomicdata.LoadCatalog(ctx, *catalogPath)
	if err != nil {
		log.Fatalf("load macroeconomic catalog: %v", err)
	}
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := macroeconomicdata.PublishCatalog(ctx, db, catalog); err != nil {
		log.Fatalf("publish macroeconomic catalog: %v", err)
	}
	log.Printf("published macroeconomic catalog: %d domains, %d storylines", len(catalog.Domains), len(catalog.Storylines))
}
