package compose

import (
	"context"
	"fmt"
	"sync"

	"waku/internal/chapter/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

var (
	agentExecutorOnce sync.Once
	agentExecutorErr  error
)

// ensureAgentExecutor 确保 Agent 执行器已初始化
func ensureAgentExecutor() error {
	agentExecutorOnce.Do(func() {
		// 这里可以添加 Agent 执行器的初始化逻辑
		// 目前暂时不需要特殊初始化
		agentExecutorErr = nil
	})
	return agentExecutorErr
}

// executeGenerateAgent 执行章节生成 Agent
func executeGenerateAgent(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	input *agents.ChapterGeneratorInput,
) (string, error) {
	agent, err := agents.NewChapterGeneratorAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建章节生成 Agent 失败: %w", err)
	}
	return agents.GenerateChapters(ctx, agent, input)
}

// executePolishAgent 执行章节润色 Agent
func executePolishAgent(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	input *agents.ChapterPolisherInput,
) (string, error) {
	agent, err := agents.NewChapterPolisherAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建章节润色 Agent 失败: %w", err)
	}
	return agents.PolishChapters(ctx, agent, input)
}

// executeReviewAgent 执行章节审核 Agent
func executeReviewAgent(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	input *agents.ChapterReviewerInput,
) (string, error) {
	agent, err := agents.NewChapterReviewerAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建章节审核 Agent 失败: %w", err)
	}
	return agents.ReviewChapters(ctx, agent, input)
}

// executeReviseAgent 执行章节修改 Agent
func executeReviseAgent(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	input *agents.ChapterReviserInput,
) (string, error) {
	agent, err := agents.NewChapterReviserAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建章节修改 Agent 失败: %w", err)
	}
	return agents.ReviseChapters(ctx, agent, input)
}
