package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"waku/internal/plotline/contracts"

	"github.com/cloudwego/eino/adk"
)

func runPlotlineJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
	return runJSONQuery(ctx, agent, query, stage, contracts.NormalizePlotlinePayload)
}

func runPlotlineReviewJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
	return runJSONQuery(ctx, agent, query, stage, contracts.NormalizePlotlineReviewPayload)
}

func runJSONQuery[T any](ctx context.Context, agent *adk.ChatModelAgent, query, stage string, normalize func(string) (string, *T, error)) (string, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)
	var builder strings.Builder
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", fmt.Errorf("%s failed: %w", stage, event.Err)
		}
		if event.Output == nil {
			continue
		}

		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil || msg.Content == "" {
			continue
		}
		slog.Info(msg.Content)
		builder.WriteString(msg.Content)
	}

	result := builder.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("%s failed: empty response", stage)
	}

	normalized, _, err := normalize(result)
	if err != nil {
		return "", fmt.Errorf("%s returned invalid JSON: %w", stage, err)
	}
	return normalized, nil
}
