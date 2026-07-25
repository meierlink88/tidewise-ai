package v1

import (
	"errors"
	"net/http"
)

type HTTPError struct {
	status  int
	code    string
	message string
}

func (e *HTTPError) Error() string {
	return e.message
}

func (e *HTTPError) Status() int {
	return e.status
}

func (e *HTTPError) Code() string {
	return e.code
}

func (e *HTTPError) Message() string {
	return e.message
}

var (
	ErrInvalidRequest = &HTTPError{
		status: http.StatusBadRequest, code: "INVALID_REQUEST", message: "invalid research request",
	}
	ErrResearchResultNotFound = &HTTPError{
		status: http.StatusNotFound, code: "RESEARCH_RESULT_NOT_FOUND", message: "research result not found",
	}
	ErrResearchThemeNotFound = &HTTPError{
		status: http.StatusNotFound, code: "RESEARCH_THEME_NOT_FOUND", message: "research Theme was not found",
	}
	ErrResearchReasoningTreesNotFound = &HTTPError{
		status: http.StatusNotFound, code: "RESEARCH_REASONING_TREES_NOT_FOUND", message: "research Theme has no published reasoning trees",
	}
	ErrResearchReasoningTreeNotFound = &HTTPError{
		status: http.StatusNotFound, code: "RESEARCH_REASONING_TREE_NOT_FOUND", message: "research reasoning tree was not found for the Theme",
	}
	ErrResearchDataFailure = &HTTPError{
		status: http.StatusInternalServerError, code: "RESEARCH_DATA_UNAVAILABLE", message: "research data service failure",
	}
	ErrResearchDataUnavailable = &HTTPError{
		status: http.StatusBadGateway, code: "RESEARCH_DATA_UNAVAILABLE", message: "research data is temporarily unavailable",
	}
)

func PublicError(err error) (*HTTPError, bool) {
	var public *HTTPError
	if !errors.As(err, &public) {
		return nil, false
	}
	return public, true
}
