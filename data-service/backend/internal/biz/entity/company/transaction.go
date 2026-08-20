package company

import "context"

type IndustryLink struct {
	ID         string
	IndustryID IndustryID
}

type IndustryTransaction interface {
	ReplaceIndustries(context.Context, ID, []IndustryLink) (Company, error)
}
