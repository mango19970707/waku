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

// OutlineReviewerInput 是大纲审核 Agent 的输入。
type OutlineReviewerInput struct {
	Outline        string
	UserSuggestion string
	Genre          string
}

// NewOutlineReviewerAgent 创建大纲审核 Agent。
func NewOutlineReviewerAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StageReview, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineReviewer",
		Description: "审核 Novel JSON 中的大纲内容，并只返回供修改节点使用的修改建议；若存在 user_suggestion，则优先保障其不被后续结构冲突破坏",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// ReviewOutline 审核小说大纲，并只返回修改建议 JSON。
func ReviewOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineReviewerInput) (string, error) {
	if input.Outline == "" {
		return "", fmt.Errorf("待审核大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	inputJSON, err := json.Marshal(map[string]any{
		"novel":           json.RawMessage(input.Outline),
		"user_suggestion": input.UserSuggestion,
	})
	if err != nil {
		return "", fmt.Errorf("序列化审核输入 JSON 失败: %w", err)
	}

	query := fmt.Sprintf(`请审核下面的输入 JSON，只返回修改建议。

输出要求：
1. 只返回 JSON。
2. 返回内容必须是建议数组，或包含 review_suggestions 字段的 JSON 对象。
3. 每条建议至少包含 type、problem、suggestion、severity。
4. 不要返回修改后的 Novel JSON，不要返回 Markdown。

小说类型：%s
输入 JSON：%s`, input.Genre, string(inputJSON))

	return runJSONQuery(ctx, agent, query, "审核大纲")
}
