package collector

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
)

const (
	maxPlannedQueries = 12
	maxQueryRunes     = 256
)

var (
	ErrQueryPlanningModel  = errors.New("query planning model call failed")
	ErrQueryPlanningSchema = errors.New("query planning response is invalid")
)

type QueryPlanner interface {
	Plan(context.Context, *Request) (*Request, error)
}

type DeepSeekQueryPlanner struct {
	chatModel    model.BaseChatModel
	systemPrompt string
}

func NewDeepSeekQueryPlanner(chatModel model.BaseChatModel, systemPrompt string) (*DeepSeekQueryPlanner, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}
	if strings.TrimSpace(systemPrompt) == "" {
		return nil, fmt.Errorf("query planner prompt is required")
	}
	return &DeepSeekQueryPlanner{chatModel: chatModel, systemPrompt: systemPrompt}, nil
}

func (p *DeepSeekQueryPlanner) Plan(ctx context.Context, input *Request) (*Request, error) {
	if input == nil {
		return nil, fmt.Errorf("collector request is required")
	}
	userMessage, err := json.Marshal(struct {
		Objective       string   `json:"objective"`
		CollectedAt     string   `json:"collected_at_utc"`
		TimeWindowHours int      `json:"time_window_hours"`
		SearchQueries   []string `json:"search_queries"`
	}{
		Objective:       input.Objective,
		CollectedAt:     input.CollectedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		TimeWindowHours: input.TimeWindowHours,
		SearchQueries:   append([]string(nil), input.SearchQueries...),
	})
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}

	response, err := p.chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(p.systemPrompt),
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
	modelQueries, err := decodePlannedQueries(response.Content)
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}
	queries, err := mergePlannedQueries(input.SearchQueries, modelQueries)
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}

	output := *input
	output.SearchQueries = append([]string(nil), queries...)
	return &output, nil
}

func decodePlannedQueries(content string) ([]string, error) {
	if strings.TrimSpace(content) == "" {
		return nil, ErrQueryPlanningSchema
	}
	var payload struct {
		Queries []string `json:"queries"`
	}
	decoder := json.NewDecoder(bytes.NewBufferString(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, ErrQueryPlanningSchema
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, ErrQueryPlanningSchema
	}
	if len(payload.Queries) == 0 {
		return nil, ErrQueryPlanningSchema
	}
	for _, query := range payload.Queries {
		cleaned := strings.TrimSpace(query)
		if cleaned == "" || utf8.RuneCountInString(cleaned) > maxQueryRunes {
			return nil, ErrQueryPlanningSchema
		}
	}
	return payload.Queries, nil
}

func mergePlannedQueries(original, planned []string) ([]string, error) {
	result := make([]string, 0, maxPlannedQueries)
	seen := make(map[string]struct{}, maxPlannedQueries)
	appendUnique := func(queries []string, rejectEmpty bool) error {
		for _, query := range queries {
			cleaned := strings.TrimSpace(query)
			if cleaned == "" {
				if rejectEmpty {
					return ErrQueryPlanningSchema
				}
				continue
			}
			if utf8.RuneCountInString(cleaned) > maxQueryRunes {
				return ErrQueryPlanningSchema
			}
			if _, exists := seen[cleaned]; exists {
				continue
			}
			if len(result) == maxPlannedQueries {
				return nil
			}
			seen[cleaned] = struct{}{}
			result = append(result, cleaned)
		}
		return nil
	}
	if err := appendUnique(original, false); err != nil {
		return nil, err
	}
	if len(result) < maxPlannedQueries {
		if err := appendUnique(planned, true); err != nil {
			return nil, err
		}
	}
	if len(result) == 0 {
		return nil, ErrQueryPlanningSchema
	}
	return result, nil
}
