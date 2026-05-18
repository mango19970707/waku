package agents

import (
	"context"
	"fmt"
	"waku/internal/character/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewCharacterExtractorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "CharacterExtractorAgent",
		Description: "从小说大纲中提取角色信息的专业 Agent",
		Instruction: prompts.CharacterExtractorPrompt,
		Model:       chatModel,
	})
}

func ExtractCharacters(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, plotlines, existingCharacters string) (string, error) {
	query := fmt.Sprintf("请基于以下小说上下文提取并补全角色信息。\n\n小说类型：%s\n\n小说大纲：\n%s\n\n剧情线：\n%s\n\n已有角色：\n%s\n\n请严格输出 characters JSON。", genre, outline, plotlines, existingCharacters)
	return runCharacterJSONQuery(ctx, agent, query, "角色提取")
}
