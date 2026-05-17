package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlinePolisherInput 是大纲预修改润色 Agent 的输入。
type OutlinePolisherInput struct {
	Outline        string
	UserSuggestion string
	Genre          string
}

// NewOutlinePolisherAgent 创建小说大纲预修改润色 Agent。
func NewOutlinePolisherAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StagePolish, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlinePolisher",
		Description: "对 Novel JSON 执行预修改润色；若存在 user_suggestion，则先按用户建议修改，再做润色和结构整理",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// PolishOutline 对小说大纲执行预修改润色。
func PolishOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlinePolisherInput) (string, error) {
	if input.Outline == "" {
		return "", fmt.Errorf("待润色大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	inputJSON, err := json.Marshal(map[string]any{
		"novel":           json.RawMessage(input.Outline),
		"user_suggestion": input.UserSuggestion,
	})
	if err != nil {
		return "", fmt.Errorf("序列化润色输入 JSON 失败: %w", err)
	}

	query := fmt.Sprintf(`请对下面的输入 JSON 执行预修改润色，并严格按要求输出：

小说类型：%s
输入 JSON：%s`, input.Genre, string(inputJSON))

	return runNovelJSONQuery(ctx, agent, query, input.Genre, "润色大纲")
}
