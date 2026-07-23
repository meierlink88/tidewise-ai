package openapi

import (
	_ "embed"
	"net/http"

	"github.com/swaggest/swgui/v5emb"
)

//go:embed openapi.yaml
var document []byte

func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /openapi.yaml", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(document)
	})
	mux.Handle("GET /docs/", v5emb.New("Tidewise AI AgentRun API", "/openapi.yaml", "/docs/"))
}
