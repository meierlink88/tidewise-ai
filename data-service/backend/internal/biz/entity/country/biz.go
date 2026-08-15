package country

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
)

var (
	ErrNotFound    = errors.New("Country not found")
	ErrConflict    = errors.New("Country conflict")
	ErrPersistence = errors.New("Country persistence failed")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

type ReferenceError struct {
	Field   string
	Message string
}

func (e *ReferenceError) Error() string { return e.Field + ": " + e.Message }

type Region struct {
	ID         string
	Code       string
	Name       string
	NameEn     string
	RegionType string
}

type Country struct {
	ID                   string
	Code                 string
	Name                 string
	NameEn               string
	StrategicPositioning *string
	KeyResources         *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Regions              []Region
}

type Update struct {
	Name                 string
	NameEn               string
	StrategicPositioning *string
	KeyResources         *string
}

type Repository interface {
	Create(context.Context, Country) (Country, error)
	Get(context.Context, string) (Country, error)
	List(context.Context, string) ([]Country, error)
	Update(context.Context, string, Update) (Country, error)
}

type Store interface {
	Repository
	RegionTransaction
}

type UseCase struct{ store Store }

func NewUseCase(store Store) (*UseCase, error) {
	if store == nil {
		return nil, errors.New("Country store is required")
	}
	return &UseCase{store: store}, nil
}

func (s *UseCase) Create(ctx context.Context, input Country) (Country, error) {
	if err := validateCountry(input); err != nil {
		return Country{}, err
	}
	return s.store.Create(ctx, cloneCountry(input))
}

func (s *UseCase) Get(ctx context.Context, id string) (Country, error) {
	if err := validateID(id); err != nil {
		return Country{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *UseCase) List(ctx context.Context, regionID string) ([]Country, error) {
	if regionID != "" && !validRegionID(regionID) {
		return nil, &ValidationError{Field: "region_id", Message: "must be a stable Region ID"}
	}
	return s.store.List(ctx, regionID)
}

func (s *UseCase) Update(ctx context.Context, id string, input Update) (Country, error) {
	if err := validateID(id); err != nil {
		return Country{}, err
	}
	if err := validateNamesAndOptional(input.Name, input.NameEn, input.StrategicPositioning, input.KeyResources); err != nil {
		return Country{}, err
	}
	return s.store.Update(ctx, id, cloneUpdate(input))
}

func (s *UseCase) ReplaceRegions(ctx context.Context, id string, regionIDs []string) (Country, error) {
	if err := validateID(id); err != nil {
		return Country{}, err
	}
	seen := make(map[string]struct{}, len(regionIDs))
	for index, regionID := range regionIDs {
		if !validRegionID(regionID) {
			return Country{}, &ValidationError{Field: fmt.Sprintf("region_ids[%d]", index), Message: "must be a stable Region ID"}
		}
		if _, duplicate := seen[regionID]; duplicate {
			return Country{}, &ValidationError{Field: fmt.Sprintf("region_ids[%d]", index), Message: "must be unique"}
		}
		seen[regionID] = struct{}{}
	}
	return s.store.ReplaceRegions(ctx, id, append([]string(nil), regionIDs...))
}

func validateCountry(input Country) error {
	if err := validateID(input.ID); err != nil {
		return err
	}
	if len(input.Code) != 2 {
		return &ValidationError{Field: "code", Message: "must be uppercase ISO 3166-1 alpha-2"}
	}
	for _, character := range input.Code {
		if character < 'A' || character > 'Z' {
			return &ValidationError{Field: "code", Message: "must be uppercase ISO 3166-1 alpha-2"}
		}
	}
	return validateNamesAndOptional(input.Name, input.NameEn, input.StrategicPositioning, input.KeyResources)
}

func validateID(id string) error {
	if !entitybiz.IsCountryID(id) {
		return &ValidationError{Field: "country_id", Message: "must equal " + entitybiz.CountryIDPrefix + " immediately followed by a canonical lowercase UUID"}
	}
	return nil
}

func validateNamesAndOptional(name, nameEn string, strategicPositioning, keyResources *string) error {
	if strings.TrimSpace(name) == "" || utf8.RuneCountInString(name) > 100 {
		return &ValidationError{Field: "name", Message: "must be nonblank and contain at most 100 characters"}
	}
	if strings.TrimSpace(nameEn) == "" || utf8.RuneCountInString(nameEn) > 100 {
		return &ValidationError{Field: "name_en", Message: "must be nonblank and contain at most 100 characters"}
	}
	for _, optional := range []struct {
		field string
		value *string
	}{{"strategic_positioning", strategicPositioning}, {"key_resources", keyResources}} {
		if optional.value != nil && strings.TrimSpace(*optional.value) == "" {
			return &ValidationError{Field: optional.field, Message: "must be nonblank when present"}
		}
	}
	return nil
}

func validRegionID(id string) bool {
	return entitybiz.IsRegionID(id)
}

func cloneCountry(input Country) Country {
	input.StrategicPositioning = cloneString(input.StrategicPositioning)
	input.KeyResources = cloneString(input.KeyResources)
	input.Regions = append([]Region(nil), input.Regions...)
	return input
}

func cloneUpdate(input Update) Update {
	input.StrategicPositioning = cloneString(input.StrategicPositioning)
	input.KeyResources = cloneString(input.KeyResources)
	return input
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
