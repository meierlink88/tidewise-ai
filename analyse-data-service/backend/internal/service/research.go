package service

import (
	"context"
	"errors"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
)

func (s *DataService) ListResearchThemes(ctx context.Context, request *v1.ListResearchThemesRequest) (*v1.Response[v1.ResearchThemePage], error) {
	publishedFrom, err := optionalUTC(request.PublishedFrom)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "published_from must be an RFC3339 UTC timestamp")
	}
	publishedTo, err := optionalUTC(request.PublishedTo)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "published_to must be an RFC3339 UTC timestamp")
	}
	window := 0
	if publishedFrom == nil && publishedTo == nil {
		window, err = v1.ParseBoundedInt(request.WindowHours, research.DefaultResearchWindowHours, research.MinResearchWindowHours, research.MaxResearchWindowHours, "window_hours")
		if err != nil {
			return nil, err
		}
	} else if request.WindowHours != "" {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "window_hours cannot be combined with published_from and published_to")
	}
	limit, err := v1.ParseBoundedInt(request.Limit, research.DefaultResearchLimit, 1, research.MaxResearchLimit, "limit")
	if err != nil {
		return nil, err
	}
	if s == nil || s.dependencies.Research == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.dependencies.Research.ListThemes(ctx, research.ResearchListRequest{
		WindowHours: window, PublishedFrom: publishedFrom, PublishedTo: publishedTo,
		Limit: limit, Cursor: request.Cursor,
	})
	if err != nil {
		return nil, researchError(err)
	}
	return &v1.Response[v1.ResearchThemePage]{Status: v1.StatusOK, Result: researchThemePageDTO(result)}, nil
}

func (s *DataService) GetResearchTheme(ctx context.Context, request *v1.GetResearchThemeRequest) (*v1.Response[v1.ResearchThemeDetail], error) {
	window, err := v1.ParseBoundedInt(request.WindowHours, research.DefaultResearchWindowHours, research.MinResearchWindowHours, research.MaxResearchWindowHours, "window_hours")
	if err != nil {
		return nil, err
	}
	if s == nil || s.dependencies.Research == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.dependencies.Research.GetTheme(ctx, request.ThemeID, research.ResearchDetailRequest{WindowHours: window})
	if err != nil {
		return nil, researchError(err)
	}
	return &v1.Response[v1.ResearchThemeDetail]{Status: v1.StatusOK, Result: researchThemeDetailDTO(result)}, nil
}

func (s *DataService) ListResearchReasoningTrees(ctx context.Context, request *v1.ReasoningTreeListRequest) (*v1.Response[v1.ResearchReasoningTreeList], error) {
	if request.HasQuery {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "reasoning tree list does not accept query parameters")
	}
	if s == nil || s.dependencies.Research == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.dependencies.Research.ListReasoningTrees(ctx, request.ThemeID)
	if err != nil {
		return nil, reasoningTreeError(err)
	}
	return &v1.Response[v1.ResearchReasoningTreeList]{Status: v1.StatusOK, Result: reasoningTreeListDTO(result)}, nil
}

func (s *DataService) GetResearchReasoningTree(ctx context.Context, request *v1.ReasoningTreeDetailRequest) (*v1.Response[v1.ResearchReasoningTreeDetail], error) {
	if request.HasQuery {
		return nil, publicError(v1.StatusBadRequest, "INVALID_REQUEST", "reasoning tree detail does not accept query parameters")
	}
	if s == nil || s.dependencies.Research == nil {
		return nil, publicError(v1.StatusInternalServerError, "DATA_SERVICE_NOT_READY", "research service is unavailable")
	}
	result, err := s.dependencies.Research.GetReasoningTree(ctx, request.ThemeID, request.ReasoningTreeID)
	if err != nil {
		return nil, reasoningTreeError(err)
	}
	return &v1.Response[v1.ResearchReasoningTreeDetail]{Status: v1.StatusOK, Result: reasoningTreeDetailDTO(result)}, nil
}

func researchError(err error) error {
	switch {
	case errors.Is(err, research.ErrInvalidRequest):
		return publicError(v1.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, research.ErrNotFound):
		return publicError(v1.StatusNotFound, "NOT_FOUND", "research aggregate was not found")
	default:
		return publicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "research aggregate failed")
	}
}

func reasoningTreeError(err error) error {
	switch {
	case errors.Is(err, research.ErrInvalidRequest):
		return publicError(v1.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, research.ErrThemeNotFound):
		return publicError(v1.StatusNotFound, "RESEARCH_THEME_NOT_FOUND", "research Theme was not found")
	case errors.Is(err, research.ErrReasoningTreesNotFound):
		return publicError(v1.StatusNotFound, "RESEARCH_REASONING_TREES_NOT_FOUND", "research Theme has no published reasoning trees")
	case errors.Is(err, research.ErrReasoningTreeNotFound):
		return publicError(v1.StatusNotFound, "RESEARCH_REASONING_TREE_NOT_FOUND", "research reasoning tree was not found for the Theme")
	case errors.Is(err, research.ErrReasoningTreeInvariantViolation):
		return publicError(v1.StatusInternalServerError, "RESEARCH_REASONING_TREE_INVARIANT_VIOLATION", "published research reasoning tree data is incomplete")
	default:
		return publicError(v1.StatusInternalServerError, "DATA_REPOSITORY_FAILURE", "research reasoning tree failed")
	}
}
