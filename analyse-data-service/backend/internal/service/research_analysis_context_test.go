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
			name:       "response budget exceeded",
			err:        &researchanalysiscontext.ResourceLimitError{Reason: "page exceeds budget"},
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
		})
	}
}
