package agents

import (
	"context"
	"fmt"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlineExpanderInput 是大纲扩写 Agent 的输入。
type OutlineExpanderInput struct {
	BriefOutline string
	Genre        string
}

// NewOutlineExpanderAgent 创建小说大纲扩写 Agent。
func NewOutlineExpanderAgent(ctx context.Context, chatModel model.BaseChatModel, genre string) (*adk.ChatModelAgent, error) {
	instruction, err := prompts.ResolveOutlinePrompt(prompts.StageExpand, genre)
	if err != nil {
		return nil, err
	}

	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineExpander",
		Description: "在现有 Novel JSON 基础上扩写大纲，并保持统一输出结构",
		Instruction: instruction,
		Model:       chatModel,
	})
}

// ExpandOutline 扩写小说大纲。
func ExpandOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineExpanderInput) (string, error) {
	if input.BriefOutline == "" {
		return "", fmt.Errorf("待扩写大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请基于以下 Novel JSON 扩写小说大纲，并严格按要求输出：

小说类型：%s
当前 Novel JSON：%s`, input.Genre, input.BriefOutline)

	return runNovelJSONQuery(ctx, agent, query, input.Genre, "扩写大纲")
}
