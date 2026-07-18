package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/database"
	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/dbmigration"
	"github.com/meierlink88/tidewise-ai/backend/services/data/adapters/graphdb"
	"github.com/meierlink88/tidewise-ai/backend/services/data/config"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/graphprojection"
)

type executor interface {
	Check(context.Context) error
	Project(context.Context, repositories.GraphProjectionMode) (repositories.GraphProjectionRun, error)
}

func main() {
	ctx := context.Background()
	exec, closeFn, err := newActualExecutor(ctx)
	if err != nil {
		log.Fatalf("initialize graph projector: %v", err)
	}
	defer closeFn()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, exec))
}

func run(ctx context.Context, args []string, out io.Writer, exec executor) int {
	if len(args) != 1 {
		writeUsage(out)
		return 2
	}

	switch args[0] {
	case "check":
		if err := exec.Check(ctx); err != nil {
			fmt.Fprintf(out, "graph-projector check failed: %v\n", err)
			return 1
		}
		fmt.Fprintln(out, "neo4j connectivity ok")
		return 0
	case "project-entities":
		return runProjection(ctx, out, exec, repositories.GraphProjectionModeProjectEntities)
	case "rebuild-entities":
		return runProjection(ctx, out, exec, repositories.GraphProjectionModeRebuildEntities)
	default:
		writeUsage(out)
		return 2
	}
}

func runProjection(ctx context.Context, out io.Writer, exec executor, mode repositories.GraphProjectionMode) int {
	report, err := exec.Project(ctx, mode)
	if err != nil {
		fmt.Fprintf(out, "graph-projector %s failed: %v\n", mode, err)
		return 1
	}
	fmt.Fprintf(out, "run=%s status=%s source_rows=%d projected=%d skipped=%d failed=%d\n",
		report.ID,
		report.Status,
		report.SourceRowCount,
		report.ProjectedCount,
		report.SkippedCount,
		report.FailedCount,
	)
	return 0
}

func writeUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: graph-projector <check|project-entities|rebuild-entities>")
}

type actualExecutor struct {
	cfg    config.Config
	driver graphdb.Driver
}

func newActualExecutor(ctx context.Context) (*actualExecutor, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, func() {}, err
	}
	if _, err := dbmigration.CheckPostgres(ctx, cfg, cfg.Migration.AutoApply); err != nil {
		return nil, func() {}, err
	}
	driver, err := graphdb.Open(ctx, cfg.Neo4j, nil, nil)
	if err != nil {
		return nil, func() {}, err
	}
	if driver == nil {
		return nil, func() {}, fmt.Errorf("neo4j is disabled")
	}
	return &actualExecutor{cfg: cfg, driver: driver}, func() { _ = driver.Close(context.Background()) }, nil
}

func (e *actualExecutor) Check(context.Context) error {
	return nil
}

func (e *actualExecutor) Project(ctx context.Context, mode repositories.GraphProjectionMode) (repositories.GraphProjectionRun, error) {
	db, err := database.Open(ctx, e.cfg)
	if err != nil {
		return repositories.GraphProjectionRun{}, err
	}
	defer db.Close()

	repo := repositories.NewPostgresRepository(db)
	writer := graphprojection.NewNeo4jGraphWriter(e.driver, e.cfg.Neo4j.Database)
	projector := graphprojection.NewProjector(repo, writer, "tidewise", nil)
	return projector.ProjectEntities(ctx, graphprojection.ProjectOptions{Mode: mode})
}
