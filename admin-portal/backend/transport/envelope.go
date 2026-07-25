package transport

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

const requestIDHeader = "X-Request-ID"

type successEnvelope struct {
	RequestID string `json:"request_id"`
	Result    any    `json:"result"`
}

type errorEnvelope struct {
	Error     errorDetail `json:"error"`
	RequestID string      `json:"request_id"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

func resolveRequestID(value, prefix string) string {
	if requestID := strings.TrimSpace(value); requestID != "" && len(requestID) <= 128 {
		return requestID
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err == nil {
		return prefix + "-" + hex.EncodeToString(random[:])
	}
	return prefix + "-generated"
}

func successResponse(requestID string, result any) successEnvelope {
	return successEnvelope{RequestID: requestID, Result: result}
}

func errorResponse(requestID, code, message string, details any) errorEnvelope {
	if details == nil {
		details = map[string]any{}
	}
	return errorEnvelope{
		Error:     errorDetail{Code: code, Message: message, Details: details},
		RequestID: requestID,
	}
}
