package compose

import (
	"context"
	"fmt"
	"waku/internal/outline/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// agentExecutorImpl 是默认的 Agent 执行器实现。
type agentExecutorImpl struct{}

func (e *agentExecutorImpl) GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error) {
	agent, err := agents.NewOutlineGeneratorAgent(ctx, chatModel, genre)
	if err != nil {
		return "", fmt.Errorf("创建生成 Agent 失败: %w", err)
	}

	input := &agents.OutlineGeneratorInput{
		OriginalOutline: originalOutline,
		Genre:           genre,
	}
	return agents.GenerateOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error) {
	agent, err := agents.NewOutlineExpanderAgent(ctx, chatModel, genre)
	if err != nil {
		return "", fmt.Errorf("创建扩写 Agent 失败: %w", err)
	}

	input := &agents.OutlineExpanderInput{
		BriefOutline: briefOutline,
		Genre:        genre,
	}
	return agents.ExpandOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, userSuggestion, genre string) (string, error) {
	agent, err := agents.NewOutlinePolisherAgent(ctx, chatModel, genre)
	if err != nil {
		return "", fmt.Errorf("创建润色 Agent 失败: %w", err)
	}

	input := &agents.OutlinePolisherInput{
		Outline:        outline,
		UserSuggestion: userSuggestion,
		Genre:          genre,
	}
	return agents.PolishOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) ReviewOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, userSuggestion, genre string) (string, error) {
	agent, err := agents.NewOutlineReviewerAgent(ctx, chatModel, genre)
	if err != nil {
		return "", fmt.Errorf("创建审核 Agent 失败: %w", err)
	}

	input := &agents.OutlineReviewerInput{
		Outline:        outline,
		UserSuggestion: userSuggestion,
		Genre:          genre,
	}
	return agents.ReviewOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) ReviseOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, suggestions, genre string) (string, error) {
	agent, err := agents.NewOutlineReviserAgent(ctx, chatModel, genre)
	if err != nil {
		return "", fmt.Errorf("创建修改 Agent 失败: %w", err)
	}

	input := &agents.OutlineReviserInput{
		OriginalOutline: originalOutline,
		Suggestions:     suggestions,
		Genre:           genre,
	}
	return agents.ReviseOutline(ctx, agent, input)
}

func init() {
	RegisterAgentExecutor(&agentExecutorImpl{})
}
