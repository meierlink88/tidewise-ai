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
		status: http.StatusBadRequest, code: "INVALID_REQUEST", message: "invalid request",
	}
	ErrReportNotFound = &HTTPError{
		status: http.StatusNotFound, code: "REPORT_NOT_FOUND", message: "report not found",
	}
	ErrReportLayerNotFound = &HTTPError{
		status: http.StatusNotFound, code: "REPORT_LAYER_NOT_FOUND", message: "report layer not found",
	}
	ErrReportIndustryChainNotFound = &HTTPError{
		status: http.StatusNotFound, code: "REPORT_INDUSTRY_CHAIN_NOT_FOUND", message: "report industry chain not found",
	}
	ErrReportEvidenceScopeNotFound = &HTTPError{
		status: http.StatusNotFound, code: "REPORT_EVIDENCE_SCOPE_NOT_FOUND", message: "report evidence scope not found",
	}
	ErrReportServiceUnavailable = &HTTPError{
		status: http.StatusServiceUnavailable, code: "REPORT_SERVICE_UNAVAILABLE", message: "report service unavailable",
	}
)

func PublicError(err error) (*HTTPError, bool) {
	var public *HTTPError
	if !errors.As(err, &public) {
		return nil, false
	}
	return public, true
}
