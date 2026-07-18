package internalapi

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	domainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventimport"
	eventapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventimport"
	"github.com/meierlink88/tidewise-ai/backend/services/data/usecase/rawimport"
)

func (d Dependencies) importRawDocuments(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.RawImports == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "raw import service is unavailable")
		return
	}
	var input struct {
		IdempotencyKey string                `json:"idempotency_key"`
		Items          []rawimport.Candidate `json:"items"`
	}
	if err := decodeStrictLimited(response, request, &input); err != nil {
		writeDecodeError(response, requestID, err)
		return
	}
	result, err := d.RawImports.Import(request.Context(), principal.Identity, input.IdempotencyKey, rawimport.Batch{Items: input.Items})
	if err != nil {
		writeRawImportError(response, requestID, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeEnvelope(response, status, requestID, result)
}

func (d Dependencies) rawImportStatus(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.RawImports == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "raw import service is unavailable")
		return
	}
	key := strings.TrimSpace(request.PathValue("idempotency_key"))
	if key == "" || utf8.RuneCountInString(key) > 200 {
		writeError(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "idempotency_key must contain 1..200 characters")
		return
	}
	status, err := d.RawImports.Status(request.Context(), principal.Identity, key)
	if err != nil {
		writeRawImportError(response, requestID, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"request_id": requestID, "status": status.State, "result": status.Result})
}

func (d Dependencies) importReviewedEvent(response http.ResponseWriter, request *http.Request, _ Principal, requestID string) {
	if d.ReviewedEvents == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "reviewed event import service is unavailable")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	pkg, err := domainimport.DecodeStrict(request.Body)
	if err != nil {
		writeDecodeError(response, requestID, err)
		return
	}
	if _, err := pkg.Validate(); err != nil {
		writeError(response, requestID, http.StatusUnprocessableEntity, "REVIEWED_EVENT_IMPORT_REJECTED", "reviewed event package failed validation")
		return
	}
	result, err := d.ReviewedEvents.Import(request.Context(), pkg)
	if err != nil {
		if errors.Is(err, eventapp.ErrIdempotencyConflict) {
			writeError(response, requestID, http.StatusConflict, "EVENT_IMPORT_IDEMPOTENCY_CONFLICT", "idempotency key conflicts with reviewed event payload")
			return
		}
		writeError(response, requestID, http.StatusInternalServerError, "REVIEWED_EVENT_IMPORT_FAILED", "reviewed event import failed")
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeEnvelope(response, status, requestID, reviewedResult(result))
}

func reviewedResult(result eventapp.Result) map[string]any {
	return map[string]any{
		"package_id": result.PackageID, "receipt_id": result.ReceiptID, "event_id": result.EventID,
		"raw_document_ids": result.RawDocumentIDs, "event_source_ids": result.EventSourceIDs,
		"event_tag_map_ids": result.EventTagMapIDs, "payload_hash": result.PayloadHash,
		"counts": map[string]int{"raw_documents": len(result.RawDocumentIDs), "events": 1, "event_sources": len(result.EventSourceIDs), "event_tags": len(result.EventTagMapIDs), "receipts": 1},
	}
}
