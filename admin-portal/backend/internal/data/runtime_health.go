package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

const dataRuntimeHealthPath = dataAPIPrefix + "/runtime-health"

type runtimeHealthWire struct {
	CheckedAt time.Time                  `json:"checked_at"`
	Services  []runtimeHealthServiceWire `json:"services"`
}

type runtimeHealthServiceWire struct {
	Key         biz.RuntimeServiceKey `json:"key"`
	DisplayName string                `json:"display_name"`
	Status      biz.RuntimeStatus     `json:"status"`
	CheckedAt   time.Time             `json:"checked_at"`
	LatencyMS   *int64                `json:"latency_ms,omitempty"`
	ReasonCode  biz.RuntimeReasonCode `json:"reason_code,omitempty"`
}

func (wire *runtimeHealthWire) UnmarshalJSON(content []byte) error {
	type alias runtimeHealthWire
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*wire = runtimeHealthWire(decoded)
	return nil
}

func (wire *runtimeHealthServiceWire) UnmarshalJSON(content []byte) error {
	type alias runtimeHealthServiceWire
	var decoded alias
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*wire = runtimeHealthServiceWire(decoded)
	return nil
}

func (wire runtimeHealthWire) toBiz() (biz.ProviderRuntimeHealth, error) {
	if wire.CheckedAt.IsZero() || len(wire.Services) != 1 {
		return biz.ProviderRuntimeHealth{}, &biz.RuntimeHealthProviderError{ReasonCode: biz.RuntimeReasonInvalidResponse}
	}
	services := make([]biz.RuntimeHealthService, 0, 1)
	for _, item := range wire.Services {
		if item.Key != biz.RuntimeServiceData || item.DisplayName != item.Key.DisplayName() ||
			item.CheckedAt.IsZero() || item.LatencyMS != nil && *item.LatencyMS < 0 ||
			!item.Status.Valid() || item.Status == biz.RuntimeStatusReady && item.ReasonCode != "" ||
			item.Status != biz.RuntimeStatusReady && !item.ReasonCode.Valid() {
			return biz.ProviderRuntimeHealth{}, &biz.RuntimeHealthProviderError{ReasonCode: biz.RuntimeReasonInvalidResponse}
		}
		services = append(services, biz.RuntimeHealthService{
			Key: item.Key, DisplayName: item.DisplayName, Status: item.Status, CheckedAt: item.CheckedAt,
			LatencyMS: item.LatencyMS, ReasonCode: item.ReasonCode,
		})
	}
	return biz.ProviderRuntimeHealth{CheckedAt: wire.CheckedAt, Services: services}, nil
}

func (c *DataHTTPClient) GetRuntimeHealth(ctx context.Context) (biz.ProviderRuntimeHealth, error) {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		requestID = newRequestID()
	}
	callContext, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var envelope responseEnvelope[runtimeHealthWire]
	err := c.doJSONAttempt(callContext, http.MethodGet, "Data.GetRuntimeHealth", dataRuntimeHealthPath, dataRuntimeHealthPath, nil, requestID, &envelope)
	wire, err := unwrapEnvelope(envelope, err)
	if err != nil {
		return biz.ProviderRuntimeHealth{}, runtimeHealthProviderError(callContext, err)
	}
	return wire.toBiz()
}

func runtimeHealthProviderError(ctx context.Context, err error) error {
	reason := biz.RuntimeReasonUnreachable
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		reason = biz.RuntimeReasonTimeout
	}
	var dataError *Error
	if errors.As(err, &dataError) {
		switch {
		case dataError.Kind == ErrorKindTimeout || dataError.Kind == ErrorKindCanceled:
			reason = biz.RuntimeReasonTimeout
		case dataError.StatusCode == http.StatusUnauthorized || dataError.StatusCode == http.StatusForbidden:
			reason = biz.RuntimeReasonAuthenticationFailed
		case dataError.Kind == ErrorKindDecode || dataError.Kind == ErrorKindProtocol || dataError.Kind == ErrorKindClient:
			reason = biz.RuntimeReasonInvalidResponse
		}
	}
	return &biz.RuntimeHealthProviderError{ReasonCode: reason}
}

var _ biz.RuntimeHealthProvider = (*DataHTTPClient)(nil)
