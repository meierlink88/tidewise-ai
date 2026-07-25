package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func TestEveryResearchEndpointExecutesKratosMiddleware(t *testing.T) {
	seen := map[string]int{}
	server := kratoshttp.NewServer(kratoshttp.Middleware(func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, request any) (any, error) {
			serverTransport, ok := transport.FromServerContext(ctx)
			if !ok {
				t.Fatal("middleware context is missing Kratos server transport")
			}
			seen[serverTransport.Operation()]++
			return handler(ctx, request)
		}
	}))
	RegisterResearchHTTPServer(server, stubResearchHTTPServer{})

	for _, request := range []struct {
		path      string
		operation string
	}{
		{path: "/api/miniapp/v1/research/themes", operation: OperationListResearchThemes},
		{path: "/api/miniapp/v1/research/themes/theme-id", operation: OperationGetResearchTheme},
		{path: "/api/miniapp/v1/research/themes/theme-id/reasoning-trees", operation: OperationListResearchThemeReasoningTrees},
		{path: "/api/miniapp/v1/research/themes/theme-id/reasoning-trees/anchor-id", operation: OperationGetResearchThemeReasoningTree},
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, request.path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", request.path, response.Code, http.StatusOK)
		}
		if seen[request.operation] != 1 {
			t.Fatalf("GET %s middleware calls for %q = %d, want 1", request.path, request.operation, seen[request.operation])
		}
	}
}

type stubResearchHTTPServer struct{}

func (stubResearchHTTPServer) ListResearchThemes(context.Context, *ListResearchThemesRequest) (*ResearchThemeListResponse, error) {
	return &ResearchThemeListResponse{}, nil
}

func (stubResearchHTTPServer) GetResearchTheme(context.Context, *GetResearchThemeRequest) (*ResearchThemeDetailResponse, error) {
	return &ResearchThemeDetailResponse{}, nil
}

func (stubResearchHTTPServer) ListResearchThemeReasoningTrees(context.Context, *ListResearchThemeReasoningTreesRequest) (*ResearchReasoningTreeListResponse, error) {
	return &ResearchReasoningTreeListResponse{}, nil
}

func (stubResearchHTTPServer) GetResearchThemeReasoningTree(context.Context, *GetResearchThemeReasoningTreeRequest) (*ResearchReasoningTreeDetailResponse, error) {
	return &ResearchReasoningTreeDetailResponse{}, nil
}
