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

func TestListEvidenceCategoriesMapsExactCatalogDTO(t *testing.T) {
	description := "简短报告已经发生或正在发生的事件，核心目的是说明发生了什么。"
	service, err := NewService(failingUseCase{catalog: evidencebiz.CategoryCatalog{Categories: []evidencebiz.Category{{
		ID: "EVCc18ddddb-14bc-5496-99ea-963ee2c25597", Code: "EVENT_BRIEF", Name: "事件快讯", Description: description,
	}}}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.ListEvidenceCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != v1.StatusOK || len(response.Result.Categories) != 1 {
		t.Fatalf("response = %#v", response)
	}
	category := response.Result.Categories[0]
	if category.ID != "EVCc18ddddb-14bc-5496-99ea-963ee2c25597" || category.Code != "EVENT_BRIEF" || category.Name != "事件快讯" || category.Description != description {
		t.Fatalf("category = %#v", category)
	}
}

func TestListEvidenceCategoriesMapsSafeFailures(t *testing.T) {
	for _, test := range []struct {
		name       string
		useCaseErr error
		wantStatus int
		wantCode   string
		wantError  error
	}{
		{name: "catalog failure", useCaseErr: errors.New("persisted invariant"), wantStatus: v1.StatusInternalServerError, wantCode: evidenceapi.ErrorEvidenceCategoryCatalogFailed},
		{name: "execution deadline", useCaseErr: context.DeadlineExceeded, wantStatus: v1.StatusServiceUnavailable, wantCode: evidenceapi.ErrorEvidenceCategoryCatalogTimeout},
		{name: "caller cancellation", useCaseErr: context.Canceled, wantError: context.Canceled},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewService(failingUseCase{err: test.useCaseErr})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.ListEvidenceCategories(context.Background())
			if test.wantError != nil {
				if !errors.Is(err, test.wantError) {
					t.Fatalf("error=%v, want %v", err, test.wantError)
				}
				return
			}
			var public *v1.PublicError
			if !errors.As(err, &public) || public.Status != test.wantStatus || public.Code != test.wantCode {
				t.Fatalf("error=%#v, want status/code %d/%s", err, test.wantStatus, test.wantCode)
			}
		})
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

type failingUseCase struct {
	err     error
	catalog evidencebiz.CategoryCatalog
}

func (u failingUseCase) ListCategories(context.Context) (evidencebiz.CategoryCatalog, error) {
	return u.catalog, u.err
}

func (failingUseCase) PublishRawEvidence(context.Context, evidencebiz.RawEvidence) (evidencebiz.RawEvidenceResult, error) {
	return evidencebiz.RawEvidenceResult{}, context.Canceled
}

func (u failingUseCase) GetRawEvidence(context.Context, string) (evidencebiz.StoredRawEvidence, error) {
	return evidencebiz.StoredRawEvidence{}, u.err
}

func (u failingUseCase) PublishEvidence(context.Context, string, []evidencebiz.Evidence) (evidencebiz.EvidenceResult, error) {
	return evidencebiz.EvidenceResult{}, u.err
}
