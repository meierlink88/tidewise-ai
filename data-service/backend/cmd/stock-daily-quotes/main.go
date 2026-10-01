package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	stockdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/stock"
)

func run() error {
	file := flag.String("file", "", "source daily K-line JSON")
	expected := flag.String("sha256", "", "required source SHA-256")
	apply := flag.Bool("apply", false, "apply to local database; default checks without writing")
	flag.Parse()
	if *file == "" || len(*expected) != 64 {
		return fmt.Errorf("file and sha256 required")
	}
	f, err := os.Open(*file)
	if err != nil {
		return fmt.Errorf("cannot read source file")
	}
	raw, readErr := io.ReadAll(io.LimitReader(f, biz.MaxDailyQuoteFileBytes+1))
	closeErr := f.Close()
	if readErr != nil || closeErr != nil || len(raw) > biz.MaxDailyQuoteFileBytes {
		return fmt.Errorf("source read failed or exceeds 32 MiB")
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != *expected {
		return fmt.Errorf("source checksum mismatch")
	}
	batch, err := biz.DecodeDailyQuotes(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return err
	}
	if cfg.App.Env != conf.EnvLocal || cfg.Database.Name != "tidewise_local" || (cfg.Database.Host != "127.0.0.1" && cfg.Database.Host != "localhost" && cfg.Database.Host != "postgres" && cfg.Database.Host != "host.docker.internal") {
		return fmt.Errorf("daily quote publication only supports local tidewise_local")
	}
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, 5*time.Minute)
	defer cancel()
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		return fmt.Errorf("database connection failed")
	}
	defer db.Close()
	repo, err := stockdata.NewStore(db)
	if err != nil {
		return err
	}
	result, err := biz.PublishDailyQuotes(ctx, repo, batch, *apply)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		SHA256 string          `json:"source_sha256"`
		Meta   json.RawMessage `json:"source_meta"`
		biz.DailyQuoteResult
	}{*expected, batch.Meta, result})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
