package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	chainnodeapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/chainnode"
	conceptapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/concept"
	countryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/country"
	industryapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industry"
	industrychainapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/industrychain"
	organizationapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/entity/organization"
	eventapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/event"
	evidenceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/evidence"
	researchapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/research"
	runtimehealthapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/runtimehealth"
	sourceapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/source"
)

const (
	ScopeResearchRead         = "data.research.read"
	ScopeResearchImport       = "data.research.import"
	ScopeAdminRead            = "data.admin.read"
	ScopeEventPublish         = "data.events.publish"
	ScopeRawEvidenceImport    = "data.raw-evidences.import"
	ScopeRawEvidenceRead      = "data.raw-evidences.read"
	ScopeEvidenceImport       = "data.evidences.import"
	ScopeEvidenceCategoryRead = "data.evidence-categories.read"
	ScopeCountryRead          = "data.countries.read"
	ScopeCountryWrite         = "data.countries.write"
	ScopeIndustryRead         = "data.industries.read"
	ScopeIndustryWrite        = "data.industries.write"
	ScopeConceptRead          = "data.concepts.read"
	ScopeConceptWrite         = "data.concepts.write"
	ScopeChainNodeRead        = "data.chain-nodes.read"
	ScopeChainNodeWrite       = "data.chain-nodes.write"
	ScopeIndustryChainRead    = "data.industry-chains.read"
	ScopeIndustryChainWrite   = "data.industry-chains.write"
	ScopeOrganizationRead     = "data.organizations.read"
	ScopeOrganizationWrite    = "data.organizations.write"
	ScopeSourceRead           = "data.sources.read"
	ScopeSourceWrite          = "data.sources.write"
	operationHealth           = "data.health"
	operationReady            = "data.ready"
)

type Credential struct {
	Secret    string
	Principal v1.Principal
}

type Authenticator struct {
	credentials []Credential
}

func NewAuthenticator(credentials []Credential) (*Authenticator, error) {
	result := &Authenticator{credentials: make([]Credential, 0, len(credentials))}
	seenSecret := map[string]struct{}{}
	for _, credential := range credentials {
		credential.Secret = strings.TrimSpace(credential.Secret)
		credential.Principal.Identity = strings.TrimSpace(credential.Principal.Identity)
		if credential.Secret == "" || credential.Principal.Identity == "" || len(credential.Principal.Scopes) == 0 {
			return nil, fmt.Errorf("service credential, identity and scopes are required")
		}
		if utf8.RuneCountInString(credential.Principal.Identity) > 200 {
			return nil, fmt.Errorf("service identity must contain at most 200 characters")
		}
		if _, duplicate := seenSecret[credential.Secret]; duplicate {
			return nil, fmt.Errorf("service credentials must be unique")
		}
		seenSecret[credential.Secret] = struct{}{}
		result.credentials = append(result.credentials, credential)
	}
	return result, nil
}

func (a *Authenticator) Authenticate(header string) (v1.Principal, bool) {
	const prefix = "Bearer "
	if a == nil || !strings.HasPrefix(header, prefix) {
		return v1.Principal{}, false
	}
	presented := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	for _, credential := range a.credentials {
		if len(presented) == len(credential.Secret) &&
			subtle.ConstantTimeCompare([]byte(presented), []byte(credential.Secret)) == 1 {
			return credential.Principal, true
		}
	}
	return v1.Principal{}, false
}

func authenticationMiddleware(authenticator *Authenticator) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, v1.NewPublicError(v1.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error", nil)
			}
			operation := serverTransport.Operation()
			if operation == operationHealth || operation == operationReady {
				return next(ctx, request)
			}
			scope, known := requiredScope(operation)
			if !known {
				return nil, v1.NewPublicError(v1.StatusInternalServerError, "INTERNAL_ERROR", "internal data service error", nil)
			}
			principal, authenticated := authenticator.Authenticate(serverTransport.RequestHeader().Get("Authorization"))
			if !authenticated {
				return nil, v1.NewPublicError(v1.StatusUnauthorized, "UNAUTHENTICATED", "valid service identity is required", nil)
			}
			if !principal.HasScope(scope) {
				return nil, v1.NewPublicError(v1.StatusForbidden, "FORBIDDEN", "service identity lacks the required scope", nil)
			}
			return next(v1.WithPrincipal(ctx, principal), request)
		}
	}
}

func requiredScope(operation string) (string, bool) {
	switch operation {
	case evidenceapi.OperationPublishRawEvidence:
		return ScopeRawEvidenceImport, true
	case evidenceapi.OperationGetRawEvidence:
		return ScopeRawEvidenceRead, true
	case evidenceapi.OperationPublishEvidence:
		return ScopeEvidenceImport, true
	case evidenceapi.OperationListEvidenceCategories:
		return ScopeEvidenceCategoryRead, true
	case researchapi.OperationPublishResearchTheme:
		return ScopeResearchImport, true
	case eventapi.OperationPublishEvent:
		return ScopeEventPublish, true
	case researchapi.OperationListResearchThemes, researchapi.OperationGetResearchTheme,
		researchapi.OperationListResearchThemeReasoningTrees, researchapi.OperationGetResearchThemeReasoningTree,
		researchapi.OperationSearchResearchGraph:
		return ScopeResearchRead, true
	case eventapi.OperationListAdminEvents, evidenceapi.OperationListAdminEvidence, runtimehealthapi.OperationGet:
		return ScopeAdminRead, true
	case countryapi.OperationList, countryapi.OperationGet:
		return ScopeCountryRead, true
	case countryapi.OperationCreate, countryapi.OperationUpdate, countryapi.OperationReplaceRegions:
		return ScopeCountryWrite, true
	case industryapi.OperationList, industryapi.OperationGet:
		return ScopeIndustryRead, true
	case industryapi.OperationCreate, industryapi.OperationUpdate:
		return ScopeIndustryWrite, true
	case conceptapi.OperationList, conceptapi.OperationGet:
		return ScopeConceptRead, true
	case conceptapi.OperationCreate, conceptapi.OperationUpdate:
		return ScopeConceptWrite, true
	case chainnodeapi.OperationList, chainnodeapi.OperationGet:
		return ScopeChainNodeRead, true
	case chainnodeapi.OperationCreate, chainnodeapi.OperationUpdate:
		return ScopeChainNodeWrite, true
	case industrychainapi.OperationList, industrychainapi.OperationGet:
		return ScopeIndustryChainRead, true
	case industrychainapi.OperationCreate, industrychainapi.OperationUpdate:
		return ScopeIndustryChainWrite, true
	case organizationapi.OperationList, organizationapi.OperationGet, organizationapi.OperationGetCatalog, organizationapi.OperationListMembers:
		return ScopeOrganizationRead, true
	case organizationapi.OperationCreate, organizationapi.OperationUpdate, organizationapi.OperationReplaceDomainTags,
		organizationapi.OperationCreateMember, organizationapi.OperationUpdateMember, organizationapi.OperationDeleteMember:
		return ScopeOrganizationWrite, true
	case sourceapi.OperationList, sourceapi.OperationSnapshot:
		return ScopeSourceRead, true
	case sourceapi.OperationCreate, sourceapi.OperationUpdate, sourceapi.OperationDelete:
		return ScopeSourceWrite, true
	default:
		return "", false
	}
}
