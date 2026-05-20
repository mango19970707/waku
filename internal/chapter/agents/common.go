package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"waku/internal/chapter/contracts"

	"github.com/cloudwego/eino/adk"
)

// runChapterJSONQuery 执行章节相关的 JSON 查询（生成、润色、修改）
func runChapterJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
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
			return "", fmt.Errorf("%s失败: %w", stage, event.Err)
		}
		if event.Output == nil {
			continue
		}

		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		slog.Info(msg.Content)
		builder.WriteString(msg.Content)
	}

	result := builder.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("%s失败: 未生成任何内容", stage)
	}

	normalized, _, err := contracts.NormalizeChapterPayload(result)
	if err != nil {
		return "", fmt.Errorf("%s输出格式无效: %w", stage, err)
	}
	return normalized, nil
}

// runChapterReviewJSONQuery 专门用于章节审核 Agent，输出格式为 {"suggestions":[...]}
func runChapterReviewJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
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
			return "", fmt.Errorf("%s失败: %w", stage, event.Err)
		}
		if event.Output == nil {
			continue
		}

		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		slog.Info(msg.Content)
		builder.WriteString(msg.Content)
	}

	result := builder.String()
	if strings.TrimSpace(result) == "" {
		return "", fmt.Errorf("%s失败: 未生成任何内容", stage)
	}

	normalized, _, err := contracts.NormalizeChapterReviewPayload(result)
	if err != nil {
		return "", fmt.Errorf("%s输出格式无效: %w", stage, err)
	}
	return normalized, nil
}
