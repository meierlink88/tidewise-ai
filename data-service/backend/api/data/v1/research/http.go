package research

import (
	"context"
	"encoding/json"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	ReadExecutionBudget  = 5 * time.Second
	HeavyExecutionBudget = 15 * time.Second
)

func callWithBudget[T any](ctx kratoshttp.Context, operation string, budget time.Duration, request any, invoke func(context.Context) (*v1.Response[T], error)) error {
	return v1.Call(ctx, operation, request, func(callContext context.Context) (*v1.Response[T], error) {
		deadlineContext, cancel := context.WithTimeout(callContext, budget)
		defer cancel()
		return invoke(deadlineContext)
	})
}

func decodeResearchThemeImport(payload []byte) (*ResearchThemeImportRequest, error) {
	request := new(ResearchThemeImportRequest)
	if err := v1.DecodeStrictJSON(payload, researchThemeSnapshotImportShape(), request); err != nil {
		return nil, v1.NewPublicError(v1.StatusBadRequest, "INVALID_REQUEST", "request body is not valid for the Research Theme Aggregate V3 analyst_snapshot contract", map[string]any{
			"theme_key": researchThemeKeyAtPath(payload, v1.StrictJSONErrorPath(err)),
			"path":      v1.StrictJSONErrorPath(err),
		})
	}
	return request, nil
}

func researchThemeSnapshotImportShape() *v1.StrictJSONShape {
	evidenceIDs := v1.StrictJSONArray(v1.StrictJSONString())
	impact := v1.StrictJSONRequiredObject([]string{
		"node_key", "display_name", "relation_role", "impact_direction", "impact_summary", "display_order",
	}, map[string]*v1.StrictJSONShape{
		"node_key": v1.StrictJSONString(), "display_name": v1.StrictJSONString(), "relation_role": v1.StrictJSONString(),
		"impact_direction": v1.StrictJSONString(), "impact_summary": v1.StrictJSONNullableString(), "display_order": v1.StrictJSONScalar(),
	})
	themeEvent := v1.StrictJSONRequiredObject([]string{"event_id", "evidence_role", "supported_claim"}, map[string]*v1.StrictJSONShape{
		"event_id": v1.StrictJSONString(), "evidence_ids": evidenceIDs, "evidence_role": v1.StrictJSONString(),
		"supported_claim": v1.StrictJSONNullableString(),
	})
	theme := v1.StrictJSONRequiredObject([]string{
		"theme_key", "title", "one_line_conclusion", "conclusion_direction", "impact_strength",
		"attention_level", "conclusion_status", "transmission_stage", "investment_guidance_action",
		"investment_guidance_summary", "time_horizon_category", "time_horizon_summary",
		"transmission_summary", "checkpoint_summary", "risk_summary", "impacts", "events",
	}, map[string]*v1.StrictJSONShape{
		"theme_key": v1.StrictJSONString(), "title": v1.StrictJSONString(), "one_line_conclusion": v1.StrictJSONString(),
		"conclusion_direction": v1.StrictJSONString(), "impact_strength": v1.StrictJSONString(),
		"attention_level": v1.StrictJSONNullableString(), "conclusion_status": v1.StrictJSONNullableString(),
		"transmission_stage": v1.StrictJSONString(), "investment_guidance_action": v1.StrictJSONString(),
		"investment_guidance_summary": v1.StrictJSONString(), "time_horizon_category": v1.StrictJSONString(),
		"time_horizon_summary": v1.StrictJSONNullableString(), "transmission_summary": v1.StrictJSONNullableString(),
		"checkpoint_summary": v1.StrictJSONNullableString(), "risk_summary": v1.StrictJSONNullableString(),
		"impacts": v1.StrictJSONArray(impact), "events": v1.StrictJSONArray(themeEvent),
	})
	checkpoint := v1.StrictJSONRequiredObject([]string{"type", "summary"}, map[string]*v1.StrictJSONShape{
		"type": v1.StrictJSONString(), "summary": v1.StrictJSONString(),
	})
	treeEvent := v1.StrictJSONRequiredObject([]string{"event_id", "evidence_role", "display_order"}, map[string]*v1.StrictJSONShape{
		"event_id": v1.StrictJSONString(), "evidence_ids": evidenceIDs, "evidence_role": v1.StrictJSONString(), "display_order": v1.StrictJSONScalar(),
	})
	signal := v1.StrictJSONRequiredObject([]string{
		"signal_key", "display_summary", "role", "display_order", "variable_name", "direction",
	}, map[string]*v1.StrictJSONShape{
		"signal_key": v1.StrictJSONString(), "display_summary": v1.StrictJSONString(), "role": v1.StrictJSONString(),
		"display_order": v1.StrictJSONScalar(), "variable_name": v1.StrictJSONNullableString(), "direction": v1.StrictJSONNullableString(),
	})
	incoming := v1.StrictJSONRequiredObject([]string{"title", "mechanism", "condition_summary"}, map[string]*v1.StrictJSONShape{
		"title": v1.StrictJSONNullableString(), "mechanism": v1.StrictJSONString(), "condition_summary": v1.StrictJSONNullableString(),
	})
	node := v1.StrictJSONRequiredObject([]string{
		"node_key", "display_name", "position", "state_summary", "impact_direction", "impact_strength",
		"impact_summary", "reasoning_basis_summary", "evidence_gap_summary", "incoming_transmission", "signals",
	}, map[string]*v1.StrictJSONShape{
		"node_key": v1.StrictJSONString(), "display_name": v1.StrictJSONString(), "position": v1.StrictJSONScalar(),
		"state_summary": v1.StrictJSONNullableString(), "impact_direction": v1.StrictJSONString(), "impact_strength": v1.StrictJSONString(),
		"impact_summary": v1.StrictJSONNullableString(), "reasoning_basis_summary": v1.StrictJSONNullableString(),
		"evidence_gap_summary":  v1.StrictJSONNullableString(),
		"incoming_transmission": v1.StrictJSONNullable(incoming),
		"signals":               v1.StrictJSONArray(signal),
	})
	tree := v1.StrictJSONRequiredObject([]string{
		"tree_key", "display_name", "title", "display_order", "one_line_conclusion", "fact_summary",
		"transmission_summary", "impact_direction", "impact_strength", "impact_summary",
		"conclusion_boundary_summary", "support_summary", "counter_summary", "invalidation_conditions",
		"checkpoints", "events", "nodes",
	}, map[string]*v1.StrictJSONShape{
		"tree_key": v1.StrictJSONString(), "display_name": v1.StrictJSONString(), "title": v1.StrictJSONString(), "display_order": v1.StrictJSONScalar(),
		"one_line_conclusion": v1.StrictJSONString(), "fact_summary": v1.StrictJSONNullableString(),
		"transmission_summary": v1.StrictJSONNullableString(), "impact_direction": v1.StrictJSONString(),
		"impact_strength": v1.StrictJSONString(), "impact_summary": v1.StrictJSONNullableString(),
		"conclusion_boundary_summary": v1.StrictJSONNullableString(), "support_summary": v1.StrictJSONNullableString(),
		"counter_summary": v1.StrictJSONNullableString(), "invalidation_conditions": v1.StrictJSONArray(v1.StrictJSONString()),
		"checkpoints": v1.StrictJSONArray(checkpoint), "events": v1.StrictJSONArray(treeEvent), "nodes": v1.StrictJSONArray(node),
	})
	return v1.StrictJSONRequiredObject([]string{
		"publication_mode", "analysis_batch_id", "analysis_as_of", "discovery_window_start",
		"discovery_window_end", "theme", "reasoning_trees",
	}, map[string]*v1.StrictJSONShape{
		"publication_mode": v1.StrictJSONString(), "analysis_batch_id": v1.StrictJSONString(), "analysis_as_of": v1.StrictJSONString(),
		"discovery_window_start": v1.StrictJSONString(), "discovery_window_end": v1.StrictJSONString(),
		"theme": theme, "reasoning_trees": v1.StrictJSONArray(tree),
	})
}

func researchThemeKeyAtPath(payload []byte, path string) string {
	var hint struct {
		Theme struct {
			ThemeKey string `json:"theme_key"`
		} `json:"theme"`
	}
	_ = json.Unmarshal(payload, &hint)
	return hint.Theme.ThemeKey
}

func searchResearchGraphHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request, err := v1.DecodeStrictJSONBody[ResearchGraphSearchRequest](ctx)
		if err != nil {
			return err
		}
		return callWithBudget(
			ctx,
			OperationSearchResearchGraph, HeavyExecutionBudget,
			request,
			func(callContext context.Context) (*v1.Response[ResearchGraphSearchResult], error) {
				return application.SearchResearchGraph(callContext, request)
			},
		)
	}
}

func researchThemeImportHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		payload, err := v1.ReadImportPayload(ctx)
		if err != nil {
			return err
		}
		request, err := decodeResearchThemeImport(payload)
		if err != nil {
			return err
		}
		return callWithBudget(ctx, OperationPublishResearchTheme, HeavyExecutionBudget, request, func(callContext context.Context) (*v1.Response[ResearchThemeImportResult], error) {
			return application.PublishResearchTheme(callContext, request)
		})
	}
}

func listResearchThemesHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ListResearchThemesRequest{
			WindowHours:   ctx.Query().Get("window_hours"),
			PublishedFrom: ctx.Query().Get("published_from"),
			PublishedTo:   ctx.Query().Get("published_to"),
			Limit:         ctx.Query().Get("limit"),
			Cursor:        ctx.Query().Get("cursor"),
		}
		return callWithBudget(ctx, OperationListResearchThemes, ReadExecutionBudget, request, func(callContext context.Context) (*v1.Response[ResearchThemePage], error) {
			return application.ListResearchThemes(callContext, request)
		})
	}
}

func getResearchThemeHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &GetResearchThemeRequest{ThemeID: ctx.Vars().Get("theme_id"), WindowHours: ctx.Query().Get("window_hours")}
		return callWithBudget(ctx, OperationGetResearchTheme, ReadExecutionBudget, request, func(callContext context.Context) (*v1.Response[ResearchThemeDetail], error) {
			return application.GetResearchTheme(callContext, request)
		})
	}
}

func listReasoningTreesHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeListRequest{ThemeID: ctx.Vars().Get("theme_id"), HasQuery: ctx.Request().URL.RawQuery != ""}
		return callWithBudget(ctx, OperationListResearchThemeReasoningTrees, ReadExecutionBudget, request, func(callContext context.Context) (*v1.Response[ResearchReasoningTreeList], error) {
			return application.ListResearchReasoningTrees(callContext, request)
		})
	}
}

func getReasoningTreeHandler(application Service) kratoshttp.HandlerFunc {
	return func(ctx kratoshttp.Context) error {
		request := &ReasoningTreeDetailRequest{
			ThemeID: ctx.Vars().Get("theme_id"), ReasoningTreeID: ctx.Vars().Get("reasoning_tree_id"), HasQuery: ctx.Request().URL.RawQuery != "",
		}
		return callWithBudget(ctx, OperationGetResearchThemeReasoningTree, ReadExecutionBudget, request, func(callContext context.Context) (*v1.Response[ResearchReasoningTreeDetail], error) {
			return application.GetResearchReasoningTree(callContext, request)
		})
	}
}
func RegisterHTTPServer(server *kratoshttp.Server, application Service) {
	router := server.Route(v1.APIPrefix)
	router.POST("/research-theme-imports", researchThemeImportHandler(application))
	router.GET("/research/themes", listResearchThemesHandler(application))
	router.GET("/research/themes/{theme_id}", getResearchThemeHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees", listReasoningTreesHandler(application))
	router.GET("/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}", getReasoningTreeHandler(application))
	router.POST("/research-graph:search", searchResearchGraphHandler(application))
}
