package source

import (
	"context"
	"encoding/json"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate   = "data.v1.createSource"
	OperationList     = "data.v1.listSources"
	OperationUpdate   = "data.v1.updateSource"
	OperationDelete   = "data.v1.deleteSource"
	OperationSnapshot = "data.v1.getSourceSnapshot"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationUpdate, OperationDelete, OperationSnapshot}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[Source], error)
	List(context.Context) (*v1.Response[SourceList], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[Source], error)
	Delete(context.Context, *DeleteRequest) (*v1.Response[DeleteResult], error)
	Snapshot(context.Context) (*v1.Response[SourceSnapshot], error)
}

type CreateRequest struct {
	Code               string          `json:"code"`
	Name               string          `json:"name"`
	Enabled            bool            `json:"enabled"`
	Endpoint           string          `json:"endpoint"`
	AppKey             *string         `json:"app_key"`
	Config             json.RawMessage `json:"config"`
	Priority           int             `json:"priority"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxResults         int             `json:"max_results"`
	DefaultSourceLevel string          `json:"default_source_level"`
}

type UpdateRequest struct {
	SourceID           string          `json:"-"`
	Name               string          `json:"name"`
	AdapterKey         string          `json:"adapter_key"`
	Enabled            bool            `json:"enabled"`
	Endpoint           string          `json:"endpoint"`
	AppKey             *string         `json:"app_key"`
	Config             json.RawMessage `json:"config"`
	Priority           int             `json:"priority"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxResults         int             `json:"max_results"`
	DefaultSourceLevel string          `json:"default_source_level"`
}

type DeleteRequest struct{ SourceID string }

type Source struct {
	ID                 string          `json:"id"`
	Code               string          `json:"code"`
	Name               string          `json:"name"`
	OwnershipType      string          `json:"ownership_type"`
	ChannelType        string          `json:"channel_type"`
	AdapterKey         string          `json:"adapter_key"`
	Enabled            bool            `json:"enabled"`
	Endpoint           string          `json:"endpoint"`
	AppKey             *string         `json:"app_key"`
	Config             json.RawMessage `json:"config"`
	Priority           int             `json:"priority"`
	TimeoutSeconds     int             `json:"timeout_seconds"`
	MaxResults         int             `json:"max_results"`
	DefaultSourceLevel string          `json:"default_source_level"`
	CreatedAt          string          `json:"created_at"`
	UpdatedAt          string          `json:"updated_at"`
}

type SourceList struct {
	Sources []Source `json:"sources"`
}

type SourceSnapshot struct {
	Sources []Source `json:"sources"`
}

type DeleteResult struct {
	ID string `json:"id"`
}
