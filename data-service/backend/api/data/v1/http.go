package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

const (
	APIPrefix          = "/api/data/v1"
	MaxRequestBodySize = 1_048_576
)

type StrictJSONShape = bindingShape

func StrictJSONString() *StrictJSONShape         { return stringShape }
func StrictJSONScalar() *StrictJSONShape         { return scalarShape }
func StrictJSONBoolean() *StrictJSONShape        { return booleanShape }
func StrictJSONInteger() *StrictJSONShape        { return integerShape }
func StrictJSONNullableString() *StrictJSONShape { return nullableStringShape }
func StrictJSONAny() *StrictJSONShape            { return anyShape }
func StrictJSONArray(item *StrictJSONShape) *StrictJSONShape {
	return arrayShape(item)
}
func StrictJSONRequiredObject(required []string, fields map[string]*StrictJSONShape) *StrictJSONShape {
	return requiredObjectShape(required, fields)
}
func StrictJSONObject(fields map[string]*StrictJSONShape) *StrictJSONShape {
	return objectShape(fields)
}
func StrictJSONNullable(shape *StrictJSONShape) *StrictJSONShape {
	if shape == nil {
		return nil
	}
	clone := *shape
	clone.null = true
	return &clone
}
func DecodeStrictJSON(payload []byte, shape *StrictJSONShape, target any) error {
	return decodeStrictBinding(payload, shape, target)
}
func StrictJSONErrorPath(err error) string { return bindingErrorPath(err) }

func DecodeStrictJSONBody[T any](ctx kratoshttp.Context) (*T, error) {
	payload, err := io.ReadAll(io.LimitReader(ctx.Request().Body, MaxRequestBodySize+1))
	if err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	if len(payload) > MaxRequestBodySize {
		return nil, NewPublicError(StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes", nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var request T
	if err := decoder.Decode(&request); err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	return &request, nil
}

func ReadImportPayload(ctx kratoshttp.Context) ([]byte, error) {
	payload, err := io.ReadAll(io.LimitReader(ctx.Request().Body, MaxRequestBodySize+1))
	if err != nil {
		return nil, NewPublicError(StatusBadRequest, "INVALID_REQUEST", "request body is not valid for this contract", nil)
	}
	if len(payload) > MaxRequestBodySize {
		return nil, NewPublicError(StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1048576 bytes", nil)
	}
	return payload, nil
}

func Call[T any](ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (*Response[T], error)) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	result, err := handler(ctx, request)
	if err != nil {
		return err
	}
	response, ok := result.(*Response[T])
	if !ok || response == nil {
		return fmt.Errorf("data API operation %s returned an invalid response", operation)
	}
	return ctx.Result(response.Status, response.Result)
}

func ParseBoundedInt(raw string, fallback, minimum, maximum int, name string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, NewPublicError(StatusBadRequest, "INVALID_REQUEST", fmt.Sprintf("%s must be between %d and %d", name, minimum, maximum), nil)
	}
	return value, nil
}
