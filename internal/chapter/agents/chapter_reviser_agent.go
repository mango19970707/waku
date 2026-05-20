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

// ChapterReviserInput 是章节修改 Agent 的输入
type ChapterReviserInput struct {
	Chapters          string // 章节列表（JSON 格式）
	ReviewSuggestions string // 审核建议（JSON 格式）
	NovelOutline      string // 小说大纲
	VolumeOutline     string // 分卷大纲
	Plotlines         string // 剧情线列表
	Characters        string // 角色列表
}

// NewChapterReviserAgent 创建章节修改 Agent
func NewChapterReviserAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChapterReviserAgent",
		Description: "根据审核建议修改章节内容的专业 Agent",
		Instruction: prompts.ChapterReviserPrompt,
		Model:       chatModel,
	})
}

// ReviseChapters 执行章节修改
func ReviseChapters(ctx context.Context, agent *adk.ChatModelAgent, input *ChapterReviserInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("章节修改输入不能为空")
	}
	if strings.TrimSpace(input.Chapters) == "" {
		return "", fmt.Errorf("章节列表不能为空")
	}
	if strings.TrimSpace(input.ReviewSuggestions) == "" {
		return "", fmt.Errorf("审核建议不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息修改章节内容，并且只输出 JSON。

当前章节列表：
%s

审核建议：
%s

小说大纲：
%s

分卷大纲：
%s

全部剧情线：
%s

关键角色：
%s

要求：
1. 顶层字段只能是 chapters。
2. 只输出需要修改的章节，未修改的章节不要输出。
3. 保留已有章节的 id 字段。
4. 保留被修改章节原有的 chapter_number，不要补充未修改章节。
5. 确保修改后的章节完全解决了审核建议指出的问题。
6. 不要输出解释文字。`,
		input.Chapters,
		input.ReviewSuggestions,
		input.NovelOutline,
		input.VolumeOutline,
		input.Plotlines,
		input.Characters)

	return runChapterJSONQuery(ctx, agent, query, "章节修改")
}
