package compose

import (
	"context"
	"fmt"
	"waku/internal/character/agents"

	einoModel "github.com/cloudwego/eino/components/model"
)

type AgentExecutor interface {
	ExtractCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingCharacters string) (string, error)
	PolishCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error)
	ReviewCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error)
	ReviseCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error)
}

type defaultAgentExecutor struct{}

var agentExecutor AgentExecutor = &defaultAgentExecutor{}

func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

func ensureAgentExecutor() error {
	if agentExecutor == nil {
		return fmt.Errorf("agent executor 未初始化")
	}
	return nil
}

func (e *defaultAgentExecutor) ExtractCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingCharacters string) (string, error) {
	agent, err := agents.NewCharacterExtractorAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.ExtractCharacters(ctx, agent, outline, genre, existingCharacters)
}

func (e *defaultAgentExecutor) PolishCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error) {
	agent, err := agents.NewCharacterPolisherAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.PolishCharacters(ctx, agent, outline, genre, characters, userSuggestion)
}

func (e *defaultAgentExecutor) ReviewCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error) {
	agent, err := agents.NewCharacterReviewerAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.ReviewCharacters(ctx, agent, outline, genre, characters, userSuggestion)
}

func (e *defaultAgentExecutor) ReviseCharacters(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters, userSuggestion string) (string, error) {
	agent, err := agents.NewCharacterReviserAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}
	return agents.ReviseCharacters(ctx, agent, outline, genre, characters, userSuggestion)
}
