package api

import _ "embed"

//go:embed openapi.yaml
var document []byte

func Document() []byte {
	return document
}
