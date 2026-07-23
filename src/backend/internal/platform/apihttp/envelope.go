// Package apihttp provides business-free HTTP API response primitives.
package apihttp

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

const RequestIDHeader = "X-Request-ID"

type SuccessEnvelope struct {
	RequestID string `json:"request_id"`
	Result    any    `json:"result"`
}

type ErrorEnvelope struct {
	Error     ErrorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

// ResolveRequestID preserves a valid caller request ID and otherwise creates one.
func ResolveRequestID(value, prefix string) string {
	if requestID := strings.TrimSpace(value); requestID != "" && len(requestID) <= 128 {
		return requestID
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err == nil {
		return prefix + "-" + hex.EncodeToString(random[:])
	}
	return prefix + "-generated"
}

func Success(requestID string, result any) SuccessEnvelope {
	return SuccessEnvelope{RequestID: requestID, Result: result}
}

func Error(requestID, code, message string, details any) ErrorEnvelope {
	if details == nil {
		details = map[string]any{}
	}
	return ErrorEnvelope{
		Error:     ErrorDetail{Code: code, Message: message, Details: details},
		RequestID: requestID,
	}
}
