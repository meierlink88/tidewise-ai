package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	file := flag.String("file", "", "complete supplemental stock JSON")
	expected := flag.String("sha256", "", "required source SHA-256")
	apply := flag.Bool("apply", false, "apply locally; default validates and rolls back")
	flag.Parse()
	if *file == "" || len(*expected) != 64 {
		return fmt.Errorf("file and sha256 required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != *expected {
		return fmt.Errorf("source checksum mismatch")
	}
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return err
	}
	if cfg.App.Env != conf.EnvLocal || cfg.Database.Name != "tidewise_local" || (cfg.Database.Host != "127.0.0.1" && cfg.Database.Host != "localhost" && cfg.Database.Host != "postgres" && cfg.Database.Host != "host.docker.internal") {
		return fmt.Errorf("classification publication only supports the local tidewise_local database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	repo, err := stockdata.NewStore(db)
	if err != nil {
		return err
	}
	result, err := biz.PublishClassifications(ctx, repo, bytes.NewReader(raw), *apply)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		SHA256 string `json:"source_sha256"`
		biz.ClassificationResult
	}{*expected, result})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
