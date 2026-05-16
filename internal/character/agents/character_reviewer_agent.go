package agents

import (
	"context"
	"fmt"
	"waku/internal/character/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewCharacterReviewerAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterReviewerAgent",
		Description: "审核角色设定合理性的专业 Agent",
		Instruction: prompts.CharacterReviewerPrompt,
		Model:       chatModel,
	})
}

func ReviewCharacters(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, characters, userSuggestion string) (string, error) {
	query := fmt.Sprintf("请审核以下角色信息并把审核意见折叠回角色数据。\n\n小说类型：%s\n\n小说大纲：\n%s\n\n角色数据：\n%s\n\n用户建议：\n%s\n\n请严格输出 characters JSON。", genre, outline, characters, userSuggestion)
	return runCharacterJSONQuery(ctx, agent, query, "角色审核")
}
