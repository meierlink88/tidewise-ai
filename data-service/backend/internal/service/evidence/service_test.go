package evidence

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
)

func TestNewServiceRejectsMissingUseCase(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil")
	}
}

func TestServiceClassifiesDeadlineAndPreservesCancellation(t *testing.T) {
	for _, test := range []struct {
		name       string
		useCaseErr error
		wantStatus int
		wantCode   string
		wantError  error
	}{
		{
			name:       "execution deadline",
			useCaseErr: context.DeadlineExceeded,
			wantStatus: v1.StatusServiceUnavailable,
			wantCode:   evidenceapi.ErrorEvidencePublicationTimeout,
		},
		{
			name:       "caller cancellation",
			useCaseErr: context.Canceled,
			wantError:  context.Canceled,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(failingUseCase{err: test.useCaseErr})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.PublishEvidence(context.Background(), &evidenceapi.EvidencePublicationRequest{})
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("PublishEvidence() error = %v, want %v", err, test.wantError)
				}
				return
			}
			var public *v1.PublicError
			if !errors.As(err, &public) {
				t.Fatalf("PublishEvidence() error = %T %v, want PublicError", err, err)
			}
			if public.Status != test.wantStatus || public.Code != test.wantCode {
				t.Fatalf("PublishEvidence() error = %#v", public)
			}
		})
	}
}

type failingUseCase struct{ err error }

func (failingUseCase) PublishRawEvidence(context.Context, evidencebiz.RawEvidence) (evidencebiz.RawEvidenceResult, error) {
	return evidencebiz.RawEvidenceResult{}, context.Canceled
}

func (u failingUseCase) GetRawEvidence(context.Context, string) (evidencebiz.StoredRawEvidence, error) {
	return evidencebiz.StoredRawEvidence{}, u.err
}

func (u failingUseCase) PublishEvidence(context.Context, string, []evidencebiz.Evidence) (evidencebiz.EvidenceResult, error) {
	return evidencebiz.EvidenceResult{}, u.err
}
