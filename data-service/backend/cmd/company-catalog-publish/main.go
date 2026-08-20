package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	companydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/company"
)

func main() {
	catalogPath := flag.String("file", "/app/initdata/companies-v2.json", "path to the Company initialization package")
	flag.Parse()

	config, err := conf.LoadDatabaseOperation()
	if err != nil {
		log.Fatalf("load database operation config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	catalog, err := companydata.LoadCatalog(ctx, *catalogPath)
	if err != nil {
		log.Fatalf("load Company catalog: %v", err)
	}
	db, err := data.OpenPostgres(ctx, config)
	if err != nil {
		log.Fatalf("open Data PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := companydata.PublishCatalog(ctx, db, catalog); err != nil {
		log.Fatalf("publish Company catalog: %v", err)
	}
	log.Printf("published Company catalog: %d companies", len(catalog.Companies))
}
