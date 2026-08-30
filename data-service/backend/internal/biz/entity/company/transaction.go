package company

import (
	"context"
	"time"
)

type IndustryLink struct {
	ID         string
	CompanyID  ID
	IndustryID IndustryID
	CreatedAt  time.Time
}

type IndustryTransaction interface {
	ReplaceIndustries(context.Context, ID, []IndustryLink) (Company, error)
}
