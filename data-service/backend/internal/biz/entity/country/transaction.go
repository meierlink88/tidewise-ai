package country

import "context"

type RegionTransaction interface {
	ReplaceRegions(context.Context, string, []string) (Country, error)
}
