package report

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	reportapi "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/report"
	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
)

type UseCase interface {
	Publish(context.Context, string, string, reportbiz.Content) (reportbiz.PublicationResult, error)
	List(context.Context, reportbiz.ListRequest) (reportbiz.Page, error)
	GetHome(context.Context, string) (reportbiz.Home, error)
	GetLayer(context.Context, string, string) (reportbiz.Summary, reportbiz.Layer, []reportbiz.IndustryChainSummary, error)
	ListIndustryChains(context.Context, reportbiz.IndustryChainListRequest) (reportbiz.IndustryChainPage, error)
	GetIndustryChain(context.Context, string, string) (reportbiz.Summary, reportbiz.IndustryChain, error)
	ListEvidence(context.Context, string, reportbiz.ScopeType, string) ([]reportbiz.Evidence, error)
}

type Service struct{ useCase UseCase }

func NewService(useCase UseCase) (*Service, error) {
	if useCase == nil {
		return nil, errors.New("Report use case is required")
	}
	return &Service{useCase: useCase}, nil
}

func (s *Service) PublishReport(ctx context.Context, request *reportapi.PublicationRequest) (*v1.Response[reportapi.PublicationResult], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report publication is required")
	}
	if s == nil || s.useCase == nil {
		return nil, publicError(v1.StatusInternalServerError, reportapi.ErrorDataServiceNotReady, "Report service is unavailable")
	}
	var content reportbiz.Content
	if err := mapContract(request.Content, &content); err != nil {
		return nil, publicError(v1.StatusUnprocessableEntity, reportapi.ErrorInvalidRequest, "Report content timestamps are not RFC3339")
	}
	result, err := s.useCase.Publish(ctx, request.ContractVersion, request.PublisherReportID, content)
	if err != nil {
		return nil, publicationError(err)
	}
	status := v1.StatusCreated
	if result.Replayed {
		status = v1.StatusOK
	}
	return &v1.Response[reportapi.PublicationResult]{Status: status, Result: reportapi.PublicationResult{ReportID: result.Record.ID, ContentHash: result.ContentHash, PublishedAt: result.Record.PublishedAt.UTC().Format(time.RFC3339Nano), Replayed: result.Replayed}}, nil
}

func (s *Service) ListReports(ctx context.Context, request *reportapi.ListRequest) (*v1.Response[reportapi.Collection], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report query is required")
	}
	limit, err := parseLimit(request.Limit)
	if err != nil {
		return nil, err
	}
	from, err := optionalUTC(request.PublishedFrom)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "published_from must be a UTC RFC3339 timestamp")
	}
	to, err := optionalUTC(request.PublishedTo)
	if err != nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "published_to must be a UTC RFC3339 timestamp")
	}
	page, err := s.useCase.List(ctx, reportbiz.ListRequest{PublishedFrom: from, PublishedTo: to, Limit: limit, Cursor: request.Cursor})
	if err != nil {
		return nil, readError(err)
	}
	items := make([]reportapi.Summary, len(page.Items))
	for i, item := range page.Items {
		items[i] = apiSummary(item)
	}
	return &v1.Response[reportapi.Collection]{Status: v1.StatusOK, Result: reportapi.Collection{Items: items, NextCursor: page.NextCursor}}, nil
}

func (s *Service) GetReportHome(ctx context.Context, request *reportapi.ReportRequest) (*v1.Response[reportapi.Home], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report identity is required")
	}
	home, err := s.useCase.GetHome(ctx, request.ReportID)
	if err != nil {
		return nil, readError(err)
	}
	result := reportapi.Home{Report: apiSummary(home.Report)}
	if home.Geopolitics != nil {
		result.Geopolitics = &reportapi.LayerSnapshot{}
		if err := mapContract(home.Geopolitics, result.Geopolitics); err != nil {
			return nil, repositoryMappingError()
		}
	}
	if home.Macroeconomics != nil {
		result.Macroeconomics = &reportapi.LayerSnapshot{}
		if err := mapContract(home.Macroeconomics, result.Macroeconomics); err != nil {
			return nil, repositoryMappingError()
		}
	}
	return &v1.Response[reportapi.Home]{Status: v1.StatusOK, Result: result}, nil
}

func (s *Service) GetReportLayer(ctx context.Context, request *reportapi.LayerRequest) (*v1.Response[reportapi.LayerDetail], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report section identity is required")
	}
	summary, layer, related, err := s.useCase.GetLayer(ctx, request.ReportID, request.LayerKey)
	if err != nil {
		return nil, readError(err)
	}
	var apiLayer reportapi.Layer
	if err := mapContract(layer, &apiLayer); err != nil {
		return nil, repositoryMappingError()
	}
	apiRelated := make([]reportapi.IndustryChainSummary, len(related))
	for i, item := range related {
		if err := mapContract(item, &apiRelated[i]); err != nil {
			return nil, repositoryMappingError()
		}
	}
	return &v1.Response[reportapi.LayerDetail]{Status: v1.StatusOK, Result: reportapi.LayerDetail{Report: apiSummary(summary), Layer: apiLayer, RelatedIndustryChains: apiRelated}}, nil
}

func (s *Service) ListReportIndustryChains(ctx context.Context, request *reportapi.ChainListRequest) (*v1.Response[reportapi.IndustryChainCollection], error) {
	if request == nil || request.HasUnknownQuery {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report industry-chain query is invalid")
	}
	limit, err := parseLimit(request.Limit)
	if err != nil {
		return nil, err
	}
	page, err := s.useCase.ListIndustryChains(ctx, reportbiz.IndustryChainListRequest{ReportID: request.ReportID, Limit: limit, Cursor: request.Cursor})
	if err != nil {
		return nil, readError(err)
	}
	items := make([]reportapi.IndustryChainSummary, len(page.Items))
	for i, item := range page.Items {
		if err := mapContract(item, &items[i]); err != nil {
			return nil, repositoryMappingError()
		}
	}
	return &v1.Response[reportapi.IndustryChainCollection]{Status: v1.StatusOK, Result: reportapi.IndustryChainCollection{Items: items, NextCursor: page.NextCursor}}, nil
}

func (s *Service) GetReportIndustryChain(ctx context.Context, request *reportapi.ChainRequest) (*v1.Response[reportapi.IndustryChainDetail], error) {
	if request == nil {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "Report industry-chain identity is required")
	}
	summary, chain, err := s.useCase.GetIndustryChain(ctx, request.ReportID, request.ChainKey)
	if err != nil {
		return nil, readError(err)
	}
	var apiChain reportapi.IndustryChain
	if err := mapContract(chain, &apiChain); err != nil {
		return nil, repositoryMappingError()
	}
	return &v1.Response[reportapi.IndustryChainDetail]{Status: v1.StatusOK, Result: reportapi.IndustryChainDetail{Report: apiSummary(summary), IndustryChain: apiChain}}, nil
}

func (s *Service) ListReportEvidence(ctx context.Context, request *reportapi.EvidenceRequest) (*v1.Response[reportapi.EvidenceCollection], error) {
	if request == nil || request.HasUnknownQuery || strings.TrimSpace(request.ScopeType) == "" || strings.TrimSpace(request.ScopeKey) == "" {
		return nil, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "scope_type and scope_key are required")
	}
	values, err := s.useCase.ListEvidence(ctx, request.ReportID, reportbiz.ScopeType(request.ScopeType), request.ScopeKey)
	if err != nil {
		return nil, readError(err)
	}
	items := make([]reportapi.EvidenceItem, len(values))
	for i, item := range values {
		var publishedAt *string
		if item.PublishedAt != nil {
			formatted := item.PublishedAt.UTC().Format(time.RFC3339Nano)
			publishedAt = &formatted
		}
		items[i] = reportapi.EvidenceItem{EvidenceID: item.EvidenceID, Role: item.Role, DisplayOrder: item.DisplayOrder, PublishedAt: publishedAt, Summary: item.Summary, Keywords: item.Keywords}
	}
	return &v1.Response[reportapi.EvidenceCollection]{Status: v1.StatusOK, Result: reportapi.EvidenceCollection{ReportID: request.ReportID, ScopeType: request.ScopeType, ScopeKey: request.ScopeKey, Items: items}}, nil
}

func mapContract(source, target any) error {
	payload, err := json.Marshal(source)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func apiSummary(value reportbiz.Summary) reportapi.Summary {
	return reportapi.Summary{ID: value.ID, PublisherReportID: value.PublisherReportID, ReportType: value.ReportType, Title: value.Title, GenerationStatus: value.GenerationStatus, Simulation: value.Simulation, GeneratedAt: value.GeneratedAt.UTC().Format(time.RFC3339Nano), Timezone: value.Timezone, HasGeopolitics: value.HasGeopolitics, HasMacroeconomics: value.HasMacroeconomics, Statistics: reportapi.Statistics{EventCount: value.Statistics.EventCount, OrdinaryFactCount: value.Statistics.OrdinaryFactCount, SignalFactCount: value.Statistics.SignalFactCount, TransmissionHypothesisCount: value.Statistics.TransmissionHypothesisCount, GeopoliticAnchorCount: value.Statistics.GeopoliticAnchorCount, MacroeconomicAnchorCount: value.Statistics.MacroeconomicAnchorCount, SignaledChainNodeCount: value.Statistics.SignaledChainNodeCount, IndustryChainCount: value.Statistics.IndustryChainCount}, PublishedAt: value.PublishedAt.UTC().Format(time.RFC3339Nano)}
}

func parseLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return reportbiz.DefaultLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, "limit must be an integer")
	}
	return value, nil
}
func optionalUTC(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil || value.Location() != time.UTC {
		return nil, errors.New("timestamp must be UTC RFC3339")
	}
	value = value.UTC()
	return &value, nil
}

func publicationError(err error) error {
	if errors.Is(err, reportbiz.ErrPublicationConflict) {
		return publicError(v1.StatusConflict, reportapi.ErrorReportPublicationConflict, "publisher_report_id conflicts with another Report payload")
	}
	var validation *reportbiz.ValidationError
	if errors.As(err, &validation) {
		return publicError(v1.StatusUnprocessableEntity, reportapi.ErrorInvalidRequest, validation.Error())
	}
	var reference *reportbiz.ReferenceError
	if errors.As(err, &reference) {
		return publicError(v1.StatusUnprocessableEntity, reportapi.ErrorReportEvidenceReferenceInvalid, "Report publication contains an invalid reference")
	}
	return repositoryMappingError()
}
func readError(err error) error {
	var validation *reportbiz.ValidationError
	switch {
	case errors.As(err, &validation):
		return publicError(v1.StatusBadRequest, reportapi.ErrorInvalidRequest, validation.Error())
	case errors.Is(err, reportbiz.ErrReportNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportNotFound, "Report was not found")
	case errors.Is(err, reportbiz.ErrLayerNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportLayerNotFound, "Report section was not found")
	case errors.Is(err, reportbiz.ErrChainNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportIndustryChainNotFound, "Report industry chain was not found")
	case errors.Is(err, reportbiz.ErrEvidenceScopeNotFound):
		return publicError(v1.StatusNotFound, reportapi.ErrorReportEvidenceScopeNotFound, "Report Evidence scope was not found")
	default:
		return repositoryMappingError()
	}
}
func repositoryMappingError() error {
	return publicError(v1.StatusInternalServerError, reportapi.ErrorReportRepositoryFailure, "Report operation failed")
}
func publicError(status int, code, message string) error {
	return v1.NewPublicError(status, code, message, nil)
}
