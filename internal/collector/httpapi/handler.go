package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun"
	agentrunopenapi "github.com/guanchaojia/tidewise-ai-agentrun/internal/agentrun/openapi"
	collectorapp "github.com/guanchaojia/tidewise-ai-agentrun/internal/collector/application"
	"github.com/jackc/pgx/v5"
)

const (
	runsPath        = "/internal/agent-run/v1/collector/runs"
	maxPromptBytes  = 64 * 1024
	maxRequestBytes = maxPromptBytes*6 + 4096
)

type application interface {
	Ready(context.Context) error
	CreateCollectorRun(context.Context, string, string) (agentrun.Execution, agentrun.CreateDisposition, error)
	GetCollectorRun(context.Context, string) (agentrun.Execution, error)
}

type handler struct {
	application  application
	serviceToken string
}

func NewHandler(application application, serviceToken string) http.Handler {
	h := &handler{application: application, serviceToken: serviceToken}
	mux := http.NewServeMux()
	agentrunopenapi.Register(mux)
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /readyz", h.ready)
	mux.Handle("POST "+runsPath, h.authenticate(http.HandlerFunc(h.createCollectorRun)))
	mux.Handle("GET "+runsPath+"/", h.authenticate(http.HandlerFunc(h.getCollectorRun)))
	return mux
}

func (h *handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if h.serviceToken == "" || request.Header.Get("Authorization") != "Bearer "+h.serviceToken {
			writeError(writer, http.StatusUnauthorized, "unauthorized", "Authentication is required")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (h *handler) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *handler) ready(writer http.ResponseWriter, request *http.Request) {
	if h.serviceToken == "" || h.application.Ready(request.Context()) != nil {
		writeError(writer, http.StatusServiceUnavailable, "not_ready", "AgentRun is not ready")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"status": "ready"})
}

func (h *handler) createCollectorRun(writer http.ResponseWriter, request *http.Request) {
	idempotencyKey := strings.TrimSpace(request.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeError(writer, http.StatusBadRequest, "idempotency_key_required", "Idempotency-Key is required")
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, maxRequestBytes+1))
	if err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if len(body) > maxRequestBytes {
		writeError(writer, http.StatusRequestEntityTooLarge, "prompt_too_large", "Prompt exceeds 64 KiB")
		return
	}
	var input struct {
		Prompt string `json:"prompt"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeError(writer, http.StatusBadRequest, "invalid_request", "Request body is invalid")
		return
	}
	if strings.TrimSpace(input.Prompt) == "" {
		writeError(writer, http.StatusBadRequest, "prompt_required", "Prompt must not be blank")
		return
	}
	if len([]byte(input.Prompt)) > maxPromptBytes {
		writeError(writer, http.StatusRequestEntityTooLarge, "prompt_too_large", "Prompt exceeds 64 KiB")
		return
	}
	execution, _, err := h.application.CreateCollectorRun(request.Context(), idempotencyKey, input.Prompt)
	if err != nil {
		h.writeCreateError(writer, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, runResponse(execution))
}

func (h *handler) writeCreateError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, collectorapp.ErrNotReady):
		writeError(writer, http.StatusServiceUnavailable, "configuration_not_ready", "Collector configuration is not ready")
	case errors.Is(err, agentrun.ErrIdempotencyConflict):
		writeError(writer, http.StatusConflict, "idempotency_conflict", "Idempotency-Key belongs to another Prompt")
	default:
		var active *agentrun.ActiveExecutionError
		if errors.As(err, &active) {
			writeJSON(writer, http.StatusConflict, map[string]any{
				"error_code": "active_execution_exists", "message": "Another Collector run is active",
				"active_execution_id":  active.ActiveExecutionID,
				"skipped_execution_id": active.SkippedExecutionID,
			})
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not create Collector run")
	}
}

func (h *handler) getCollectorRun(writer http.ResponseWriter, request *http.Request) {
	id := strings.TrimPrefix(request.URL.Path, runsPath+"/")
	if id == "" || strings.Contains(id, "/") {
		writeError(writer, http.StatusNotFound, "execution_not_found", "Agent Execution was not found")
		return
	}
	if _, err := uuid.Parse(id); err != nil {
		writeError(writer, http.StatusNotFound, "execution_not_found", "Agent Execution was not found")
		return
	}
	execution, err := h.application.GetCollectorRun(request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(writer, http.StatusNotFound, "execution_not_found", "Agent Execution was not found")
			return
		}
		writeError(writer, http.StatusInternalServerError, "internal_error", "Could not read Agent Execution")
		return
	}
	writeJSON(writer, http.StatusOK, runResponse(execution))
}

func runResponse(execution agentrun.Execution) map[string]any {
	return map[string]any{
		"schema": "collector_run.v1", "agent_key": "collector", "agent_version": execution.AgentVersion,
		"execution_id": execution.ID, "status": execution.Status,
		"status_url":    runsPath + "/" + execution.ID,
		"prompt_sha256": execution.PromptSHA256, "prompt_bytes": execution.PromptBytes,
		"invocations": execution.Invocations, "candidate_counts": execution.CandidateCounts,
		"artifacts": execution.Artifacts, "created_at": execution.CreatedAt,
		"started_at": execution.StartedAt, "completed_at": execution.CompletedAt,
		"error_code": execution.ErrorCode, "error_summary": execution.ErrorSummary,
		"stop_reason": execution.StopReason, "blocked_by_execution_id": execution.BlockedByExecutionID,
	}
}

func writeError(writer http.ResponseWriter, status int, code, message string) {
	writeJSON(writer, status, map[string]any{"error_code": code, "message": message})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
