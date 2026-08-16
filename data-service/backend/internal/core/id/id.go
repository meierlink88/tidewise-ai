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

// Kind identifies one reviewed Data Application object identity contract.
type Kind string

const (
	Entity                            Kind = "ENT"
	EntityRelation                    Kind = "ERL"
	Country                           Kind = "COU"
	Region                            Kind = "REG"
	Organization                      Kind = "ORG"
	OrganizationCategory              Kind = "OCA"
	OrganizationDomainTag             Kind = "ODT"
	OrganizationDomainTagLink         Kind = "ODL"
	RawEvidence                       Kind = "RAW"
	Evidence                          Kind = "EVD"
	EvidenceCategory                  Kind = "EVC"
	RawEvidenceCategoryLink           Kind = "RCL"
	ChainNodePhysicalConstraint       Kind = "CPC"
	ChainNodeRelation                 Kind = "CNR"
	CountryRegionLink                 Kind = "CRL"
	DirectImpactAssertion             Kind = "DIA"
	EntityExternalIdentifier          Kind = "EEI"
	EventEntityLink                   Kind = "ENL"
	EventPublicationReceipt           Kind = "EPR"
	EventSemanticCandidateSnapshot    Kind = "ECS"
	EventSemanticContextLease         Kind = "SCL"
	EventSemanticResolutionBinding    Kind = "ERB"
	EventSemanticReviewSnapshot       Kind = "ERS"
	EventSemanticSubmission           Kind = "ESS"
	EventEvidenceLink                 Kind = "EEL"
	EventTagDefinition                Kind = "ETD"
	EventTagAssignment                Kind = "ETA"
	Event                             Kind = "EVT"
	IndustryChainGraphEdge            Kind = "IGE"
	IndustryRelationshipImportReceipt Kind = "IRI"
	OrganizationMembership            Kind = "OMB"
	EventEvidenceRecord               Kind = "EER"
	ResearchReasoningTreeReceipt      Kind = "RRI"
	ResearchReasoningTreeNode         Kind = "RRN"
	ResearchReasoningTree             Kind = "RRT"
	ResearchThemeReceipt              Kind = "RTI"
	ResearchTheme                     Kind = "RTH"
	VariableSignalMeasurement         Kind = "VSM"
	VariableSignal                    Kind = "VSG"
)

var (
	ErrInvalidPrefix   = errors.New("ID prefix must contain 2 to 8 uppercase ASCII letters")
	ErrInvalidIdentity = errors.New("ID must equal its prefix immediately followed by a canonical lowercase UUID")
	ErrInvalidSeed     = errors.New("deterministic ID seed must contain a nonblank namespace and part")
)

// New creates a random identity for a reviewed object kind.
func New(kind Kind) (string, error) {
	prefix, err := prefix(kind)
	if err != nil {
		return "", err
	}
	return prefix + uuid.NewString(), nil
}

// Derive creates a stable identity for portable catalogs and deterministic imports.
func Derive(kind Kind, namespace string, parts ...string) (string, error) {
	prefix, err := prefix(kind)
	if err != nil {
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
func FromUUID(kind Kind, suffix uuid.UUID) (string, error) {
	prefix, err := prefix(kind)
	if err != nil {
		return "", err
	}
	if suffix == uuid.Nil {
		return "", fmt.Errorf("%w: UUID suffix is nil", ErrInvalidIdentity)
	}
	return prefix + suffix.String(), nil
}

// Parse validates a canonical identity for the expected domain prefix.
func Parse(value string, kind Kind) (uuid.UUID, error) {
	expectedPrefix, err := prefix(kind)
	if err != nil {
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
func Is(value string, kind Kind) bool {
	_, err := Parse(value, kind)
	return err == nil
}

// Prefix returns the registered wire prefix for kind.
func Prefix(kind Kind) string {
	prefix, _ := prefix(kind)
	return prefix
}

func prefix(kind Kind) (string, error) {
	prefix := string(kind)
	if !registered(kind) {
		return "", ErrInvalidPrefix
	}
	if len(prefix) < minPrefixLength || len(prefix) > maxPrefixLength {
		return "", ErrInvalidPrefix
	}
	for _, character := range prefix {
		if character < 'A' || character > 'Z' {
			return "", ErrInvalidPrefix
		}
	}
	return prefix, nil
}

func registered(kind Kind) bool {
	switch kind {
	case Entity, EntityRelation, Country, Region, Organization, OrganizationCategory, OrganizationDomainTag,
		OrganizationDomainTagLink, RawEvidence, Evidence, EvidenceCategory, RawEvidenceCategoryLink,
		ChainNodePhysicalConstraint, ChainNodeRelation, CountryRegionLink, DirectImpactAssertion,
		EntityExternalIdentifier, EventEntityLink, EventPublicationReceipt, EventSemanticCandidateSnapshot,
		EventSemanticContextLease, EventSemanticResolutionBinding, EventSemanticReviewSnapshot,
		EventSemanticSubmission, EventEvidenceLink, EventTagDefinition, EventTagAssignment, Event,
		IndustryChainGraphEdge, IndustryRelationshipImportReceipt, OrganizationMembership,
		EventEvidenceRecord, ResearchReasoningTreeReceipt, ResearchReasoningTreeNode, ResearchReasoningTree,
		ResearchThemeReceipt, ResearchTheme, VariableSignalMeasurement, VariableSignal:
		return true
	default:
		return false
	}
}
