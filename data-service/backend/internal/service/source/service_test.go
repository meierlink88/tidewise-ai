package source

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
)

func TestSnapshotMapsIncompleteOrOversizedStateToFailClosedUnavailable(t *testing.T) {
	service, err := NewService(&fakeUseCase{snapshotErr: sourcebiz.ErrCapacityExceeded})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Snapshot(context.Background())
	var public *v1.PublicError
	if !errors.As(err, &public) || public.Status != v1.StatusServiceUnavailable || public.Code != "SOURCE_SNAPSHOT_FAILED" {
		t.Fatalf("Snapshot error = %#v, want fail-closed 503 SOURCE_SNAPSHOT_FAILED", err)
	}
}

type fakeUseCase struct{ snapshotErr error }

func (f *fakeUseCase) CreateDynamic(context.Context, sourcebiz.MutableSource) (sourcebiz.Source, error) {
	return sourcebiz.Source{}, nil
}
func (f *fakeUseCase) List(context.Context) ([]sourcebiz.Source, error) { return nil, nil }
func (f *fakeUseCase) Update(context.Context, string, sourcebiz.MutableSource) (sourcebiz.Source, error) {
	return sourcebiz.Source{}, nil
}
func (f *fakeUseCase) Delete(context.Context, string) error { return nil }
func (f *fakeUseCase) ActiveSnapshot(context.Context) ([]sourcebiz.Source, error) {
	return nil, f.snapshotErr
}
