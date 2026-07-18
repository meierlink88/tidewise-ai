package researchseed

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	called bool
	report Report
	err    error
}

func (f *fakeStore) Apply(context.Context, Manifest, time.Time) (Report, error) {
	f.called = true
	return f.report, f.err
}

func TestServiceValidatesBeforeApplying(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)
	if _, err := service.Apply(context.Background(), Manifest{}, time.Now()); err == nil {
		t.Fatal("Apply() error = nil, want validation error")
	}
	if store.called {
		t.Fatal("store was called for invalid manifest")
	}
}

func TestServiceReturnsStoreResult(t *testing.T) {
	want := Report{ThemeCount: 3, ChainNodeCount: 13, EventCount: 35}
	store := &fakeStore{report: want}
	service := NewService(store)
	manifest := validTestManifest()

	got, err := service.Apply(context.Background(), manifest, time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Apply() = %#v, want %#v", got, want)
	}

	store.err = errors.New("database unavailable")
	if _, err := service.Apply(context.Background(), manifest, time.Now()); err == nil {
		t.Fatal("Apply() store error = nil")
	}
}
