package connectors

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type failingHTTPClient struct {
	err error
}

type staticHTTPClient struct {
	status int
	body   string
}

func (c staticHTTPClient) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: c.status,
		Body:       io.NopCloser(strings.NewReader(c.body)),
	}, nil
}

func (c failingHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, c.err
}

func TestPostJSONDoesNotExposeTransportOrResponseDetails(t *testing.T) {
	const secret = "connector-secret-value"
	err := postJSON(
		context.Background(),
		failingHTTPClient{err: errors.New("transport leaked " + secret)},
		"https://example.com/search",
		map[string]string{"Authorization": "Bearer " + secret},
		map[string]string{"query": "market"},
		&map[string]any{},
	)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("transport error = %v", err)
	}

	err = postJSON(
		context.Background(),
		staticHTTPClient{status: http.StatusOK, body: `{"secret":"` + secret + `"`},
		"https://example.com/search",
		nil,
		map[string]string{"query": "market"},
		&struct {
			Count int `json:"count"`
		}{},
	)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("decode error = %v", err)
	}
}
