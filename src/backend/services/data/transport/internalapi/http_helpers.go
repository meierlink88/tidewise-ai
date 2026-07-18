package internalapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/rawimport"
)

func decodeStrictLimited(response http.ResponseWriter, request *http.Request, target any) error {
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func writeDecodeError(response http.ResponseWriter, requestID string, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || strings.Contains(err.Error(), "request body too large") {
		writeError(response, requestID, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes")
		return
	}
	writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract")
}

func writeRawImportError(response http.ResponseWriter, requestID string, err error) {
	switch rawimport.ErrorCode(err) {
	case rawimport.CodeIdempotencyConflict, rawimport.CodeIdentityConflict, rawimport.CodeBatchCollision:
		writeError(response, requestID, http.StatusConflict, rawimport.ErrorCode(err), err.Error())
	case rawimport.CodeInvalidRequest:
		writeError(response, requestID, http.StatusUnprocessableEntity, rawimport.ErrorCode(err), err.Error())
	default:
		writeError(response, requestID, http.StatusInternalServerError, "RAW_DOCUMENT_IMPORT_FAILED", "raw document import failed")
	}
}

func pageQuery(response http.ResponseWriter, request *http.Request, requestID string) (int, int, bool) {
	page, ok := optionalInt(response, requestID, request.URL.Query().Get("page"), 1, 1, 1_000_000, "page")
	if !ok {
		return 0, 0, false
	}
	pageSize, ok := optionalInt(response, requestID, request.URL.Query().Get("page_size"), 50, 1, 100, "page_size")
	return page, pageSize, ok
}

func optionalInt(response http.ResponseWriter, requestID, raw string, fallback, minimum, maximum int, name string) (int, bool) {
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("%s must be between %d and %d", name, minimum, maximum))
		return 0, false
	}
	return value, true
}

func optionalUTC(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	_, offset := value.Zone()
	if offset != 0 {
		return nil, fmt.Errorf("timestamp is not UTC")
	}
	value = value.UTC()
	return &value, nil
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func writeEnvelope(response http.ResponseWriter, status int, requestID string, result any) {
	writeJSON(response, status, map[string]any{"request_id": requestID, "result": result})
}

func writeError(response http.ResponseWriter, requestID string, status int, code, message string) {
	writeJSON(response, status, map[string]any{"request_id": requestID, "error": map[string]any{"code": code, "message": message, "details": map[string]any{}}})
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
