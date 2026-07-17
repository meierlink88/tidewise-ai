package prompts

import _ "embed"

//go:embed query_planner_v1.md
var queryPlannerV1 string

func QueryPlannerV1() string {
	return queryPlannerV1
}
