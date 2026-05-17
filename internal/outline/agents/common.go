package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"waku/internal/outline/contracts"

	"github.com/cloudwego/eino/adk"
)

func runNovelJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, genre, stage string) (string, error) {
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
			return "", fmt.Errorf("%s 失败: %w", stage, event.Err)
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
		return "", fmt.Errorf("%s 失败：未生成任何内容", stage)
	}

	normalized, _, err := contracts.NormalizeNovelPayload(result, genre)
	if err != nil {
		return "", fmt.Errorf("%s 输出格式无效: %w", stage, err)
	}
	return normalized, nil
}

func runJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
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

	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "", fmt.Errorf("%s failed: empty output", stage)
	}
	return result, nil
}
