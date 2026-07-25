package data

import (
	"errors"
	"net/http"
	"testing"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
)

func TestThemeAdapterErrorsMapToStableBizErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "invalid request", err: &Error{Kind: ErrorKindClient, StatusCode: http.StatusBadRequest}, want: biz.ErrInvalidResearchRequest},
		{name: "not found", err: &Error{Kind: ErrorKindClient, StatusCode: http.StatusNotFound}, want: biz.ErrResearchNotFound},
		{name: "server", err: &Error{Kind: ErrorKindServer, StatusCode: http.StatusInternalServerError}, want: biz.ErrResearchDataService},
		{name: "network", err: &Error{Kind: ErrorKindConnection}, want: biz.ErrResearchDataService},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := mapThemeDataError(test.err); !errors.Is(got, test.want) {
				t.Fatalf("mapThemeDataError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}

func TestReasoningTreeAdapterErrorsMapByEndpointContract(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{
			name: "Theme missing",
			err:  &Error{Kind: ErrorKindClient, StatusCode: http.StatusNotFound, Code: "RESEARCH_THEME_NOT_FOUND"},
			want: biz.ErrResearchThemeNotFound,
		},
		{
			name: "trees missing",
			err:  &Error{Kind: ErrorKindClient, StatusCode: http.StatusNotFound, Code: "RESEARCH_REASONING_TREES_NOT_FOUND"},
			want: biz.ErrResearchReasoningTreesNotFound,
		},
		{
			name: "tree missing",
			err:  &Error{Kind: ErrorKindClient, StatusCode: http.StatusNotFound, Code: "RESEARCH_REASONING_TREE_NOT_FOUND"},
			want: biz.ErrResearchReasoningTreeNotFound,
		},
		{
			name: "unknown 404",
			err:  &Error{Kind: ErrorKindClient, StatusCode: http.StatusNotFound, Code: "UNEXPECTED_NOT_FOUND"},
			want: biz.ErrResearchDataUnavailable,
		},
		{
			name: "network",
			err:  &Error{Kind: ErrorKindConnection},
			want: biz.ErrResearchDataUnavailable,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := mapReasoningTreeDataError(test.err); !errors.Is(got, test.want) {
				t.Fatalf("mapReasoningTreeDataError(%v) = %v, want %v", test.err, got, test.want)
			}
		})
	}
}
