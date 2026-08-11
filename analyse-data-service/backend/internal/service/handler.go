package service

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/analyse-data-service/backend/api/data/v1"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/runtimehealth"
)

const (
	Namespace = v1.APIPrefix

	ScopeAdminRead = "data.admin.read"
)

type RuntimeHealthService interface {
	Get(context.Context) runtimehealth.Result
}

type Dependencies struct {
	RuntimeHealth RuntimeHealthService
}

type DataService struct {
	dependencies Dependencies
}

func NewDataService(dependencies Dependencies) *DataService {
	return &DataService{dependencies: dependencies}
}

var _ v1.DataHTTPServer = (*DataService)(nil)
