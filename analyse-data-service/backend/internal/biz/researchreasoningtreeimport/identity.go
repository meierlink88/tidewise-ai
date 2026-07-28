package researchreasoningtreeimport

import (
	"crypto/sha1"
	"fmt"
	"strconv"
)

// Frozen UUIDv5 namespaces for the in-place V1 Reason Tree contract.
var (
	reasoningTreeNamespace = [16]byte{
		0x33, 0xf3, 0xa1, 0x72, 0x5c, 0x45, 0x55, 0x85,
		0x8b, 0xcd, 0x19, 0x52, 0x7b, 0x34, 0x61, 0x93,
	}
	reasoningTreeNodeNamespace = [16]byte{
		0x7e, 0x8c, 0xb1, 0x31, 0x70, 0x3b, 0x5c, 0xaf,
		0x98, 0xaa, 0x1f, 0x4d, 0x8a, 0xa6, 0xcb, 0x1e,
	}
)

func ReasoningTreeID(themeID, industryChainEntityID string) string {
	return uuidV5(reasoningTreeNamespace, themeID+"\x00"+industryChainEntityID)
}

func ReasoningTreeNodeID(reasoningTreeID string, position int, chainNodeEntityID string) string {
	return uuidV5(reasoningTreeNodeNamespace, reasoningTreeID+"\x00"+strconv.Itoa(position)+"\x00"+chainNodeEntityID)
}

func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	identifier := hash.Sum(nil)[:16]
	identifier[6] = (identifier[6] & 0x0f) | 0x50
	identifier[8] = (identifier[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		identifier[0:4], identifier[4:6], identifier[6:8], identifier[8:10], identifier[10:16])
}
