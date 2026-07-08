package repositories

import (
	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func IsUUID(value string) bool {
	return uuidPattern.MatchString(strings.TrimSpace(value))
}

func NormalizeUUID(value string, parts ...string) string {
	value = strings.TrimSpace(value)
	if IsUUID(value) {
		return strings.ToLower(value)
	}

	seedParts := append([]string{value}, parts...)
	seed := strings.Join(seedParts, "\x00")
	if seed == "" {
		seed = "tidewise-empty-id"
	}

	sum := sha1.Sum([]byte(seed))
	bytes := sum[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x50
	bytes[8] = (bytes[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16],
	)
}

func RawDocumentUUID(sourceID string, candidateID string, sourceExternalID string, contentHash string) string {
	if IsUUID(candidateID) {
		return strings.ToLower(strings.TrimSpace(candidateID))
	}
	key := strings.TrimSpace(sourceExternalID)
	if key == "" {
		key = strings.TrimSpace(contentHash)
	}
	if key == "" {
		key = strings.TrimSpace(candidateID)
	}
	return NormalizeUUID("raw_document", sourceID, key)
}
