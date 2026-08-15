// Package id provides database-independent domain-object identity primitives.
package id

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	minPrefixLength = 2
	maxPrefixLength = 8
)

var (
	ErrInvalidPrefix   = errors.New("ID prefix must contain 2 to 8 uppercase ASCII letters")
	ErrInvalidIdentity = errors.New("ID must equal its prefix immediately followed by a canonical lowercase UUID")
	ErrInvalidSeed     = errors.New("deterministic ID seed must contain a nonblank namespace and part")
)

// New creates a random identity for a domain-owned prefix.
func New(prefix string) (string, error) {
	if err := validatePrefix(prefix); err != nil {
		return "", err
	}
	return prefix + uuid.NewString(), nil
}

// Derive creates a stable identity for portable catalogs and deterministic imports.
func Derive(prefix, namespace string, parts ...string) (string, error) {
	if err := validatePrefix(prefix); err != nil {
		return "", err
	}
	if strings.TrimSpace(namespace) == "" || len(parts) == 0 {
		return "", ErrInvalidSeed
	}
	seedParts := make([]string, 0, len(parts)+2)
	seedParts = append(seedParts, prefix, strings.TrimSpace(namespace))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", ErrInvalidSeed
		}
		seedParts = append(seedParts, part)
	}
	suffix := uuid.NewSHA1(uuid.NameSpaceOID, []byte(strings.Join(seedParts, "\x00")))
	return prefix + suffix.String(), nil
}

// FromUUID preserves an existing UUID as the suffix of a domain identity.
func FromUUID(prefix string, suffix uuid.UUID) (string, error) {
	if err := validatePrefix(prefix); err != nil {
		return "", err
	}
	if suffix == uuid.Nil {
		return "", fmt.Errorf("%w: UUID suffix is nil", ErrInvalidIdentity)
	}
	return prefix + suffix.String(), nil
}

// Parse validates a canonical identity for the expected domain prefix.
func Parse(value, expectedPrefix string) (uuid.UUID, error) {
	if err := validatePrefix(expectedPrefix); err != nil {
		return uuid.Nil, err
	}
	if !strings.HasPrefix(value, expectedPrefix) {
		return uuid.Nil, ErrInvalidIdentity
	}
	suffix := strings.TrimPrefix(value, expectedPrefix)
	parsed, err := uuid.Parse(suffix)
	if err != nil || parsed == uuid.Nil || parsed.String() != suffix {
		return uuid.Nil, ErrInvalidIdentity
	}
	return parsed, nil
}

// Is reports whether value is canonical for the expected domain prefix.
func Is(value, expectedPrefix string) bool {
	_, err := Parse(value, expectedPrefix)
	return err == nil
}

func validatePrefix(prefix string) error {
	if len(prefix) < minPrefixLength || len(prefix) > maxPrefixLength {
		return ErrInvalidPrefix
	}
	for _, character := range prefix {
		if character < 'A' || character > 'Z' {
			return ErrInvalidPrefix
		}
	}
	return nil
}
