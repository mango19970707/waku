package agents

import (
	"context"
	"fmt"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlineReviewerInput 是大纲审核 Agent 的输入。
type OutlineReviewerInput struct {
	Outline string
	Genre   string
}

// NewOutlineReviewerAgent 创建大纲审核 Agent。
func NewOutlineReviewerAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StageReview, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineReviewer",
		Description: "审核 Novel JSON 中的大纲内容，并把审核建议写回 Outline 内层 JSON",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// ReviewOutline 审核小说大纲并返回统一 Novel JSON。
func ReviewOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineReviewerInput) (string, error) {
	if input.Outline == "" {
		return "", fmt.Errorf("待审核大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请审核以下 Novel JSON，并严格按要求输出：

小说类型：%s
当前 Novel JSON：%s`, input.Genre, input.Outline)

	return runNovelJSONQuery(ctx, agent, query, input.Genre, "审核大纲")
}
