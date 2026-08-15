package organization

import (
	"context"
	"time"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const (
	OperationCreate            = "data.v1.createOrganization"
	OperationList              = "data.v1.listOrganizations"
	OperationGet               = "data.v1.getOrganization"
	OperationUpdate            = "data.v1.updateOrganization"
	OperationReplaceDomainTags = "data.v1.replaceOrganizationDomainTags"
	OperationGetCatalog        = "data.v1.getOrganizationCatalog"
	OperationListMembers       = "data.v1.listOrganizationMembers"
	OperationCreateMember      = "data.v1.createOrganizationMember"
	OperationUpdateMember      = "data.v1.updateOrganizationMember"
	OperationDeleteMember      = "data.v1.deleteOrganizationMember"
)

func BusinessOperations() []string {
	return []string{OperationCreate, OperationList, OperationGet, OperationUpdate, OperationReplaceDomainTags, OperationGetCatalog, OperationListMembers, OperationCreateMember, OperationUpdateMember, OperationDeleteMember}
}

type Service interface {
	Create(context.Context, *CreateRequest) (*v1.Response[Organization], error)
	List(context.Context, *ListRequest) (*v1.Response[OrganizationList], error)
	Get(context.Context, *GetRequest) (*v1.Response[Organization], error)
	Update(context.Context, *UpdateRequest) (*v1.Response[Organization], error)
	ReplaceDomainTags(context.Context, *ReplaceDomainTagsRequest) (*v1.Response[Organization], error)
	GetCatalog(context.Context, *CatalogRequest) (*v1.Response[Catalog], error)
	ListMembers(context.Context, *ListMembersRequest) (*v1.Response[MemberList], error)
	CreateMember(context.Context, *CreateMemberRequest) (*v1.Response[Member], error)
	UpdateMember(context.Context, *UpdateMemberRequest) (*v1.Response[Member], error)
	DeleteMember(context.Context, *DeleteMemberRequest) (*v1.Response[DeleteResult], error)
}

type CreateRequest struct {
	Code                      string  `json:"code"`
	Name                      string  `json:"name"`
	NameEn                    string  `json:"name_en"`
	RegionID                  *string `json:"region_id"`
	CategoryCode              string  `json:"category_code"`
	FunctionCode              string  `json:"function_code"`
	LegalEntityCode           *string `json:"legal_entity_code"`
	DominantPartyID           *string `json:"dominant_party_id"`
	BindingPowerLevel         *string `json:"binding_power_level"`
	InfluenceRating           *string `json:"influence_rating"`
	StrategicPositioning      *string `json:"strategic_positioning"`
	CoreImpactScope           *string `json:"core_impact_scope"`
	FoundingDocument          *string `json:"founding_document"`
	EstablishedDate           *Date   `json:"established_date"`
	HeadquartersCity          *string `json:"headquarters_city"`
	HeadquartersCountryID     *string `json:"headquarters_country_id"`
	HeadquartersSubdivisionID *string `json:"headquarters_subdivision_id"`
	Description               *string `json:"description"`
}

type ListRequest struct {
	CategoryCode string
	FunctionCode string
	RegionID     string
	CountryID    string
	AsOf         string
}

type GetRequest struct{ OrganizationID string }

type UpdateRequest struct {
	OrganizationID            string    `json:"-"`
	Name                      string    `json:"name"`
	NameEn                    string    `json:"name_en"`
	RegionID                  *string   `json:"region_id"`
	CategoryCode              string    `json:"category_code"`
	FunctionCode              string    `json:"function_code"`
	LegalEntityCode           *string   `json:"legal_entity_code"`
	DominantPartyID           *string   `json:"dominant_party_id"`
	BindingPowerLevel         *string   `json:"binding_power_level"`
	InfluenceRating           *string   `json:"influence_rating"`
	StrategicPositioning      *string   `json:"strategic_positioning"`
	CoreImpactScope           *string   `json:"core_impact_scope"`
	FoundingDocument          *string   `json:"founding_document"`
	EstablishedDate           *Date     `json:"established_date"`
	HeadquartersCity          *string   `json:"headquarters_city"`
	HeadquartersCountryID     *string   `json:"headquarters_country_id"`
	HeadquartersSubdivisionID *string   `json:"headquarters_subdivision_id"`
	Description               *string   `json:"description"`
	DomainTagCodes            *[]string `json:"domain_tag_codes"`
}

type ReplaceDomainTagsRequest struct {
	OrganizationID string   `json:"-"`
	DomainTagCodes []string `json:"domain_tag_codes"`
}

type CatalogRequest struct{}

type DomainTag struct {
	Code         string `json:"code"`
	FunctionCode string `json:"function_code"`
	NameZh       string `json:"name_zh"`
}

type Catalog struct {
	Categories []CatalogTerm `json:"categories"`
	Functions  []CatalogTerm `json:"functions"`
	DomainTags []DomainTag   `json:"domain_tags"`
}

type ListMembersRequest struct{ OrganizationID, AsOf string }

type CreateMemberRequest struct {
	OrganizationID string `json:"-"`
	CountryID      string `json:"country_id"`
	MembershipType string `json:"membership_type"`
	EffectiveDate  *Date  `json:"effective_date"`
	ExpiryDate     *Date  `json:"expiry_date"`
}

type UpdateMemberRequest struct {
	OrganizationID string `json:"-"`
	MemberID       string `json:"-"`
	CountryID      string `json:"country_id"`
	MembershipType string `json:"membership_type"`
	EffectiveDate  *Date  `json:"effective_date"`
	ExpiryDate     *Date  `json:"expiry_date"`
}

type DeleteMemberRequest struct {
	OrganizationID string
	MemberID       string
}

type Member struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	CountryID      string  `json:"country_id"`
	MembershipType string  `json:"membership_type"`
	EffectiveDate  *string `json:"effective_date"`
	ExpiryDate     *string `json:"expiry_date"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type MemberList struct {
	Items []Member `json:"items"`
}
type DeleteResult struct {
	Deleted bool `json:"deleted"`
}

type CatalogTerm struct {
	Code   string `json:"code"`
	NameZh string `json:"name_zh"`
}

type Organization struct {
	ID                        string      `json:"id"`
	Code                      string      `json:"code"`
	Name                      string      `json:"name"`
	NameEn                    string      `json:"name_en"`
	RegionID                  *string     `json:"region_id"`
	Category                  CatalogTerm `json:"category"`
	Function                  CatalogTerm `json:"function"`
	LegalEntityCode           *string     `json:"legal_entity_code"`
	DominantPartyID           *string     `json:"dominant_party_id"`
	BindingPowerLevel         *string     `json:"binding_power_level"`
	InfluenceRating           *string     `json:"influence_rating"`
	StrategicPositioning      *string     `json:"strategic_positioning"`
	CoreImpactScope           *string     `json:"core_impact_scope"`
	FoundingDocument          *string     `json:"founding_document"`
	EstablishedDate           *string     `json:"established_date"`
	HeadquartersCity          *string     `json:"headquarters_city"`
	HeadquartersCountryID     *string     `json:"headquarters_country_id"`
	HeadquartersSubdivisionID *string     `json:"headquarters_subdivision_id"`
	Description               *string     `json:"description"`
	DomainTags                []DomainTag `json:"domain_tags"`
	CreatedAt                 string      `json:"created_at"`
	UpdatedAt                 string      `json:"updated_at"`
}

type OrganizationList struct {
	Items []Organization `json:"items"`
}

// Date is a strict API calendar date, separate from timestamp fields.
type Date struct{ time.Time }

func (d *Date) UnmarshalJSON(value []byte) error {
	parsed, err := time.Parse(`"2006-01-02"`, string(value))
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}
