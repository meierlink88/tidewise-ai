package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
)

const (
	ScopeResearchRead        = "data.research.read"
	ScopeResearchImport      = "data.research.import"
	ScopeAdminRead           = "data.admin.read"
	ScopeReviewedEventImport = "data.reviewed-events.import"
	ScopeRawEvidenceImport   = "data.raw-evidences.import"
	ScopeEvidenceImport      = "data.evidences.import"
	ScopeEventTagRead        = "data.event-tags.read"
	ScopeEventSemanticsRead  = "data.event-semantics.read"
	ScopeEventSemanticsWrite = "data.event-semantics.write"
	operationHealth          = "data.health"
	operationReady           = "data.ready"
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
	case v1.OperationPublishReviewedEvents:
		return ScopeReviewedEventImport, true
	case v1.OperationPublishRawEvidence:
		return ScopeRawEvidenceImport, true
	case v1.OperationPublishEvidence:
		return ScopeEvidenceImport, true
	case v1.OperationListActiveEventTags:
		return ScopeEventTagRead, true
	case v1.OperationPublishResearchTheme:
		return ScopeResearchImport, true
	case v1.OperationListResearchThemes, v1.OperationGetResearchTheme,
		v1.OperationListResearchThemeReasoningTrees, v1.OperationGetResearchThemeReasoningTree,
		v1.OperationListResearchAnalysisContext, v1.OperationSearchResearchGraph:
		return ScopeResearchRead, true
	case v1.OperationListAdminRawDocuments, v1.OperationListAdminEvents, v1.OperationGetRuntimeHealth:
		return ScopeAdminRead, true
	case v1.OperationListEligibleEventSemanticEvents,
		v1.OperationGetEventSemanticContext, v1.OperationGetEventSemantics:
		return ScopeEventSemanticsRead, true
	case v1.OperationCreateEventSemanticContextLease,
		v1.OperationCreateEventSemanticSubmission,
		v1.OperationSubmitEventSemanticReview:
		return ScopeEventSemanticsWrite, true
	default:
		return "", false
	}
}
