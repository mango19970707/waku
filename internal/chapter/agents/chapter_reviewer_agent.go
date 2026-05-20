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

// ChapterReviewerInput 是章节审核 Agent 的输入
type ChapterReviewerInput struct {
	Chapters        string // 章节列表（JSON 格式）
	NovelOutline    string // 小说大纲
	NovelWorldview  string // 世界观
	CultivationSys  string // 修行体系
	VolumeOutline   string // 分卷大纲
	Plotlines       string // 剧情线列表
	Characters      string // 角色列表
}

// NewChapterReviewerAgent 创建章节审核 Agent
func NewChapterReviewerAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChapterReviewerAgent",
		Description: "检查章节之间的逻辑连续性、节奏分布、剧情承接和设定一致性的专业 Agent",
		Instruction: prompts.ChapterReviewerPrompt,
		Model:       chatModel,
	})
}

// ReviewChapters 执行章节审核
func ReviewChapters(ctx context.Context, agent *adk.ChatModelAgent, input *ChapterReviewerInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("章节审核输入不能为空")
	}
	if strings.TrimSpace(input.Chapters) == "" {
		return "", fmt.Errorf("章节列表不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息审核章节的合理性，并且只输出 JSON。

当前章节列表：
%s

小说大纲：
%s

世界观设定：
%s

修行体系：
%s

分卷大纲：
%s

全部剧情线：
%s

关键角色：
%s

要求：
1. 顶层字段只能是 suggestions。
2. 检查章节连续性、节奏分布、剧情承接和设定一致性。
3. 如果所有章节都合理，输出空数组：{"suggestions": []}。
4. 每条建议必须包含 suggestion、target、id。
5. target 必须是 outline、plotline、character、volume、chapter 之一。
6. 不要输出解释文字。`,
		input.Chapters,
		input.NovelOutline,
		input.NovelWorldview,
		input.CultivationSys,
		input.VolumeOutline,
		input.Plotlines,
		input.Characters)

	return runChapterReviewJSONQuery(ctx, agent, query, "章节审核")
}
