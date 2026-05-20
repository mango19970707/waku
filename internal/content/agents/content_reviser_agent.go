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

// ContentReviserInput 是正文修改 Agent 的输入。
type ContentReviserInput struct {
	Chapter           string
	ReviewSuggestions string
	NovelContext      string
	VolumeContext     string
	ChapterContext    string
	PreviousContext   string
	NextContext       string
	Characters        string
	Plotlines         string
}

// NewContentReviserAgent 创建正文修改 Agent。
func NewContentReviserAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{Name: "ContentReviserAgent", Description: "根据审核建议修改章节正文的专业 Agent", Instruction: prompts.ContentReviserPrompt, Model: chatModel})
}

// ReviseContent 执行正文修改。
func ReviseContent(ctx context.Context, agent *adk.ChatModelAgent, input *ContentReviserInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("正文修改输入不能为空")
	}
	if strings.TrimSpace(input.Chapter) == "" {
		return "", fmt.Errorf("待修改正文不能为空")
	}
	if strings.TrimSpace(input.ReviewSuggestions) == "" {
		return "", fmt.Errorf("审核建议不能为空")
	}
	query := fmt.Sprintf(`请根据审核建议修改章节正文，并且只输出 JSON。

当前章节正文 JSON：
%s

审核建议：
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
2. 保留 id、novel_id、volume_id、chapter_number、title。
3. 只根据审核建议修改 content、word_count、status。
4. 不要输出解释文字。`, input.Chapter, input.ReviewSuggestions, input.NovelContext, input.VolumeContext, input.ChapterContext, input.PreviousContext, input.NextContext, input.Characters, input.Plotlines)
	return runContentJSONQuery(ctx, agent, query, "正文修改")
}
