package internalapi

import (
	"errors"
	"net/http"

	publicationdomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventpublication"
	researchanchordomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchanchorimport"
	researchdomainimport "github.com/meierlink88/tidewise-ai/backend/services/data/domain/researchthemeimport"
	eventpublicationapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/eventpublication"
	researchanchorimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchanchorimport"
	researchimportapp "github.com/meierlink88/tidewise-ai/backend/services/data/usecase/researchthemeimport"
)

func (d Dependencies) importEventPublication(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.EventPublications == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "Event Publication service is unavailable")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	publication, err := publicationdomain.DecodeStrict(request.Body)
	if err != nil {
		writeDecodeError(response, requestID, err)
		return
	}
	result, err := d.EventPublications.Import(request.Context(), principal.Identity, publication)
	if err != nil {
		var validation *publicationdomain.ValidationError
		if errors.As(err, &validation) {
			writeErrorWithDetails(
				response, requestID, http.StatusUnprocessableEntity,
				"EVENT_PUBLICATION_INVALID", "Event Publication failed validation", map[string]any{"issues": validation.Issues},
			)
			return
		}
		var conflict *eventpublicationapp.ConflictError
		if errors.As(err, &conflict) {
			writeErrorWithDetails(
				response, requestID, http.StatusConflict,
				"EVENT_PUBLICATION_CONFLICT", "Event Publication conflicts with stored data", map[string]any{"issues": conflict.Issues},
			)
			return
		}
		writeError(response, requestID, http.StatusInternalServerError, "EVENT_PUBLICATION_FAILED", "Event Publication failed")
		return
	}
	writeEnvelope(response, http.StatusCreated, requestID, result)
}

func (d Dependencies) importResearchThemes(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.ResearchThemeImports == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Theme import service is unavailable")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	batch, err := researchdomainimport.DecodeStrict(request.Body)
	if err != nil {
		var decodeError *researchdomainimport.DecodeError
		if errors.As(err, &decodeError) {
			writeErrorWithDetails(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Theme V1 contract", map[string]any{
				"theme_key": decodeError.ThemeKey, "path": decodeError.Path,
			})
			return
		}
		writeDecodeError(response, requestID, err)
		return
	}
	result, err := d.ResearchThemeImports.Import(request.Context(), principal.Identity, batch)
	if err != nil {
		writeResearchThemeImportError(response, requestID, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeEnvelope(response, status, requestID, result)
}

func writeResearchThemeImportError(response http.ResponseWriter, requestID string, err error) {
	var validation *researchdomainimport.ValidationError
	if errors.As(err, &validation) {
		writeErrorWithDetails(response, requestID, http.StatusBadRequest, "RESEARCH_THEME_IMPORT_REJECTED", "research Theme batch failed validation", map[string]any{
			"theme_key": validation.ThemeKey, "path": validation.Path, "reference": validation.Reference,
		})
		return
	}
	var reference *researchimportapp.ReferenceError
	if errors.As(err, &reference) {
		writeErrorWithDetails(response, requestID, http.StatusUnprocessableEntity, "RESEARCH_THEME_REFERENCE_NOT_FOUND", "research Theme batch references missing master data", map[string]any{
			"theme_key": reference.ThemeKey, "path": reference.Path, "reference": reference.Reference,
		})
		return
	}
	switch {
	case errors.Is(err, researchimportapp.ErrPayloadConflict):
		writeError(response, requestID, http.StatusConflict, "RESEARCH_THEME_PAYLOAD_CONFLICT", "analysis_batch_id conflicts with the published payload")
	case errors.Is(err, researchimportapp.ErrPublisherConflict):
		writeError(response, requestID, http.StatusConflict, "RESEARCH_THEME_PUBLISHER_CONFLICT", "analysis_batch_id belongs to another publisher subject")
	default:
		writeError(response, requestID, http.StatusInternalServerError, "RESEARCH_THEME_IMPORT_FAILED", "research Theme import failed")
	}
}

func (d Dependencies) importResearchAnchors(response http.ResponseWriter, request *http.Request, principal Principal, requestID string) {
	if d.ResearchAnchorImports == nil {
		writeError(response, requestID, http.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research Anchor import service is unavailable")
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, MaxRequestBodyBytes)
	publication, err := researchanchordomainimport.DecodeStrict(request.Body)
	if err != nil {
		var decodeError *researchanchordomainimport.DecodeError
		if errors.As(err, &decodeError) {
			writeResearchAnchorErrorDetails(response, requestID, http.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Anchor V1 contract", decodeError.CenterChainNodeID, decodeError.Path, "")
			return
		}
		writeDecodeError(response, requestID, err)
		return
	}
	result, err := d.ResearchAnchorImports.Import(request.Context(), principal.Identity, publication)
	if err != nil {
		writeResearchAnchorImportError(response, requestID, err)
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeEnvelope(response, status, requestID, result)
}

func writeResearchAnchorImportError(response http.ResponseWriter, requestID string, err error) {
	var validation *researchanchordomainimport.ValidationError
	if errors.As(err, &validation) {
		writeResearchAnchorErrorDetails(response, requestID, http.StatusBadRequest, "RESEARCH_ANCHOR_IMPORT_REJECTED", "research Anchor publication failed validation", validation.CenterChainNodeID, validation.Path, validation.Reference)
		return
	}
	var contractError *researchanchorimportapp.ContractError
	if errors.As(err, &contractError) {
		writeResearchAnchorErrorDetails(response, requestID, http.StatusBadRequest, "RESEARCH_ANCHOR_IMPORT_REJECTED", "research Anchor publication failed validation", contractError.CenterChainNodeID, contractError.Path, contractError.Reference)
		return
	}
	var reference *researchanchorimportapp.ReferenceError
	if errors.As(err, &reference) {
		code := "RESEARCH_ANCHOR_REFERENCE_NOT_FOUND"
		message := "research Anchor publication references missing data"
		if reference.Kind == researchanchorimportapp.ReferenceInvalid {
			code = "RESEARCH_ANCHOR_REFERENCE_INVALID"
			message = "research Anchor publication references data outside its Theme boundary"
		}
		writeResearchAnchorErrorDetails(response, requestID, http.StatusUnprocessableEntity, code, message, reference.CenterChainNodeID, reference.Path, reference.Reference)
		return
	}
	switch {
	case errors.Is(err, researchanchorimportapp.ErrPayloadConflict):
		writeError(response, requestID, http.StatusConflict, "RESEARCH_ANCHOR_PAYLOAD_CONFLICT", "theme_id conflicts with the published Research Anchor payload")
	case errors.Is(err, researchanchorimportapp.ErrPublisherConflict):
		writeError(response, requestID, http.StatusConflict, "RESEARCH_ANCHOR_PUBLISHER_CONFLICT", "Theme or Anchor receipt belongs to another publisher subject")
	default:
		writeError(response, requestID, http.StatusInternalServerError, "RESEARCH_ANCHOR_IMPORT_FAILED", "research Anchor import failed")
	}
}

func writeResearchAnchorErrorDetails(response http.ResponseWriter, requestID string, status int, code, message, centerID, path, reference string) {
	writeErrorWithDetails(response, requestID, status, code, message, map[string]any{
		"center_chain_node_id": centerID,
		"path":                 path,
		"reference":            reference,
	})
}
