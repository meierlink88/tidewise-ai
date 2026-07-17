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
	chatModel model.BaseChatModel
}

func NewDeepSeekQueryPlanner(chatModel model.BaseChatModel) (*DeepSeekQueryPlanner, error) {
	if chatModel == nil {
		return nil, fmt.Errorf("chat model is required")
	}
	return &DeepSeekQueryPlanner{chatModel: chatModel}, nil
}

func (p *DeepSeekQueryPlanner) Plan(ctx context.Context, input *Request) (*Request, error) {
	if input == nil {
		return nil, fmt.Errorf("collector request is required")
	}
	if strings.TrimSpace(input.Objective) == "" {
		return nil, ErrQueryPlanningSchema
	}
	userMessage, err := json.Marshal(struct {
		CollectedAt     string `json:"collected_at_utc"`
		TimeWindowHours int    `json:"time_window_hours"`
	}{
		CollectedAt:     input.CollectedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		TimeWindowHours: input.TimeWindowHours,
	})
	if err != nil {
		return nil, ErrQueryPlanningSchema
	}

	response, err := p.chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(input.Objective),
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
	queries, err := normalizePlannedQueries(modelQueries)
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
	return payload.Queries, nil
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
