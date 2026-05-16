package compose

import (
	"context"
	"fmt"

	"waku/internal/volume/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

// AgentExecutor 定义分卷工作流的 Agent 执行接口。
type AgentExecutor interface {
	GenerateVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, existingVolumes string) (string, error)
	PolishVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error)
	ReviewVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error)
	ReviseVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error)
}

type defaultAgentExecutor struct{}

var agentExecutor AgentExecutor = &defaultAgentExecutor{}

// RegisterAgentExecutor 注册自定义执行器。
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

func ensureAgentExecutor() error {
	if agentExecutor == nil {
		return fmt.Errorf("agent executor 未初始化")
	}
	return nil
}

func (e *defaultAgentExecutor) GenerateVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, existingVolumes string) (string, error) {
	agent, err := agents.NewVolumeGeneratorAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.GenerateVolumes(ctx, agent, &agents.VolumeGeneratorInput{
		Outline:         outline,
		Genre:           genre,
		Plotlines:       plotlines,
		ExistingVolumes: existingVolumes,
	})
}

func (e *defaultAgentExecutor) PolishVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error) {
	agent, err := agents.NewVolumePolisherAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.PolishVolumes(ctx, agent, &agents.VolumePolisherInput{
		Outline:        outline,
		Genre:          genre,
		Plotlines:      plotlines,
		Volumes:        volumes,
		TargetVolume:   targetVolume,
		UserSuggestion: userSuggestion,
	})
}

func (e *defaultAgentExecutor) ReviewVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error) {
	agent, err := agents.NewVolumeReviewerAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.ReviewVolumes(ctx, agent, &agents.VolumeReviewerInput{
		Outline:        outline,
		Genre:          genre,
		Plotlines:      plotlines,
		Volumes:        volumes,
		TargetVolume:   targetVolume,
		UserSuggestion: userSuggestion,
	})
}

func (e *defaultAgentExecutor) ReviseVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, plotlines, volumes, targetVolume, userSuggestion string) (string, error) {
	agent, err := agents.NewVolumeReviserAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.ReviseVolumes(ctx, agent, &agents.VolumeReviserInput{
		Outline:        outline,
		Genre:          genre,
		Plotlines:      plotlines,
		Volumes:        volumes,
		TargetVolume:   targetVolume,
		UserSuggestion: userSuggestion,
	})
}

func init() {
	RegisterAgentExecutor(&defaultAgentExecutor{})
}
