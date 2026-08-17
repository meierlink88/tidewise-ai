package concept

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate = "data.v1.createConcept"
	OperationList   = "data.v1.listConcepts"
	OperationGet    = "data.v1.getConcept"
	OperationUpdate = "data.v1.updateConcept"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationGet, OperationUpdate}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[Concept], error)
	List(context.Context, *ListRequest) (*v1.Response[ConceptList], error)
	Get(context.Context, *GetRequest) (*v1.Response[Concept], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[Concept], error)
}

type CreateRequest struct {
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	ConceptType  string   `json:"concept_type"`
	Definition   string   `json:"definition"`
	ReviewStatus string   `json:"review_status"`
}

type ListRequest struct{}
type GetRequest struct{ ConceptID string }

type UpdateRequest struct {
	ConceptID    string   `json:"-"`
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	ConceptType  string   `json:"concept_type"`
	Definition   string   `json:"definition"`
	ReviewStatus string   `json:"review_status"`
}

type Concept struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	ConceptType  string   `json:"concept_type"`
	Definition   string   `json:"definition"`
	ReviewStatus string   `json:"review_status"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

type ConceptList struct {
	Items []Concept `json:"items"`
}
