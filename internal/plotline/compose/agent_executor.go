package compose

import (
	"context"

	"waku/internal/plotline/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// realAgentExecutor 真实的 Agent 执行器实现
type realAgentExecutor struct{}

// ExtractPlotline 从大纲提取剧情线
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
