package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"waku/internal/content/contracts"

	"github.com/cloudwego/eino/adk"
)

// runContentJSONQuery 执行正文生成、润色、修改查询，并统一校验 chapter JSON。
func runContentJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{EnableStreaming: true, Agent: agent})
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
	normalized, _, err := contracts.NormalizeContentPayload(result)
	if err != nil {
		return "", fmt.Errorf("%s输出格式无效: %w", stage, err)
	}
	return normalized, nil
}

// runContentReviewJSONQuery 执行正文审核查询，并统一校验 suggestions JSON。
func runContentReviewJSONQuery(ctx context.Context, agent *adk.ChatModelAgent, query, stage string) (string, error) {
	runner := adk.NewRunner(ctx, adk.RunnerConfig{EnableStreaming: true, Agent: agent})
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
	normalized, _, err := contracts.NormalizeContentReviewPayload(result)
	if err != nil {
		return "", fmt.Errorf("%s输出格式无效: %w", stage, err)
	}
	return normalized, nil
}
