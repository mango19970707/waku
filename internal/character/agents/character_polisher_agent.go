package agents

import (
	"context"
	"fmt"
	"waku/internal/character/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewCharacterPolisherAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterPolisherAgent",
		Description: "润色角色设定表达和完整性的专业 Agent",
		Instruction: prompts.CharacterPolisherPrompt,
		Model:       chatModel,
	})
}

func PolishCharacters(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, plotlines, characters, userSuggestion string) (string, error) {
	query := fmt.Sprintf("请润色以下角色信息。\n\n小说类型：%s\n\n小说大纲：\n%s\n\n剧情线：\n%s\n\n角色数据：\n%s\n\n用户建议：\n%s\n\n请严格输出 characters JSON。", genre, outline, plotlines, characters, userSuggestion)
	return runCharacterJSONQuery(ctx, agent, query, "角色润色")
}
