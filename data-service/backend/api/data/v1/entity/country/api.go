package country

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate         = "data.v1.createCountry"
	OperationList           = "data.v1.listCountries"
	OperationGet            = "data.v1.getCountry"
	OperationUpdate         = "data.v1.updateCountry"
	OperationReplaceRegions = "data.v1.replaceCountryRegions"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationGet, OperationUpdate, OperationReplaceRegions}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[Country], error)
	List(context.Context, *ListRequest) (*v1.Response[CountryList], error)
	Get(context.Context, *GetRequest) (*v1.Response[Country], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[Country], error)
	ReplaceRegions(context.Context, *ReplaceRegionsRequest) (*v1.Response[Country], error)
}

type CreateRequest struct {
	ID                   string  `json:"id"`
	Code                 string  `json:"code"`
	Name                 string  `json:"name"`
	NameEn               string  `json:"name_en"`
	StrategicPositioning *string `json:"strategic_positioning"`
	KeyResources         *string `json:"key_resources"`
}

type ListRequest struct{ RegionID string }
type GetRequest struct{ CountryID string }

type UpdateRequest struct {
	CountryID            string  `json:"-"`
	Name                 string  `json:"name"`
	NameEn               string  `json:"name_en"`
	StrategicPositioning *string `json:"strategic_positioning"`
	KeyResources         *string `json:"key_resources"`
}

type ReplaceRegionsRequest struct {
	CountryID string   `json:"-"`
	RegionIDs []string `json:"region_ids"`
}

type Region struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	NameEn     string `json:"name_en"`
	RegionType string `json:"region_type"`
}

type Country struct {
	ID                   string   `json:"id"`
	Code                 string   `json:"code"`
	Name                 string   `json:"name"`
	NameEn               string   `json:"name_en"`
	StrategicPositioning *string  `json:"strategic_positioning"`
	KeyResources         *string  `json:"key_resources"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
	Regions              []Region `json:"regions"`
}

type CountryList struct {
	Items []Country `json:"items"`
}
