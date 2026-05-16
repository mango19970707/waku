package agents

import (
	"context"
	"fmt"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlineReviserInput 是大纲修改 Agent 的输入。
type OutlineReviserInput struct {
	OriginalOutline string
	Suggestions     string
	Genre           string
}

// NewOutlineReviserAgent 创建小说大纲修改 Agent。
func NewOutlineReviserAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StageRevise, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineReviser",
		Description: "根据审核结果或用户建议修改 Novel JSON，并输出统一结构",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// ReviseOutline 根据建议修改大纲。
func ReviseOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineReviserInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.Suggestions == "" {
		return "", fmt.Errorf("修改建议不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息修改小说大纲，并严格按要求输出：
小说类型：%s
当前 Novel JSON：%s

修改依据：%s`, input.Genre, input.OriginalOutline, input.Suggestions)

	return runNovelJSONQuery(ctx, agent, query, input.Genre, "修改大纲")
}
