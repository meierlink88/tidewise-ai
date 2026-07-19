package researchthemeimport

import (
	"crypto/sha1"
	"fmt"
)

// themeNamespace is UUIDv5(DNS, "tidewise.ai/research-theme-publication/v1").
var themeNamespace = [16]byte{
	0x7b, 0x95, 0x0d, 0x74, 0x76, 0x8c, 0x57, 0xe0,
	0x97, 0xb5, 0xea, 0x4f, 0x3d, 0xa1, 0xbc, 0x88,
}

// ThemeID returns the stable V1 UUIDv5 for a Theme snapshot. The NUL separator
// prevents ambiguous concatenation between batch IDs and batch-local keys.
func ThemeID(analysisBatchID, themeKey string) string {
	hash := sha1.New()
	_, _ = hash.Write(themeNamespace[:])
	_, _ = hash.Write([]byte(analysisBatchID + "\x00" + themeKey))
	identifier := hash.Sum(nil)[:16]
	identifier[6] = (identifier[6] & 0x0f) | 0x50
	identifier[8] = (identifier[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		identifier[0:4], identifier[4:6], identifier[6:8], identifier[8:10], identifier[10:16])
}
