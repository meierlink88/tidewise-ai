package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/configuration"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/data"
	adapter "github.com/meierlink88/tidewise-ai/user-service/backend/internal/data/configuration"
)

func run(input io.Reader) error {
	// Parse and validate before opening a database; never accept secrets in flags.
	raw, err := io.ReadAll(io.LimitReader(input, 4097))
	if err != nil || len(raw) > 4096 {
		return biz.ErrUnavailable
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var credentials biz.Credentials
	if decoder.Decode(&credentials) != nil || decoder.Decode(new(any)) != io.EOF {
		return biz.ErrUnavailable
	}
	entry, err := biz.NewEntry(credentials, time.Now())
	if err != nil {
		return err
	}
	c, err := conf.Load("", false)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := data.Open(ctx, c.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = data.Ready(ctx, db); err != nil {
		return err
	}
	return adapter.New(db).Save(ctx, entry)
}
func main() {
	if run(os.Stdin) != nil {
		fmt.Fprintln(os.Stderr, "WeChat configuration import failed; verify input, User database and write permissions")
		os.Exit(1)
	}
	fmt.Println("WeChat configuration saved; restart User Service to apply")
}
