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
		Description: "审核剧情线之间与大纲之间的冲突和不一致",
		Instruction: prompts.PlotlineReviewerPrompt,
		Model:       chatModel,
	})
}

// ReviewPlotlines 审核剧情线集合并返回统一 JSON。
func ReviewPlotlines(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, plotlines, userSuggestion string) (string, error) {
	if outline == "" {
		return "", fmt.Errorf("小说大纲不能为空")
	}
	if genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}
	if plotlines == "" {
		return "", fmt.Errorf("剧情线集合不能为空")
	}

	query := fmt.Sprintf(`请审核以下剧情线集合，找出它们与大纲或彼此之间可能存在的冲突、不一致和节奏问题。

小说类型：
%s

小说大纲：
%s

用户建议：
%s

剧情线集合：
%s

请只输出 JSON，顶层只能包含 plotlines。你必须保留全部剧情线，并把审核建议写回每条剧情线的扩展字段。`, genre, outline, userSuggestion, plotlines)

	return runPlotlineJSONQuery(ctx, agent, query, "审核剧情线")
}
