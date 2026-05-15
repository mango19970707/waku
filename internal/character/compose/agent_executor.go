package compose

import (
	"context"

	einoModel "github.com/cloudwego/eino/components/model"

	"waku/internal/character/agents"
)

// AgentExecutor Agent 执行器接口
// 用于工作流中调用各个 Agent，支持依赖注入和测试 Mock
type AgentExecutor interface {
	// ExtractCharacter 提取人物信息（返回字符串）
	ExtractCharacter(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error)

	// ExtractCharactersJSON 提取人物信息（返回结构化 JSON）
	ExtractCharactersJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (*agents.CharacterExtractorOutput, error)

	// ReviewCharactersJSON 审核人物信息（返回结构化 JSON）
	ReviewCharactersJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters string) (*agents.CharacterReviewerOutput, error)

	// ReviseCharactersJSON 优化人物信息（返回结构化 JSON）
	ReviseCharactersJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingCharacters, userSuggestion string) (*agents.CharacterRevisionOutput, error)
}

// agentExecutor 全局执行器实例
var agentExecutor AgentExecutor

// RegisterAgentExecutor 注册 Agent 执行器
//
// 参数：
//   - executor: 执行器实例
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// GetAgentExecutor 获取 Agent 执行器
//
// 返回：
//   - AgentExecutor: 执行器实例
func GetAgentExecutor() AgentExecutor {
	if agentExecutor == nil {
		// 如果未注册，使用默认的真实实现
		agentExecutor = &realAgentExecutor{}
	}
	return agentExecutor
}

// realAgentExecutor 真实的 Agent 执行器实现
type realAgentExecutor struct{}

// ExtractCharacter 提取人物信息（返回字符串）
func (e *realAgentExecutor) ExtractCharacter(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	agent, err := agents.NewCharacterExtractorAgent(ctx, chatModel)
	if err != nil {
		return "", err
	}

	return agents.ExtractCharacter(ctx, agent, &agents.CharacterExtractorInput{
		Outline: outline,
		Genre:   genre,
	})
}

// ExtractCharactersJSON 提取人物信息（返回结构化 JSON）
func (e *realAgentExecutor) ExtractCharactersJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (*agents.CharacterExtractorOutput, error) {
	agent, err := agents.NewCharacterExtractorAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.ExtractCharacters(ctx, agent, &agents.CharacterExtractorInput{
		Outline: outline,
		Genre:   genre,
	})
}

// ReviewCharactersJSON 审核人物信息（返回结构化 JSON）
func (e *realAgentExecutor) ReviewCharactersJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, characters string) (*agents.CharacterReviewerOutput, error) {
	agent, err := agents.NewCharacterReviewerAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.ReviewCharacters(ctx, agent, &agents.CharacterReviewerInput{
		Outline:    outline,
		Genre:      genre,
		Characters: characters,
	})
}

// ReviseCharactersJSON 优化人物信息（返回结构化 JSON）
func (e *realAgentExecutor) ReviseCharactersJSON(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, existingCharacters, userSuggestion string) (*agents.CharacterRevisionOutput, error) {
	agent, err := agents.NewCharacterRevisionAgent(ctx, chatModel)
	if err != nil {
		return nil, err
	}

	return agents.ReviseCharacters(ctx, agent, &agents.CharacterRevisionInput{
		Outline:            outline,
		Genre:              genre,
		ExistingCharacters: existingCharacters,
		UserSuggestion:     userSuggestion,
	})
}

// init 初始化函数，注册默认的真实实现
func init() {
	RegisterAgentExecutor(&realAgentExecutor{})
}
