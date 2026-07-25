package researchanchorimport

import (
	"crypto/sha1"
	"fmt"
)

var anchorNamespace = [16]byte{
	0xf2, 0x19, 0xde, 0xd4, 0xfc, 0x65, 0x59, 0x48,
	0x9e, 0x28, 0xc1, 0xcd, 0xb6, 0xa8, 0x28, 0x8e,
}

func AnchorID(themeID, centerChainNodeID string) string {
	hash := sha1.New()
	_, _ = hash.Write(anchorNamespace[:])
	_, _ = hash.Write([]byte(themeID + "\x00" + centerChainNodeID))
	identifier := hash.Sum(nil)[:16]
	identifier[6] = (identifier[6] & 0x0f) | 0x50
	identifier[8] = (identifier[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		identifier[0:4], identifier[4:6], identifier[6:8], identifier[8:10], identifier[10:16])
}
