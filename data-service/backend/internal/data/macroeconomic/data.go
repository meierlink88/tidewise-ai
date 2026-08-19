// Package macroeconomic persists independent macroeconomic narrative blueprints.
package macroeconomic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type MacroType string

const (
	MacroTypeMonetary     MacroType = "MONETARY"
	MacroTypeFiscal       MacroType = "FISCAL"
	MacroTypeTradePolicy  MacroType = "TRADE_POLICY"
	MacroTypeRegulatory   MacroType = "REGULATORY"
	MacroTypeDataEconomic MacroType = "DATA_ECONOMIC"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusDormant  Status = "DORMANT"
	StatusArchived Status = "ARCHIVED"
)

var (
	ErrInvalidMacroEconomic = errors.New("invalid MacroEconomic")
	ErrNotFound             = errors.New("MacroEconomic not found")
	ErrPersistence          = errors.New("MacroEconomic persistence failed")
)

type CreateInput struct {
	Name        string
	NameEn      string
	MacroType   MacroType
	Description string
	Status      Status
}

type UpdateInput struct {
	ID          string
	Name        string
	NameEn      string
	MacroType   MacroType
	Description string
	Status      Status
}

type MacroEconomic struct {
	ID          string
	Name        string
	NameEn      string
	MacroType   MacroType
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Filter struct {
	MacroType *MacroType
	Status    *Status
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("MacroEconomic database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (MacroEconomic, error) {
	if input.Status == "" {
		input.Status = StatusActive
	}
	if err := validateCreate(input); err != nil {
		return MacroEconomic{}, err
	}
	id, err := coreid.New(coreid.MacroEconomic)
	if err != nil {
		return MacroEconomic{}, ErrPersistence
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO macro_economics (id, name, name_en, macro_type, description, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING `+macroEconomicColumns,
		id, input.Name, input.NameEn, string(input.MacroType), input.Description, string(input.Status),
	)
	created, err := scanMacroEconomic(row)
	if err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (MacroEconomic, error) {
	if !coreid.Is(id, coreid.MacroEconomic) {
		return MacroEconomic{}, ErrInvalidMacroEconomic
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+macroEconomicColumns+` FROM macro_economics WHERE id = $1`, id)
	result, err := scanMacroEconomic(row)
	if err != nil {
		return MacroEconomic{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]MacroEconomic, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+macroEconomicColumns+`
FROM macro_economics
WHERE ($1::macro_economic_type IS NULL OR macro_type = $1::macro_economic_type)
  AND ($2::macro_economic_status IS NULL OR status = $2::macro_economic_status)
ORDER BY name_en ASC, name ASC, id ASC`, nullableMacroType(filter.MacroType), nullableStatus(filter.Status))
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]MacroEconomic, 0)
	for rows.Next() {
		item, err := scanMacroEconomic(rows)
		if err != nil {
			return nil, classifyReadError(err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, input UpdateInput) (MacroEconomic, error) {
	if err := validateUpdate(input); err != nil {
		return MacroEconomic{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE macro_economics
SET name = $2,
    name_en = $3,
    macro_type = $4,
    description = $5,
    status = $6,
    updated_at = now()
WHERE id = $1
RETURNING `+macroEconomicColumns,
		input.ID, input.Name, input.NameEn, string(input.MacroType), input.Description, string(input.Status),
	)
	updated, err := scanMacroEconomic(row)
	if err != nil {
		return MacroEconomic{}, classifyWriteError(err)
	}
	return updated, nil
}

const macroEconomicColumns = `
id, name, name_en, macro_type::text, description, status::text,
created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanMacroEconomic(row rowScanner) (MacroEconomic, error) {
	var result MacroEconomic
	var macroType, status string
	if err := row.Scan(
		&result.ID, &result.Name, &result.NameEn, &macroType,
		&result.Description, &status, &result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return MacroEconomic{}, err
	}
	result.MacroType = MacroType(macroType)
	result.Status = Status(status)
	if err := validateStored(result); err != nil {
		return MacroEconomic{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		strings.TrimSpace(input.Description) == "" ||
		!validMacroType(input.MacroType) || !validStatus(input.Status) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func validateUpdate(input UpdateInput) error {
	if !coreid.Is(input.ID, coreid.MacroEconomic) ||
		!validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		strings.TrimSpace(input.Description) == "" ||
		!validMacroType(input.MacroType) || !validStatus(input.Status) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.MacroType != nil && !validMacroType(*filter.MacroType) {
		return ErrInvalidMacroEconomic
	}
	if filter.Status != nil && !validStatus(*filter.Status) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func validateStored(input MacroEconomic) error {
	if !coreid.Is(input.ID, coreid.MacroEconomic) ||
		!validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		strings.TrimSpace(input.Description) == "" ||
		!validMacroType(input.MacroType) || !validStatus(input.Status) ||
		input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidMacroEconomic
	}
	return nil
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validMacroType(value MacroType) bool {
	switch value {
	case MacroTypeMonetary, MacroTypeFiscal, MacroTypeTradePolicy, MacroTypeRegulatory, MacroTypeDataEconomic:
		return true
	default:
		return false
	}
}

func validStatus(value Status) bool {
	switch value {
	case StatusActive, StatusDormant, StatusArchived:
		return true
	default:
		return false
	}
}

func nullableMacroType(value *MacroType) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableStatus(value *Status) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ErrPersistence
	}
	switch postgresError.Code {
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidMacroEconomic
	default:
		return ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if errors.Is(err, ErrInvalidMacroEconomic) {
		return fmt.Errorf("%w: invalid persisted MacroEconomic", ErrPersistence)
	}
	return ErrPersistence
}
