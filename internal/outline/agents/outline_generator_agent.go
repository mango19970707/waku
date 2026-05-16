package agents

import (
	"context"
	"fmt"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlineGeneratorInput 是大纲生成 Agent 的输入。
type OutlineGeneratorInput struct {
	OriginalOutline string
	Genre           string
}

// NewOutlineGeneratorAgent 创建小说大纲生成 Agent。
func NewOutlineGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StageGenerate, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineGenerator",
		Description: "根据创意生成小说大纲，并输出统一的 Novel JSON",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// GenerateOutline 生成小说大纲。
func GenerateOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineGeneratorInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息生成小说大纲，并严格按要求输出：
小说类型：%s
原始创意：%s`, input.Genre, input.OriginalOutline)

	return runNovelJSONQuery(ctx, agent, query, input.Genre, "生成大纲")
}
