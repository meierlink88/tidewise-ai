package v1

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func Bind(ctx kratoshttp.Context, target any) error {
	mediaType, _, err := mime.ParseMediaType(ctx.Request().Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &Error{400, "INVALID_REQUEST"}
	}
	limit := int64(4096)
	if ctx.Request().URL.Path == "/api/user/v1/watchlist/check" {
		limit = 8192
	}
	if ctx.Request().URL.Path == "/api/user/v1/profiles/nickname" {
		limit = 3 * 1024 * 1024
	}
	body, err := io.ReadAll(http.MaxBytesReader(ctx.Response(), ctx.Request().Body, limit))
	if err != nil {
		return &Error{400, "INVALID_REQUEST"}
	}
	// Require an object and reject duplicate keys before decoding the typed DTO.
	check := json.NewDecoder(bytes.NewReader(body))
	first, err := check.Token()
	if err != nil || first != json.Delim('{') {
		return &Error{400, "INVALID_REQUEST"}
	}
	seen := map[string]bool{}
	for check.More() {
		key, err := check.Token()
		if err != nil {
			return &Error{400, "INVALID_REQUEST"}
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return &Error{400, "INVALID_REQUEST"}
		}
		seen[name] = true
		var value json.RawMessage
		if check.Decode(&value) != nil {
			return &Error{400, "INVALID_REQUEST"}
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		return &Error{400, "INVALID_REQUEST"}
	}

	return nil
}

type Error struct {
	Status int
	Code   string
}

func (e *Error) Error() string { return e.Code }
