package industrychain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	industrychainbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industrychain"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("IndustryChain database is required")
	}
	return &Store{db: db}, nil
}

const industryChainColumns = `i.id, i.name, array_to_json(i.aliases), i.scope, i.target_output, i.end_use,
i.geography, i.primary_country_id, i.as_of_date, i.review_status, i.review_note,
i.technology_route_qualifier, array_to_json(i.observable_variables), i.created_at, i.updated_at,
(i.primary_country_id IS NULL OR EXISTS (SELECT 1 FROM countries country WHERE country.id = i.primary_country_id))`

func (s *Store) Create(ctx context.Context, input industrychainbiz.IndustryChain) (industrychainbiz.IndustryChain, error) {
	exists, err := s.objectIdentityExists(ctx, input.ID)
	if err != nil {
		return industrychainbiz.IndustryChain{}, err
	}
	if exists {
		return industrychainbiz.IndustryChain{}, industrychainbiz.ErrConflict
	}
	if err := s.requireCountry(ctx, input.PrimaryCountryID); err != nil {
		return industrychainbiz.IndustryChain{}, err
	}
	row := s.db.QueryRowContext(ctx, `WITH inserted AS (
INSERT INTO industry_chain (id,name,aliases,scope,target_output,end_use,geography,primary_country_id,as_of_date,review_status,review_note,technology_route_qualifier,observable_variables)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING *)
SELECT id,name,array_to_json(aliases),scope,target_output,end_use,geography,primary_country_id,as_of_date,review_status,review_note,technology_route_qualifier,array_to_json(observable_variables),created_at,updated_at,
(primary_country_id IS NULL OR EXISTS (SELECT 1 FROM countries country WHERE country.id = inserted.primary_country_id)) FROM inserted`,
		input.ID, input.Name, input.Aliases, input.Scope, input.TargetOutput, input.EndUse, input.Geography, input.PrimaryCountryID, input.AsOfDate, input.ReviewStatus, input.ReviewNote, input.TechnologyRouteQualifier, input.ObservableVariables)
	return scanIndustryChain(row, classifyWriteError)
}
func (s *Store) Get(ctx context.Context, id industrychainbiz.ID) (industrychainbiz.IndustryChain, error) {
	return scanIndustryChain(s.db.QueryRowContext(ctx, `SELECT `+industryChainColumns+` FROM industry_chain i WHERE i.id=$1`, id), classifyReadError)
}
func (s *Store) List(ctx context.Context, query industrychainbiz.ListQuery) (industrychainbiz.ListResult, error) {
	var afterID any
	if query.After != nil {
		afterID = query.After.ID
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+industryChainColumns+` FROM industry_chain i WHERE $1::text IS NULL OR i.id>$1 ORDER BY i.id LIMIT $2`, afterID, query.PageSize+1)
	if err != nil {
		return industrychainbiz.ListResult{}, classifyReadError(err)
	}
	defer rows.Close()
	items := make([]industrychainbiz.IndustryChain, 0, query.PageSize+1)
	for rows.Next() {
		item, err := scanIndustryChain(rows, classifyReadError)
		if err != nil {
			return industrychainbiz.ListResult{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return industrychainbiz.ListResult{}, classifyReadError(err)
	}
	hasMore := len(items) > query.PageSize
	if hasMore {
		items = items[:query.PageSize]
	}
	return industrychainbiz.ListResult{Items: items, HasMore: hasMore}, nil
}
func (s *Store) Update(ctx context.Context, id industrychainbiz.ID, input industrychainbiz.Update) (industrychainbiz.IndustryChain, error) {
	if err := s.requireCountry(ctx, input.PrimaryCountryID); err != nil {
		return industrychainbiz.IndustryChain{}, err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE industry_chain SET name=$2,aliases=$3,scope=$4,target_output=$5,end_use=$6,geography=$7,primary_country_id=$8,as_of_date=$9,review_status=$10,review_note=$11,technology_route_qualifier=$12,observable_variables=$13,updated_at=now()
WHERE id=$1 AND ROW(name,aliases,scope,target_output,end_use,geography,primary_country_id,as_of_date,review_status,review_note,technology_route_qualifier,observable_variables)
IS DISTINCT FROM ROW($2::text,$3::text[],$4::text,$5::text,$6::text,$7::text,$8::text,$9::date,$10::varchar,$11::text,$12::text,$13::text[])`, id, input.Name, input.Aliases, input.Scope, input.TargetOutput, input.EndUse, input.Geography, input.PrimaryCountryID, input.AsOfDate, input.ReviewStatus, input.ReviewNote, input.TechnologyRouteQualifier, input.ObservableVariables)
	if err != nil {
		return industrychainbiz.IndustryChain{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func scanIndustryChain(row rowScanner, classify func(error) error) (industrychainbiz.IndustryChain, error) {
	var result industrychainbiz.IndustryChain
	var aliasesJSON, variablesJSON []byte
	var country, reviewNote, qualifier sql.NullString
	var countryValid bool
	if err := row.Scan(&result.ID, &result.Name, &aliasesJSON, &result.Scope, &result.TargetOutput, &result.EndUse, &result.Geography, &country, &result.AsOfDate, &result.ReviewStatus, &reviewNote, &qualifier, &variablesJSON, &result.CreatedAt, &result.UpdatedAt, &countryValid); err != nil {
		return industrychainbiz.IndustryChain{}, classify(err)
	}
	if country.Valid {
		result.PrimaryCountryID = &country.String
	}
	if reviewNote.Valid {
		result.ReviewNote = &reviewNote.String
	}
	if qualifier.Valid {
		result.TechnologyRouteQualifier = &qualifier.String
	}
	if err := json.Unmarshal(aliasesJSON, &result.Aliases); err != nil {
		return industrychainbiz.IndustryChain{}, industrychainbiz.ErrPersistence
	}
	if err := json.Unmarshal(variablesJSON, &result.ObservableVariables); err != nil {
		return industrychainbiz.IndustryChain{}, industrychainbiz.ErrPersistence
	}
	if !countryValid || result.Aliases == nil || result.ObservableVariables == nil || result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return industrychainbiz.IndustryChain{}, industrychainbiz.ErrPersistence
	}
	if err := industrychainbiz.ValidatePersisted(result); err != nil {
		return industrychainbiz.IndustryChain{}, industrychainbiz.ErrPersistence
	}
	return result, nil
}
func (s *Store) objectIdentityExists(ctx context.Context, id industrychainbiz.ID) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM entity_nodes WHERE id=$1 UNION ALL SELECT 1 FROM industry WHERE id=$1 UNION ALL SELECT 1 FROM concept WHERE id=$1 UNION ALL SELECT 1 FROM chain_node WHERE id=$1 UNION ALL SELECT 1 FROM industry_chain WHERE id=$1)`, id).Scan(&exists)
	if err != nil {
		return false, classifyReadError(err)
	}
	return exists, nil
}
func (s *Store) requireCountry(ctx context.Context, id *string) error {
	if id == nil {
		return nil
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM countries WHERE id=$1)`, *id).Scan(&exists); err != nil {
		return classifyReadError(err)
	}
	if !exists {
		return &industrychainbiz.ReferenceError{Field: "primary_country_id", Message: "identifies an unknown Country"}
	}
	return nil
}
func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return industrychainbiz.ErrNotFound
	}
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) {
		return industrychainbiz.ErrPersistence
	}
	switch pgerr.Code {
	case "23505", "P0001":
		return industrychainbiz.ErrConflict
	case "23503":
		return &industrychainbiz.ReferenceError{Field: "primary_country_id", Message: "identifies an unknown Country"}
	case "22001", "23502", "23514":
		return &industrychainbiz.ValidationError{Field: "industry_chain", Message: "violates the persistence contract"}
	default:
		return industrychainbiz.ErrPersistence
	}
}
func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return industrychainbiz.ErrNotFound
	}
	return industrychainbiz.ErrPersistence
}

var _ industrychainbiz.Repository = (*Store)(nil)
