package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/data"
	"github.com/pressly/goose/v3"
)

func run() error {
	directory := flag.String("dir", "user-service/backend/migrations", "migration ledger directory")
	flag.Parse()
	c, err := conf.Load("", false)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := data.Open(ctx, c.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(*directory))
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
func main() {
	if run() != nil {
		fmt.Fprintln(os.Stderr, "user migration failed; verify target, credentials and ledger")
		os.Exit(1)
	}
	fmt.Println("user migration complete")
}
