package compose

import (
	"context"

	"waku/internal/plotline/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// AgentExecutor Agent 执行器接口
type AgentExecutor interface {
	// ExtractPlotline 从大纲提取剧情线（兼容旧接口）
	ExtractPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error)

	// ExtractPlotlinesJSON 从大纲提取剧情线（返回 JSON 结构）
	ExtractPlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (*agents.PlotlineExtractorOutput, error)

	// RevisePlotlinesJSON 优化剧情线（返回 JSON 结构）
	RevisePlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingPlotlines, userSuggestion string) (*agents.PlotlineRevisionOutput, error)

	// PolishPlotline 润色剧情线
	PolishPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, plotline, genre string) (string, error)
}

// agentExecutor 全局 Agent 执行器实例
var agentExecutor AgentExecutor

// RegisterAgentExecutor 注册 Agent 执行器
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// realAgentExecutor 真实的 Agent 执行器实现
type realAgentExecutor struct{}

// ExtractPlotline 从大纲提取剧情线（兼容旧接口）
func (e *realAgentExecutor) ExtractPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	agent, err := agents.NewPlotlineExtractorAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}

	return agents.ExtractPlotline(ctx, agent, &agents.PlotlineExtractorInput{
		Outline: outline,
		Genre:   genre,
	})
}

// ExtractPlotlinesJSON 从大纲提取剧情线（返回 JSON 结构）
func (e *realAgentExecutor) ExtractPlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (*agents.PlotlineExtractorOutput, error) {
	agent, err := agents.NewPlotlineExtractorAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.ExtractPlotlines(ctx, agent, &agents.PlotlineExtractorInput{
		Outline: outline,
		Genre:   genre,
	})
}

// RevisePlotlinesJSON 优化剧情线（返回 JSON 结构）
func (e *realAgentExecutor) RevisePlotlinesJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingPlotlines, userSuggestion string) (*agents.PlotlineRevisionOutput, error) {
	agent, err := agents.NewPlotlineRevisionAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.RevisePlotlines(ctx, agent, &agents.PlotlineRevisionInput{
		Outline:           outline,
		Genre:             genre,
		ExistingPlotlines: existingPlotlines,
		UserSuggestion:    userSuggestion,
	})
}

// PolishPlotline 润色剧情线
func (e *realAgentExecutor) PolishPlotline(ctx context.Context, chatModel einoModel.BaseChatModel, plotline, genre string) (string, error) {
	agent, err := agents.NewPlotlinePolisherAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}

	return agents.PolishPlotline(ctx, agent, &agents.PlotlinePolisherInput{
		Plotline: plotline,
		Genre:    genre,
	})
}

func init() {
	// 注册真实的 Agent 执行器
	RegisterAgentExecutor(&realAgentExecutor{})
}
