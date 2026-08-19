package main

import (
	"context"
	"log"
	"os"
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
	manifest := sourcedata.CurrentFixedManifest(sourcedata.FixedManifestOptions{
		Endpoints: map[string]string{
			"bocha": os.Getenv("BOCHA_SEARCH_BASE_URL"), "tavily": os.Getenv("TAVILY_SEARCH_BASE_URL"),
			"parallel_search": os.Getenv("PARALLEL_SEARCH_BASE_URL"), "cls_telegraph": os.Getenv("CLS_TELEGRAPH_BASE_URL"),
			"eastmoney_fastnews": os.Getenv("EASTMONEY_FAST_NEWS_BASE_URL"), "eastmoney_stock_news": os.Getenv("EASTMONEY_STOCK_NEWS_BASE_URL"),
			"stcn_quicknews": os.Getenv("STCN_QUICK_NEWS_BASE_URL"),
		},
		AppKeys: map[string]string{
			"bocha": os.Getenv("BOCHA_API_KEY"), "tavily": os.Getenv("TAVILY_API_KEY"),
			"parallel_search": os.Getenv("PARALLEL_API_KEY"),
		},
	})
	result, err := useCase.PublishFixed(ctx, manifest)
	if err != nil {
		log.Fatalf("publish fixed Sources: %v", err)
	}
	log.Printf("published or verified %d fixed Sources", len(result))
}
