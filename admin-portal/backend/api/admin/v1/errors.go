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

func NewHTTPError(status int, code, message string) *HTTPError {
	return &HTTPError{status: status, code: code, message: message}
}

func (e *HTTPError) Error() string   { return e.message }
func (e *HTTPError) Status() int     { return e.status }
func (e *HTTPError) Code() string    { return e.code }
func (e *HTTPError) Message() string { return e.message }

var ErrInvalidRequest = NewHTTPError(
	http.StatusBadRequest,
	"INVALID_REQUEST",
	"request is invalid",
)

func PublicError(err error) (*HTTPError, bool) {
	var public *HTTPError
	if !errors.As(err, &public) {
		return nil, false
	}
	return public, true
}
