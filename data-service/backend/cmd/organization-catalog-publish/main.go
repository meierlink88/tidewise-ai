package main

import (
	"context"
	"log"
	"time"

	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	organizationdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/organization"
)

func main() {
	config, err := conf.Load()
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
	catalog, err := organizationbiz.AssignCatalogIdentities(organizationdata.CurrentCatalog())
	if err != nil {
		log.Fatalf("assign Organization catalog identities: %v", err)
	}
	if err := organizationdata.PublishCatalog(ctx, db, catalog); err != nil {
		log.Fatalf("publish Organization catalog: %v", err)
	}
	log.Printf("published Organization catalog: %d categories, %d functions, %d domain tags", len(catalog.Categories), len(catalog.Functions), len(catalog.DomainTags))
}
