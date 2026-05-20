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

// ChapterGeneratorInput 是章节生成 Agent 的输入
type ChapterGeneratorInput struct {
	NovelOutline    string // 小说大纲
	NovelWorldview  string // 世界观
	CultivationSys  string // 修行体系
	VolumeOutline   string // 分卷大纲
	VolumeTheme     string // 分卷主题
	Plotlines       string // 剧情线列表
	Characters      string // 角色列表
	ExistingChapters string // 已有章节上下文
}

// NewChapterGeneratorAgent 创建章节生成 Agent
func NewChapterGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChapterGeneratorAgent",
		Description: "基于分卷大纲、剧情线和角色信息生成初始章节列表的专业 Agent",
		Instruction: prompts.ChapterGeneratorPrompt,
		Model:       chatModel,
	})
}

// GenerateChapters 执行章节生成
func GenerateChapters(ctx context.Context, agent *adk.ChatModelAgent, input *ChapterGeneratorInput) (string, error) {
	if input == nil {
		return "", fmt.Errorf("章节生成输入不能为空")
	}
	if strings.TrimSpace(input.VolumeOutline) == "" {
		return "", fmt.Errorf("分卷大纲不能为空")
	}

	query := fmt.Sprintf(`请根据以下信息生成初始章节列表，并且只输出 JSON。

小说大纲：
%s

世界观设定：
%s

修行体系：
%s

分卷大纲：
%s

分卷主题：
%s

全部剧情线：
%s

关键角色：
%s

已有章节上下文：
%s

要求：
1. 顶层字段只能是 chapters。
2. 每个章节必须包含 chapter_number、title、outline、key_events、key_characters、plotline_progress。
3. 章节号从 1 开始，在当前卷内连续且唯一。
4. 每章至少推进 1 条剧情线，主线剧情线应在每章都有体现。
5. 主角应在每章都有戏份，重要配角定期出场。
6. 不要输出解释文字。`,
		input.NovelOutline,
		input.NovelWorldview,
		input.CultivationSys,
		input.VolumeOutline,
		input.VolumeTheme,
		input.Plotlines,
		input.Characters,
		input.ExistingChapters)

	return runChapterJSONQuery(ctx, agent, query, "章节生成")
}
