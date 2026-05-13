package compose

import (
	"context"
	"fmt"
	"waku/internal/volume/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// agentExecutorImpl 实现 compose.AgentExecutor 接口
type agentExecutorImpl struct{}

func (e *agentExecutorImpl) PartitionVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, novelOutline, genre string, suggestedVolumeCount int) (string, error) {
	agent, err := agents.NewVolumePartitionAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建分卷 Agent 失败: %w", err)
	}

	input := &agents.VolumePartitionInput{
		NovelOutline:         novelOutline,
		Genre:                genre,
		SuggestedVolumeCount: suggestedVolumeCount,
	}

	return agents.PartitionVolumes(ctx, agent, input)
}

func (e *agentExecutorImpl) ExtractVolumeInfo(ctx context.Context, chatModel einoModel.BaseChatModel, partitionedText, genre string) (*agents.VolumePartitionOutput, error) {
	agent, err := agents.NewVolumeExtractorAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建卷信息提取 Agent 失败: %w", err)
	}

	input := &agents.VolumeExtractorInput{
		PartitionedText: partitionedText,
		Genre:           genre,
	}

	return agents.ExtractVolumeInfo(ctx, agent, input)
}

func (e *agentExecutorImpl) PartitionChapters(ctx context.Context, chatModel einoModel.BaseChatModel, volumeOutline, genre string, volumeNumber, suggestedChapterCount int) (string, error) {
	agent, err := agents.NewChapterPartitionAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建分章 Agent 失败: %w", err)
	}

	input := &agents.ChapterPartitionInput{
		VolumeOutline:         volumeOutline,
		Genre:                 genre,
		VolumeNumber:          volumeNumber,
		SuggestedChapterCount: suggestedChapterCount,
	}

	return agents.PartitionChapters(ctx, agent, input)
}

func (e *agentExecutorImpl) ExtractChapterInfo(ctx context.Context, chatModel einoModel.BaseChatModel, partitionedText, genre string) (*agents.ChapterPartitionOutput, error) {
	return agents.ExtractChapterInfo(ctx, chatModel, partitionedText, genre)
}

// 在包初始化时注册 Agent 执行器
func init() {
	RegisterAgentExecutor(&agentExecutorImpl{})
}
