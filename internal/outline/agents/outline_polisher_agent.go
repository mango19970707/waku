package agents

import (
	"context"
	"fmt"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlinePolisherInput 是大纲润色 Agent 的输入。
type OutlinePolisherInput struct {
	Outline string
	Genre   string
}

// NewOutlinePolisherAgent 创建小说大纲润色 Agent。
func NewOutlinePolisherAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StagePolish, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlinePolisher",
		Description: "对 Novel JSON 中的大纲内容做润色优化，并保持结构稳定",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// PolishOutline 润色小说大纲。
func PolishOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlinePolisherInput) (string, error) {
	if input.Outline == "" {
		return "", fmt.Errorf("待润色大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请润色以下 Novel JSON，并严格按要求输出：

小说类型：%s
当前 Novel JSON：%s`, input.Genre, input.Outline)

	return runNovelJSONQuery(ctx, agent, query, input.Genre, "润色大纲")
}
