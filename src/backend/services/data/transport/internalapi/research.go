package internalapi

import (
	"errors"
	"net/http"

	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/research"
)

func (d Dependencies) listResearchThemes(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, limit, ok := researchListQuery(response, request, requestID)
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.ListThemes(request.Context(), research.ResearchListRequest{WindowHours: window, Limit: limit, Cursor: request.URL.Query().Get("cursor")})
	writeResearchResult(response, requestID, result, err)
}

func (d Dependencies) getResearchTheme(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, ok := optionalInt(response, requestID, request.URL.Query().Get("window_hours"), research.DefaultResearchWindowHours, research.MinResearchWindowHours, research.MaxResearchWindowHours, "window_hours")
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.GetTheme(request.Context(), request.PathValue("theme_id"), research.ResearchDetailRequest{WindowHours: window})
	if err != nil {
		writeResearchError(response, requestID, err)
		return
	}
	writeEnvelope(response, http.StatusOK, requestID, result)
}

func (d Dependencies) listResearchAnchors(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, limit, ok := researchListQuery(response, request, requestID)
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.ListAnchors(request.Context(), research.ResearchListRequest{WindowHours: window, Limit: limit, Cursor: request.URL.Query().Get("cursor")})
	writeResearchResult(response, requestID, result, err)
}

func (d Dependencies) getResearchAnchor(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	window, ok := optionalInt(response, requestID, request.URL.Query().Get("window_hours"), research.DefaultResearchWindowHours, research.MinResearchWindowHours, research.MaxResearchWindowHours, "window_hours")
	if !ok {
		return
	}
	if d.Research == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
		return
	}
	result, err := d.Research.GetAnchor(request.Context(), request.PathValue("anchor_id"), research.ResearchDetailRequest{WindowHours: window})
	if err != nil {
		writeResearchError(response, requestID, err)
		return
	}
	writeEnvelope(response, http.StatusOK, requestID, result)
}

func researchListQuery(response http.ResponseWriter, request *http.Request, requestID string) (int, int, bool) {
	window, ok := optionalInt(response, requestID, request.URL.Query().Get("window_hours"), research.DefaultResearchWindowHours, research.MinResearchWindowHours, research.MaxResearchWindowHours, "window_hours")
	if !ok {
		return 0, 0, false
	}
	limit, ok := optionalInt(response, requestID, request.URL.Query().Get("limit"), research.DefaultResearchLimit, 1, research.MaxResearchLimit, "limit")
	return window, limit, ok
}

func writeResearchResult(response http.ResponseWriter, requestID string, result any, err error) {
	if err != nil {
		writeResearchError(response, requestID, err)
		return
	}
	writeEnvelope(response, http.StatusOK, requestID, result)
}

func writeResearchError(response http.ResponseWriter, requestID string, err error) {
	switch {
	case errors.Is(err, research.ErrInvalidRequest):
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, research.ErrNotFound):
		writeError(response, requestID, http.StatusNotFound, "NOT_FOUND", "research aggregate was not found")
	default:
		writeError(response, requestID, http.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "research aggregate failed")
	}
}
