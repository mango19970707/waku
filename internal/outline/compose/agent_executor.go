package compose

import (
	"context"
	"fmt"
	"waku/internal/outline/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// agentExecutorImpl 实现 compose.AgentExecutor 接口
type agentExecutorImpl struct{}

func (e *agentExecutorImpl) GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error) {
	agent, err := agents.NewOutlineGeneratorAgent(ctx, chatModel)
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
	agent, err := agents.NewOutlineExpanderAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建扩写 Agent 失败: %w", err)
	}

	input := &agents.OutlineExpanderInput{
		BriefOutline: briefOutline,
		Genre:        genre,
	}

	return agents.ExpandOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	agent, err := agents.NewOutlinePolisherAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建润色 Agent 失败: %w", err)
	}

	input := &agents.OutlinePolisherInput{
		Outline: outline,
		Genre:   genre,
	}

	return agents.PolishOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) ExtractOutlineInfo(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, novelName string) (*OutlineExtractorOutput, error) {
	agent, err := agents.NewOutlineExtractorAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建提取 Agent 失败: %w", err)
	}

	input := &agents.OutlineExtractorInput{
		Outline:   outline,
		Genre:     genre,
		NovelName: novelName,
	}

	result, err := agents.ExtractOutlineInfo(ctx, agent, input)
	if err != nil {
		return nil, err
	}

	// 转换为 compose 包的输出类型
	return &OutlineExtractorOutput{
		NovelTitle:        result.NovelTitle,
		Synopsis:          result.Synopsis,
		Outline:           result.Outline,
		Worldview:         result.Worldview,
		CultivationSystem: result.CultivationSystem,
	}, nil
}

func (e *agentExecutorImpl) ReviseOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, userSuggestion, genre string) (string, error) {
	agent, err := agents.NewOutlineReviserAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建修改 Agent 失败: %w", err)
	}

	input := &agents.OutlineReviserInput{
		OriginalOutline: originalOutline,
		UserSuggestion:  userSuggestion,
		Genre:           genre,
	}

	return agents.ReviseOutline(ctx, agent, input)
}

// 在包初始化时注册 Agent 执行器
func init() {
	RegisterAgentExecutor(&agentExecutorImpl{})
}
