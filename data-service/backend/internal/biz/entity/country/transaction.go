package country

import "context"

type RegionLink struct {
	ID       string
	RegionID string
}

type RegionTransaction interface {
	ReplaceRegions(context.Context, string, []RegionLink) (Country, error)
}
