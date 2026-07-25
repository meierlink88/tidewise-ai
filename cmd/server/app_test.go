package main

import (
	"context"
	"testing"
	"time"
)

func TestCloseWithinHonorsShutdownDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	blocked := make(chan struct{})
	if err := closeWithin(ctx, func() { <-blocked }); err == nil {
		t.Fatal("closeWithin accepted a cleanup that exceeded its deadline")
	}
	close(blocked)
}

func TestCloseWithinWaitsForResourceCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	closed := false
	if err := closeWithin(ctx, func() { closed = true }); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("resource cleanup did not run")
	}
}
