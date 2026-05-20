package agents

import (
	"context"
	"fmt"
	"strings"

	"waku/internal/chapter/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// ChapterPolisherInput 是章节润色 Agent 的输入
type ChapterPolisherInput struct {
	Chapters         string // 章节列表（JSON 格式）
	UserSuggestion   string // 用户建议（可选）
	NovelOutline     string // 小说大纲
	VolumeOutline    string // 分卷大纲
	Plotlines        string // 剧情线列表
	Characters       string // 角色列表
	ExistingChapters string // 已有章节上下文
}

// NewChapterPolisherAgent 创建章节润色 Agent
func NewChapterPolisherAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChapterPolisherAgent",
		Description: "优化章节质量并根据用户建议修改内容的专业 Agent",
		Instruction: prompts.ChapterPolisherPrompt,
		Model:       chatModel,
	})
}

// PolishChapters 执行章节润色
func PolishChapters(ctx context.Context, agent *adk.ChatModelAgent, input *ChapterPolisherInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("章节润色输入不能为空")
	}
	if strings.TrimSpace(input.Chapters) == "" {
		return "", fmt.Errorf("章节列表不能为空")
	}

	userSuggestionText := "无"
	if strings.TrimSpace(input.UserSuggestion) != "" {
		userSuggestionText = input.UserSuggestion
	}

	query := fmt.Sprintf(`请根据以下信息优化章节质量或根据用户建议修改内容，并且只输出 JSON。

当前章节列表：
%s

用户建议：
%s

小说大纲：
%s

分卷大纲：
%s

全部剧情线：
%s

关键角色：
%s

已有章节上下文：
%s

要求：
1. 顶层字段只能是 chapters。
2. 如果提供了用户建议，必须根据建议修改内容。
3. 保留已有章节的 id 字段，新增章节省略 id。
4. 章节号从 1 开始，在当前卷内连续且唯一。
5. 优化章节标题、简介、大纲和节奏。
6. 不要输出解释文字。`,
		input.Chapters,
		userSuggestionText,
		input.NovelOutline,
		input.VolumeOutline,
		input.Plotlines,
		input.Characters,
		input.ExistingChapters)

	return runChapterJSONQuery(ctx, agent, query, "章节润色")
}
