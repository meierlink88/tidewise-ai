package v1

import (
	_ "embed"
)

//go:embed openapi.yaml
var document []byte

func Document() []byte {
	return append([]byte(nil), document...)
}
