package service

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchanalysiscontext"
)

type failingResearchAnalysisContextService struct {
	err error
}

func (s failingResearchAnalysisContextService) List(
	context.Context,
	researchanalysiscontext.Request,
) (researchanalysiscontext.Result, error) {
	return researchanalysiscontext.Result{}, s.err
}

func TestResearchAnalysisContextMapsHistoricalAndResourceFailures(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "historical semantics unavailable",
			err:        researchanalysiscontext.ErrHistoricalSemanticsUnavailable,
			wantStatus: v1.StatusUnprocessableEntity,
			wantCode:   "RESEARCH_ANALYSIS_CONTEXT_HISTORY_UNAVAILABLE",
		},
		{
			name:       "live reference closure is inconsistent",
			err:        researchanalysiscontext.ErrReferenceClosureInconsistent,
			wantStatus: v1.StatusConflict,
			wantCode:   "RESEARCH_ANALYSIS_CONTEXT_INCONSISTENT",
		},
		{
			name: "response budget exceeded",
			err: &researchanalysiscontext.ResourceLimitError{
				Reason:        "page exceeds budget",
				Component:     "analysis_context_page",
				ActualBytes:   int64Pointer(9 * 1024 * 1024),
				MaxBytes:      int64Pointer(8 * 1024 * 1024),
				RetryGuidance: "reduce_page_size",
			},
			wantStatus: v1.StatusTooManyRequests,
			wantCode:   "RESEARCH_ANALYSIS_CONTEXT_RESOURCE_LIMIT",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewDataService(Dependencies{
				ResearchAnalysisContext: failingResearchAnalysisContextService{err: test.err},
			})
			_, err := handler.ListResearchAnalysisContext(
				context.Background(),
				&v1.ResearchAnalysisContextRequest{},
			)
			var public *v1.PublicError
			if !errors.As(err, &public) ||
				public.Status != test.wantStatus ||
				public.Code != test.wantCode {
				t.Fatalf("error = %#v, want status=%d code=%q", err, test.wantStatus, test.wantCode)
			}
			if test.wantStatus == v1.StatusTooManyRequests {
				details, ok := public.Details.(v1.ResearchResourceLimitDetails)
				if !ok ||
					details.Component != "analysis_context_page" ||
					details.ActualBytes == nil ||
					*details.ActualBytes != 9*1024*1024 ||
					details.RetryGuidance != "reduce_page_size" {
					t.Fatalf("details = %#v", public.Details)
				}
			}
			if test.wantStatus == v1.StatusConflict {
				details, ok := public.Details.(v1.ResearchAnalysisContextInconsistentDetails)
				if !ok || details.RetryGuidance != "restart_from_first_page" {
					t.Fatalf("details = %#v", public.Details)
				}
			}
		})
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
