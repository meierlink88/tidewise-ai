package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}

func publicErrorWithDetails(status int, code, message string, details any) error {
	return v1.NewPublicError(status, code, message, details)
}

func decodeError(err error) error {
	if err != nil && strings.Contains(err.Error(), "request body too large") {
		return publicError(v1.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes")
	}
	return publicError(v1.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract")
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

func principalIdentity(ctx context.Context) string {
	if principal, ok := v1.PrincipalFromContext(ctx); ok {
		return principal.Identity
	}
	return ""
}

func asPublicError(err error) *v1.PublicError {
	var public *v1.PublicError
	if errors.As(err, &public) {
		return public
	}
	return nil
}
