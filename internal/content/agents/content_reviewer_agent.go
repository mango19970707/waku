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

// ContentReviewerInput 是正文审核 Agent 的输入。
type ContentReviewerInput struct {
	Chapter         string
	NovelContext    string
	VolumeContext   string
	ChapterContext  string
	PreviousContext string
	NextContext     string
	Characters      string
	Plotlines       string
}

// NewContentReviewerAgent 创建正文审核 Agent。
func NewContentReviewerAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{Name: "ContentReviewerAgent", Description: "审核章节正文质量和设定一致性的专业 Agent", Instruction: prompts.ContentReviewerPrompt, Model: chatModel})
}

// ReviewContent 执行正文审核。
func ReviewContent(ctx context.Context, agent *adk.ChatModelAgent, input *ContentReviewerInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("正文审核输入不能为空")
	}
	if strings.TrimSpace(input.Chapter) == "" {
		return "", fmt.Errorf("待审核正文不能为空")
	}
	query := fmt.Sprintf(`请审核以下章节正文，并且只输出 JSON。

章节正文 JSON：
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
1. 顶层字段只能是 suggestions。
2. 审核通过时输出 {"suggestions":[]}。
3. 有问题时给出具体可执行建议。
4. 不要输出解释文字。`, input.Chapter, input.NovelContext, input.VolumeContext, input.ChapterContext, input.PreviousContext, input.NextContext, input.Characters, input.Plotlines)
	return runContentReviewJSONQuery(ctx, agent, query, "正文审核")
}
