package agents

import (
	"context"
	"fmt"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewPlotlineReviewerAgent 创建剧情线审核 Agent。
func NewPlotlineReviewerAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlineReviewer",
		Description: "审核剧情线并输出修改建议",
		Instruction: prompts.PlotlineReviewerPrompt,
		Model:       chatModel,
	})
}

// ReviewPlotlines 审核剧情线集合，仅返回修改意见 JSON。
func ReviewPlotlines(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, plotlines, userSuggestion string) (string, error) {
	if outline == "" {
		return "", fmt.Errorf("outline cannot be empty")
	}
	if genre == "" {
		return "", fmt.Errorf("genre cannot be empty")
	}
	if plotlines == "" {
		return "", fmt.Errorf("plotlines cannot be empty")
	}

	query := fmt.Sprintf(`请审核以下剧情线集合，找出它们与大纲或彼此之间可能存在的冲突、不一致和节奏问题。

小说类型：%s

小说大纲：%s

用户建议：%s

剧情线集合：
%s

请只输出 JSON，顶层只能包含 reviews。
每条 review 必须包含：
1. original_content：对应的原剧情线内容
2. modification_suggestion：对应的修改建议

即使某条剧情线无需修改，也请保留 original_content，并给出“无需修改”之类的建议。`, genre, outline, userSuggestion, plotlines)

	return runPlotlineReviewJSONQuery(ctx, agent, query, "review plotlines")
}
