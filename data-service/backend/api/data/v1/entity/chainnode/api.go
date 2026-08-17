package chainnode

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate = "data.v1.createChainNode"
	OperationList   = "data.v1.listChainNodes"
	OperationGet    = "data.v1.getChainNode"
	OperationUpdate = "data.v1.updateChainNode"

	ErrorTimeout           = "CHAIN_NODE_TIMEOUT"
	ErrorInvalidRequest    = "INVALID_REQUEST"
	ErrorInvalid           = "CHAIN_NODE_INVALID"
	ErrorNotFound          = "CHAIN_NODE_NOT_FOUND"
	ErrorConflict          = "CHAIN_NODE_CONFLICT"
	ErrorPersistenceFailed = "CHAIN_NODE_PERSISTENCE_FAILED"
	ErrorFailed            = "CHAIN_NODE_FAILED"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationGet, OperationUpdate}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[ChainNode], error)
	List(context.Context, *ListRequest) (*v1.Response[ChainNodeList], error)
	Get(context.Context, *GetRequest) (*v1.Response[ChainNode], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[ChainNode], error)
}

type CreateRequest struct {
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	Definition   string   `json:"definition"`
	ReviewStatus string   `json:"review_status"`
}
type ListRequest struct {
	PageSize string
	Cursor   string
}
type GetRequest struct{ ChainNodeID string }
type UpdateRequest struct {
	ChainNodeID  string   `json:"-"`
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	Definition   string   `json:"definition"`
	ReviewStatus string   `json:"review_status"`
}
type ChainNode struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Aliases      []string `json:"aliases"`
	Definition   string   `json:"definition"`
	ReviewStatus string   `json:"review_status"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}
type ChainNodeList struct {
	Items      []ChainNode `json:"items"`
	NextCursor *string     `json:"next_cursor"`
}
