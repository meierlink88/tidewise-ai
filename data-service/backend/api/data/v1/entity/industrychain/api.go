package industrychain

import (
	"context"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate        = "data.v1.createIndustryChain"
	OperationList          = "data.v1.listIndustryChains"
	OperationGet           = "data.v1.getIndustryChain"
	OperationUpdate        = "data.v1.updateIndustryChain"
	ErrorTimeout           = "INDUSTRY_CHAIN_TIMEOUT"
	ErrorInvalidRequest    = "INVALID_REQUEST"
	ErrorInvalid           = "INDUSTRY_CHAIN_INVALID"
	ErrorReferenceInvalid  = "INDUSTRY_CHAIN_REFERENCE_INVALID"
	ErrorNotFound          = "INDUSTRY_CHAIN_NOT_FOUND"
	ErrorConflict          = "INDUSTRY_CHAIN_CONFLICT"
	ErrorPersistenceFailed = "INDUSTRY_CHAIN_PERSISTENCE_FAILED"
	ErrorFailed            = "INDUSTRY_CHAIN_FAILED"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationGet, OperationUpdate}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[IndustryChain], error)
	List(context.Context, *ListRequest) (*v1.Response[IndustryChainList], error)
	Get(context.Context, *GetRequest) (*v1.Response[IndustryChain], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[IndustryChain], error)
}
type CreateRequest struct {
	Name                     string   `json:"name"`
	Aliases                  []string `json:"aliases"`
	Scope                    string   `json:"scope"`
	TargetOutput             string   `json:"target_output"`
	EndUse                   string   `json:"end_use"`
	Geography                string   `json:"geography"`
	PrimaryCountryID         *string  `json:"primary_country_id"`
	AsOfDate                 string   `json:"as_of_date"`
	ReviewStatus             string   `json:"review_status"`
	ReviewNote               *string  `json:"review_note"`
	TechnologyRouteQualifier *string  `json:"technology_route_qualifier"`
	ObservableVariables      []string `json:"observable_variables"`
}
type ListRequest struct {
	PageSize string
	Cursor   string
}
type GetRequest struct{ IndustryChainID string }
type UpdateRequest struct {
	IndustryChainID string `json:"-"`
	CreateRequest
}
type IndustryChain struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	Aliases                  []string `json:"aliases"`
	Scope                    string   `json:"scope"`
	TargetOutput             string   `json:"target_output"`
	EndUse                   string   `json:"end_use"`
	Geography                string   `json:"geography"`
	PrimaryCountryID         *string  `json:"primary_country_id"`
	AsOfDate                 string   `json:"as_of_date"`
	ReviewStatus             string   `json:"review_status"`
	ReviewNote               *string  `json:"review_note"`
	TechnologyRouteQualifier *string  `json:"technology_route_qualifier"`
	ObservableVariables      []string `json:"observable_variables"`
	CreatedAt                string   `json:"created_at"`
	UpdatedAt                string   `json:"updated_at"`
}
type IndustryChainList struct {
	Items      []IndustryChain `json:"items"`
	NextCursor *string         `json:"next_cursor"`
}
