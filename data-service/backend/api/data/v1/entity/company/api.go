package company

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	ProjectionSchemaVersion = "company-projection-snapshot.v1"
	OperationList           = "data.v1.listCompanies"

	ErrorTimeout           = "COMPANY_PROJECTION_TIMEOUT"
	ErrorInvalidRequest    = "INVALID_REQUEST"
	ErrorSnapshotChanged   = "COMPANY_PROJECTION_SNAPSHOT_CHANGED"
	ErrorPersistenceFailed = "COMPANY_PROJECTION_PERSISTENCE_FAILED"
	ErrorFailed            = "COMPANY_PROJECTION_FAILED"
)

func BusinessOperations() []string { return []string{OperationList} }

type Service interface {
	List(context.Context, *ListRequest) (*v1.Response[CompanyProjectionPage], error)
}

type ListRequest struct {
	PageSize string
	Cursor   string
}

type CompanyProjectionPage struct {
	SchemaVersion string    `json:"schema_version"`
	SnapshotID    string    `json:"snapshot_id"`
	Items         []Company `json:"items"`
	NextCursor    *string   `json:"next_cursor"`
}

type Company struct {
	ID                    string                `json:"id"`
	Code                  string                `json:"code"`
	Name                  string                `json:"name"`
	NameEn                *string               `json:"name_en"`
	LegalName             *string               `json:"legal_name"`
	Aliases               []string              `json:"aliases"`
	RegistrationCountryID *string               `json:"registration_country_id"`
	OperatingArea         *string               `json:"operating_area"`
	HeadquartersCity      *string               `json:"headquarters_city"`
	FoundingDate          *string               `json:"founding_date"`
	IPODate               *string               `json:"ipo_date"`
	LegalForm             *string               `json:"legal_form"`
	OwnershipType         *string               `json:"ownership_type"`
	StrategicPositioning  *string               `json:"strategic_positioning"`
	Description           *string               `json:"description"`
	Status                string                `json:"status"`
	CreatedAt             string                `json:"created_at"`
	UpdatedAt             string                `json:"updated_at"`
	IndustryLinks         []CompanyIndustryLink `json:"industry_links"`
}

type CompanyIndustryLink struct {
	ID         string `json:"id"`
	CompanyID  string `json:"company_id"`
	IndustryID string `json:"industry_id"`
	CreatedAt  string `json:"created_at"`
}
