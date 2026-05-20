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

// ContentPolisherInput 是正文润色 Agent 的输入。
type ContentPolisherInput struct {
	Chapter         string
	UserSuggestion  string
	NovelContext    string
	VolumeContext   string
	ChapterContext  string
	PreviousContext string
	NextContext     string
	Characters      string
	Plotlines       string
}

// NewContentPolisherAgent 创建正文润色 Agent。
func NewContentPolisherAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{Name: "ContentPolisherAgent", Description: "润色正文并根据用户建议修改正文内容的专业 Agent", Instruction: prompts.ContentPolisherPrompt, Model: chatModel})
}

// PolishContent 执行正文润色或按用户建议修改。
func PolishContent(ctx context.Context, agent *adk.ChatModelAgent, input *ContentPolisherInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("正文润色输入不能为空")
	}
	if strings.TrimSpace(input.Chapter) == "" {
		return "", fmt.Errorf("待润色正文不能为空")
	}
	userSuggestion := "无"
	if strings.TrimSpace(input.UserSuggestion) != "" {
		userSuggestion = input.UserSuggestion
	}
	query := fmt.Sprintf(`请优化或修改以下章节正文，并且只输出 JSON。

当前章节正文 JSON：
%s

用户建议：
%s

小说上下文：
%s

分卷上下文：
%s

章节规划上下文：
%s

上一章上下文：
%s

下一章上下文：
%s

相关角色：
%s

相关剧情线：
%s

要求：
1. 顶层字段只能是 chapter。
2. 如果用户建议不是“无”，必须优先按用户建议修改正文。
3. 保留 id、novel_id、volume_id、chapter_number、title。
4. 只修改 content、word_count、status。
5. 不要输出解释文字。`, input.Chapter, userSuggestion, input.NovelContext, input.VolumeContext, input.ChapterContext, input.PreviousContext, input.NextContext, input.Characters, input.Plotlines)
	return runContentJSONQuery(ctx, agent, query, "正文润色")
}
