package agents

import (
	"context"
	"fmt"
	"strings"

	"waku/internal/content/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// ContentGeneratorInput 是正文生成 Agent 的输入。
type ContentGeneratorInput struct {
	NovelContext    string
	VolumeContext   string
	ChapterContext  string
	PreviousContext string
	NextContext     string
	Characters      string
	Plotlines       string
	UserSuggestion  string
}

// NewContentGeneratorAgent 创建正文生成 Agent。
func NewContentGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{Name: "ContentGeneratorAgent", Description: "基于章节规划生成小说正文的专业 Agent", Instruction: prompts.ContentGeneratorPrompt, Model: chatModel})
}

// GenerateContent 执行正文生成。
func GenerateContent(ctx context.Context, agent *adk.ChatModelAgent, input *ContentGeneratorInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("正文生成输入不能为空")
	}
	if strings.TrimSpace(input.ChapterContext) == "" {
		return "", fmt.Errorf("章节上下文不能为空")
	}
	userSuggestion := "无"
	if strings.TrimSpace(input.UserSuggestion) != "" {
		userSuggestion = input.UserSuggestion
	}
	query := fmt.Sprintf(`请根据以下信息生成章节正文，并且只输出 JSON。

小说上下文：
%s

分卷上下文：
%s

目标章节上下文：
%s

上一章上下文：
%s

下一章上下文：
%s

相关角色：
%s

相关剧情线：
%s

用户建议：
%s

要求：
1. 顶层字段只能是 chapter。
2. 保留目标章节 id、novel_id、volume_id、chapter_number、title。
3. content 必须是完整章节正文，不能为空。
4. word_count 按正文实际字数估算。
5. status 使用 writing。
6. 不要输出解释文字。`, input.NovelContext, input.VolumeContext, input.ChapterContext, input.PreviousContext, input.NextContext, input.Characters, input.Plotlines, userSuggestion)
	return runContentJSONQuery(ctx, agent, query, "正文生成")
}
