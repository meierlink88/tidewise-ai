package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data"
	eventsemanticdata "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/data/eventsemantic"
)

func main() {
	output, err := parseOptions(os.Args[1:])
	if err != nil {
		fail(err.Error())
	}
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		fail("could not load Data database configuration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := data.OpenPostgres(ctx, cfg)
	if err != nil {
		fail("could not open Data database")
	}
	defer db.Close()
	manifest, err := eventsemanticdata.AuditHistoricalEventSemantics(ctx, db, time.Now())
	if err != nil {
		fail("could not audit historical Event Semantic inputs")
	}
	writer, closeWriter, err := outputWriter(output)
	if err != nil {
		fail(err.Error())
	}
	defer closeWriter()
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		fail("could not encode historical Event Semantic audit")
	}
}

func parseOptions(arguments []string) (string, error) {
	flags := flag.NewFlagSet("event-semantic-history-audit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	output := flags.String(
		"output", "",
		"new JSON manifest path; omit to write the read-only audit to stdout",
	)
	if err := flags.Parse(arguments); err != nil {
		return "", err
	}
	if flags.NArg() != 0 {
		return "", fmt.Errorf("unexpected positional arguments")
	}
	return *output, nil
}

func outputWriter(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return nil, func() {}, fmt.Errorf(
			"could not create audit manifest without overwriting an existing file",
		)
	}
	return file, func() { _ = file.Close() }, nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
