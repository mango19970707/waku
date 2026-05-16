package compose

import (
	"context"
	"fmt"
	"waku/internal/plotline/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// AgentExecutor 定义剧情线工作流依赖的 Agent 执行接口。
type AgentExecutor interface {
	GeneratePlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error)
	PolishPlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, plotlines, genre string) (string, error)
	ReviewPlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, userSuggestion string) (string, error)
	RevisePlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, userSuggestion string) (string, error)
}

type defaultAgentExecutor struct{}

var agentExecutor AgentExecutor = &defaultAgentExecutor{}

// RegisterAgentExecutor 用于在测试中替换默认执行器。
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

func (e *defaultAgentExecutor) GeneratePlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	agent, err := agents.NewPlotlineGeneratorAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.GeneratePlotlines(ctx, agent, outline, genre)
}

func (e *defaultAgentExecutor) PolishPlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, plotlines, genre string) (string, error) {
	agent, err := agents.NewPlotlinePolisherAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.PolishPlotlines(ctx, agent, plotlines, genre)
}

func (e *defaultAgentExecutor) ReviewPlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, userSuggestion string) (string, error) {
	agent, err := agents.NewPlotlineReviewerAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.ReviewPlotlines(ctx, agent, outline, genre, plotlines, userSuggestion)
}

func (e *defaultAgentExecutor) RevisePlotlines(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, userSuggestion string) (string, error) {
	agent, err := agents.NewPlotlineReviserAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.RevisePlotlines(ctx, agent, outline, genre, plotlines, userSuggestion)
}

func ensureAgentExecutor() error {
	if agentExecutor == nil {
		return fmt.Errorf("agent executor 未初始化")
	}
	return nil
}
