package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	geopoliticrivalrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/geopoliticrivalry"
)

func main() {
	catalogPath := flag.String("file", "/app/initdata/geopolitical-storylines-v1.json", "path to the geopolitical domain and storyline initialization package")
	flag.Parse()

	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load database operation config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	catalog, err := geopoliticrivalrydata.LoadCatalog(ctx, *catalogPath)
	if err != nil {
		log.Fatalf("load geopolitical catalog: %v", err)
	}
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := geopoliticrivalrydata.PublishCatalog(ctx, db, catalog); err != nil {
		log.Fatalf("publish geopolitical catalog: %v", err)
	}
	log.Printf("published geopolitical catalog: %d domains, %d storylines", len(catalog.Domains), len(catalog.Storylines))
}
