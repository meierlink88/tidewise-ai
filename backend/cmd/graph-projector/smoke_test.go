package main

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"
)

func TestGraphProjectorNeo4jSmoke(t *testing.T) {
	if os.Getenv("TIDEWISE_ENABLE_NEO4J_SMOKE") != "true" {
		t.Skip("set TIDEWISE_ENABLE_NEO4J_SMOKE=true with local Neo4j credentials to run graph projector smoke")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exec, closeFn, err := newActualExecutor(ctx)
	if err != nil {
		t.Fatalf("newActualExecutor() error = %v", err)
	}
	defer closeFn()

	var out bytes.Buffer
	if code := run(ctx, []string{"check"}, &out, exec); code != 0 {
		t.Fatalf("graph-projector check code = %d output = %q", code, out.String())
	}
}
