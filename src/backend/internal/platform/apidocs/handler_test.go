package apidocs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meierlink88/tidewise-ai/backend/internal/platform/runtimeconfig"
)

func TestWrapPublishesEmbeddedContractAndSwaggerOutsideProduction(t *testing.T) {
	t.Parallel()

	document := []byte("openapi: 3.0.4\ninfo:\n  title: Test API\n  version: 1.0.0\npaths: {}\n")
	for _, environment := range []runtimeconfig.Environment{runtimeconfig.EnvLocal, runtimeconfig.EnvUAT} {
		t.Run(string(environment), func(t *testing.T) {
			handler := Wrap(environment, http.NotFoundHandler(), Config{
				Title:    "Test API",
				Document: document,
			})

			spec := serve(t, handler, "/openapi.yaml")
			if spec.Code != http.StatusOK {
				t.Fatalf("GET /openapi.yaml status = %d, want %d", spec.Code, http.StatusOK)
			}
			if contentType := spec.Header().Get("Content-Type"); contentType != "application/yaml; charset=utf-8" {
				t.Fatalf("GET /openapi.yaml content type = %q", contentType)
			}
			if got := spec.Body.String(); got != string(document) {
				t.Fatalf("GET /openapi.yaml body = %q, want embedded document", got)
			}

			redirect := serve(t, handler, "/docs")
			if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "/docs/" {
				t.Fatalf("GET /docs = status %d location %q", redirect.Code, redirect.Header().Get("Location"))
			}

			index := serve(t, handler, "/docs/")
			if index.Code != http.StatusOK {
				t.Fatalf("GET /docs/ status = %d, want %d", index.Code, http.StatusOK)
			}
			if !strings.Contains(index.Body.String(), "Test API") {
				t.Fatalf("GET /docs/ does not contain configured title")
			}
			if !strings.Contains(index.Body.String(), "persistAuthorization: false") {
				t.Fatal("GET /docs/ must disable persisted Swagger authorization")
			}
		})
	}
}

func TestWrapDoesNotRegisterDocumentationInProduction(t *testing.T) {
	t.Parallel()

	fallback := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.NotFound(response, request)
	})
	handler := Wrap(runtimeconfig.EnvProd, fallback, Config{
		Title:    "Test API",
		Document: []byte("openapi: 3.0.4\n"),
	})

	for _, path := range []string{"/openapi.yaml", "/docs", "/docs/"} {
		response := serve(t, handler, path)
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestWrapPreservesApplicationRoutes(t *testing.T) {
	t.Parallel()

	application := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})
	handler := Wrap(runtimeconfig.EnvLocal, application, Config{
		Title:    "Test API",
		Document: []byte("openapi: 3.0.4\n"),
	})

	response := serve(t, handler, "/api/test/v1/resources")
	if response.Code != http.StatusNoContent {
		t.Fatalf("application route status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func serve(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Result().Body != nil {
		t.Cleanup(func() {
			_, _ = io.Copy(io.Discard, response.Result().Body)
			_ = response.Result().Body.Close()
		})
	}
	return response
}
