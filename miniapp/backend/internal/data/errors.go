package data

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz"
)

type ErrorKind string

const (
	ErrorKindClient     ErrorKind = "client"
	ErrorKindConflict   ErrorKind = "conflict"
	ErrorKindServer     ErrorKind = "server"
	ErrorKindConnection ErrorKind = "connection"
	ErrorKindTimeout    ErrorKind = "timeout"
	ErrorKindCanceled   ErrorKind = "canceled"
	ErrorKindProtocol   ErrorKind = "protocol"
	ErrorKindEncode     ErrorKind = "encode"
	ErrorKindDecode     ErrorKind = "decode"
)

// Error contains sanitized HTTP adapter diagnostics. It never crosses the Biz
// port: public repository methods map it to stable business errors first.
type Error struct {
	Kind       ErrorKind
	StatusCode int
	Code       string
	RequestID  string
}

func (e *Error) Error() string {
	if e == nil {
		return "data service request failed"
	}
	message := "data service request failed: kind=" + string(e.Kind)
	if e.StatusCode != 0 {
		message += " status=" + strconv.Itoa(e.StatusCode)
	}
	if code := safeMetadata(e.Code, 100); code != "" {
		message += " code=" + code
	}
	if requestID := safeMetadata(e.RequestID, 128); requestID != "" {
		message += " request_id=" + requestID
	}
	return message
}

func mapThemeDataError(err error) error {
	if err == nil {
		return nil
	}
	var adapterErr *Error
	if errors.As(err, &adapterErr) {
		switch adapterErr.StatusCode {
		case http.StatusBadRequest:
			return biz.ErrInvalidResearchRequest
		case http.StatusNotFound:
			return biz.ErrResearchNotFound
		}
	}
	return biz.ErrResearchDataService
}

func mapReasoningTreeDataError(err error) error {
	if err == nil {
		return nil
	}
	var adapterErr *Error
	if errors.As(err, &adapterErr) && adapterErr.StatusCode == http.StatusNotFound {
		switch adapterErr.Code {
		case "RESEARCH_THEME_NOT_FOUND":
			return biz.ErrResearchThemeNotFound
		case "RESEARCH_REASONING_TREES_NOT_FOUND":
			return biz.ErrResearchReasoningTreesNotFound
		case "RESEARCH_REASONING_TREE_NOT_FOUND":
			return biz.ErrResearchReasoningTreeNotFound
		}
	}
	return biz.ErrResearchDataUnavailable
}

func safeMetadata(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxLength {
		return ""
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-', character == '_', character == '.', character == ':':
		default:
			return ""
		}
	}
	return value
}
