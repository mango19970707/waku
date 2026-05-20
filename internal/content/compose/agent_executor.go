package compose

import (
	"context"
	"fmt"
	"sync"

	"waku/internal/content/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

var (
	agentExecutorOnce sync.Once
	agentExecutorErr  error
)

// ensureAgentExecutor 确保正文 Agent 执行器只初始化一次，预留后续接入共享资源的位置。
func ensureAgentExecutor() error {
	agentExecutorOnce.Do(func() {
		agentExecutorErr = nil
	})
	return agentExecutorErr
}

// executeGenerateAgent 创建并执行正文生成 Agent。
func executeGenerateAgent(ctx context.Context, chatModel einoModel.BaseChatModel, input *agents.ContentGeneratorInput) (string, error) {
	agent, err := agents.NewContentGeneratorAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建正文生成 Agent 失败: %w", err)
	}
	return agents.GenerateContent(ctx, agent, input)
}

// executePolishAgent 创建并执行正文润色 Agent。
func executePolishAgent(ctx context.Context, chatModel einoModel.BaseChatModel, input *agents.ContentPolisherInput) (string, error) {
	agent, err := agents.NewContentPolisherAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建正文润色 Agent 失败: %w", err)
	}
	return agents.PolishContent(ctx, agent, input)
}

// executeReviewAgent 创建并执行正文审核 Agent。
func executeReviewAgent(ctx context.Context, chatModel einoModel.BaseChatModel, input *agents.ContentReviewerInput) (string, error) {
	agent, err := agents.NewContentReviewerAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建正文审核 Agent 失败: %w", err)
	}
	return agents.ReviewContent(ctx, agent, input)
}

// executeReviseAgent 创建并执行正文修改 Agent。
func executeReviseAgent(ctx context.Context, chatModel einoModel.BaseChatModel, input *agents.ContentReviserInput) (string, error) {
	agent, err := agents.NewContentReviserAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建正文修改 Agent 失败: %w", err)
	}
	return agents.ReviseContent(ctx, agent, input)
}
