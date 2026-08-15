package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	regiondata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/region"
)

func main() {
	catalogPath := flag.String("file", "/app/initdata/regions-v1.json", "path to the Region initialization package")
	flag.Parse()

	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load database operation config: %v", err)
	}
	catalog, err := regiondata.LoadCatalog(*catalogPath)
	if err != nil {
		log.Fatalf("load Region catalog: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := regiondata.PublishCatalog(ctx, db, catalog); err != nil {
		log.Fatalf("publish Region catalog: %v", err)
	}
	log.Printf("published Region catalog: %d UN M49 sub-regions", len(catalog.Regions))
}
