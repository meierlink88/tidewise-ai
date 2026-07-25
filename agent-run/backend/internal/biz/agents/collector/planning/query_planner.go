package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/collector"
)

const (
	maxPlannedQueries = 12
	maxQueryRunes     = 256
	defaultTimeWindow = 48
)

const plannerProtocol = `You are a search-query planner. Return exactly one JSON object with no markdown or trailing text. The object must contain "queries" (an array of focused search strings), "combined_query" (one query suitable for single-query providers), and may contain "time_window_hours" only when the user's prompt explicitly states a time range. Do not add facts or choose data sources.`

var (
	ErrQueryPlanningModel  = errors.New("query planning model call failed")
	ErrQueryPlanningSchema = errors.New("query planning response is invalid")
)

type QueryPlanner interface {
	Plan(context.Context, *collector.Request) (*collector.Request, error)
}

type DeepSeekQueryPlanner struct {
	chatModel model.BaseChatModel
}

func NewDeepSeekQueryPlanner(chatModel model.BaseChatModel) (*DeepSeekQueryPlanner, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}
	return &DeepSeekQueryPlanner{chatModel: chatModel}, nil
}

func (p *DeepSeekQueryPlanner) Plan(ctx context.Context, input *collector.Request) (*collector.Request, error) {
	if input == nil {
		return nil, fmt.Errorf("collector request is required")
	}
	if strings.TrimSpace(input.Prompt) == "" {
		return nil, ErrQueryPlanningSchema
	}
	userMessage, err := json.Marshal(struct {
		Prompt      string `json:"prompt"`
		CollectedAt string `json:"collected_at_utc"`
	}{
		Prompt:      input.Prompt,
		CollectedAt: input.CollectedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}

	response, err := p.chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(plannerProtocol),
		schema.UserMessage(string(userMessage)),
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, fmt.Errorf("%w: %w", ErrQueryPlanningModel, ctxErr)
		}
		return nil, ErrQueryPlanningModel
	}
	if response == nil {
		return nil, ErrQueryPlanningSchema
	}
	plan, err := decodeQueryPlan(response.Content)
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}
	queries, err := normalizePlannedQueries(plan.Queries)
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}
	combinedQuery := strings.TrimSpace(plan.CombinedQuery)
	if combinedQuery == "" || utf8.RuneCountInString(combinedQuery) > maxQueryRunes {
		return nil, ErrQueryPlanningSchema
	}
	timeWindow := defaultTimeWindow
	if plan.TimeWindowHours != nil {
		if *plan.TimeWindowHours < 1 || *plan.TimeWindowHours > 8760 {
			return nil, ErrQueryPlanningSchema
		}
		timeWindow = *plan.TimeWindowHours
	}

	output := *input
	output.SearchQueries = append([]string(nil), queries...)
	output.CombinedQuery = combinedQuery
	output.TimeWindowHours = timeWindow
	return &output, nil
}

type queryPlan struct {
	Queries         []string `json:"queries"`
	CombinedQuery   string   `json:"combined_query"`
	TimeWindowHours *int     `json:"time_window_hours,omitempty"`
}

func decodeQueryPlan(content string) (queryPlan, error) {
	if strings.TrimSpace(content) == "" {
		return queryPlan{}, ErrQueryPlanningSchema
	}
	var payload queryPlan
	decoder := json.NewDecoder(bytes.NewBufferString(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return queryPlan{}, ErrQueryPlanningSchema
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return queryPlan{}, ErrQueryPlanningSchema
	}
	if len(payload.Queries) == 0 {
		return queryPlan{}, ErrQueryPlanningSchema
	}
	return payload, nil
}

func normalizePlannedQueries(planned []string) ([]string, error) {
	result := make([]string, 0, maxPlannedQueries)
	seen := make(map[string]struct{}, maxPlannedQueries)
	for _, query := range planned {
		cleaned := strings.TrimSpace(query)
		if cleaned == "" || utf8.RuneCountInString(cleaned) > maxQueryRunes {
			return nil, ErrQueryPlanningSchema
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		if len(result) == maxPlannedQueries {
			break
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	if len(result) == 0 {
		return nil, ErrQueryPlanningSchema
	}
	return result, nil
}
