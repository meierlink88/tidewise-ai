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

func TestSourceErrorsUseStableTimeoutAndOperationSpecificCodes(t *testing.T) {
	for _, test := range []struct {
		name       string
		service    *fakeUseCase
		invoke     func(*Service) error
		wantStatus int
		wantCode   string
	}{
		{
			name: "snapshot deadline", service: &fakeUseCase{snapshotErr: context.DeadlineExceeded},
			invoke:     func(service *Service) error { _, err := service.Snapshot(context.Background()); return err },
			wantStatus: v1.StatusServiceUnavailable, wantCode: "SOURCE_TIMEOUT",
		},
		{
			name: "management persistence", service: &fakeUseCase{listErr: sourcebiz.ErrPersistence},
			invoke:     func(service *Service) error { _, err := service.List(context.Background()); return err },
			wantStatus: v1.StatusInternalServerError, wantCode: "SOURCE_FAILED",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(test.service)
			if err != nil {
				t.Fatal(err)
			}
			err = test.invoke(service)
			var public *v1.PublicError
			if !errors.As(err, &public) || public.Status != test.wantStatus || public.Code != test.wantCode {
				t.Fatalf("error = %#v, want %d %s", err, test.wantStatus, test.wantCode)
			}
		})
	}
}

type fakeUseCase struct {
	listErr     error
	snapshotErr error
}

func (f *fakeUseCase) CreateDynamic(context.Context, sourcebiz.MutableSource) (sourcebiz.Source, error) {
	return sourcebiz.Source{}, nil
}
func (f *fakeUseCase) List(context.Context) ([]sourcebiz.Source, error) { return nil, f.listErr }
func (f *fakeUseCase) Update(context.Context, string, sourcebiz.MutableSource) (sourcebiz.Source, error) {
	return sourcebiz.Source{}, nil
}
func (f *fakeUseCase) Delete(context.Context, string) error { return nil }
func (f *fakeUseCase) ActiveSnapshot(context.Context) ([]sourcebiz.Source, error) {
	return nil, f.snapshotErr
}
