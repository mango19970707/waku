package agents

import (
	"context"
	"fmt"
	"waku/internal/character/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewCharacterReviserAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterReviserAgent",
		Description: "根据审核结论修改角色的专业 Agent",
		Instruction: prompts.CharacterReviserPrompt,
		Model:       chatModel,
	})
}

func ReviseCharacters(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, characters, reviewSuggestions string) (string, error) {
	query := fmt.Sprintf("请根据角色数据和审核修改意见输出最终角色集合。\n\n小说类型：%s\n\n小说大纲：\n%s\n\n当前角色数据：\n%s\n\n审核修改意见：\n%s\n\n请严格输出 characters JSON。", genre, outline, characters, reviewSuggestions)
	return runCharacterJSONQuery(ctx, agent, query, "角色修改")
}
