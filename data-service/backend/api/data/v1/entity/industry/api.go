package industry

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate = "data.v1.createIndustry"
	OperationList   = "data.v1.listIndustries"
	OperationGet    = "data.v1.getIndustry"
	OperationUpdate = "data.v1.updateIndustry"

	ErrorTimeout           = "INDUSTRY_TIMEOUT"
	ErrorInvalidRequest    = "INVALID_REQUEST"
	ErrorInvalid           = "INDUSTRY_INVALID"
	ErrorReferenceInvalid  = "INDUSTRY_REFERENCE_INVALID"
	ErrorNotFound          = "INDUSTRY_NOT_FOUND"
	ErrorConflict          = "INDUSTRY_CONFLICT"
	ErrorPersistenceFailed = "INDUSTRY_PERSISTENCE_FAILED"
	ErrorFailed            = "INDUSTRY_FAILED"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationGet, OperationUpdate}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[Industry], error)
	List(context.Context, *ListRequest) (*v1.Response[IndustryList], error)
	Get(context.Context, *GetRequest) (*v1.Response[Industry], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[Industry], error)
}

type CreateRequest struct {
	Name                 string   `json:"name"`
	Aliases              []string `json:"aliases"`
	ClassificationSystem string   `json:"classification_system"`
	IndustryCode         string   `json:"industry_code"`
	ParentIndustryID     *string  `json:"parent_industry_id"`
	HierarchyPathCodes   []string `json:"hierarchy_path_codes"`
	Definition           string   `json:"definition"`
	ReviewStatus         string   `json:"review_status"`
}

type ListRequest struct {
	PageSize string
	Cursor   string
}
type GetRequest struct{ IndustryID string }

type UpdateRequest struct {
	IndustryID         string   `json:"-"`
	Name               string   `json:"name"`
	Aliases            []string `json:"aliases"`
	ParentIndustryID   *string  `json:"parent_industry_id"`
	HierarchyPathCodes []string `json:"hierarchy_path_codes"`
	Definition         string   `json:"definition"`
	ReviewStatus       string   `json:"review_status"`
}

type Industry struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Aliases              []string `json:"aliases"`
	ClassificationSystem string   `json:"classification_system"`
	IndustryCode         string   `json:"industry_code"`
	ParentIndustryID     *string  `json:"parent_industry_id"`
	HierarchyPathCodes   []string `json:"hierarchy_path_codes"`
	Definition           string   `json:"definition"`
	ReviewStatus         string   `json:"review_status"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

type IndustryList struct {
	Items      []Industry `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}
