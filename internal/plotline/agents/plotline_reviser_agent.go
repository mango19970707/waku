package agents

import (
	"context"
	"fmt"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewPlotlineReviserAgent 创建剧情线修改 Agent。
func NewPlotlineReviserAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlineReviser",
		Description: "根据审核建议输出最终剧情线集合",
		Instruction: prompts.PlotlineReviserPrompt,
		Model:       chatModel,
	})
}

// RevisePlotlines 根据审核建议修改润色后的剧情线集合。
func RevisePlotlines(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, polishedPlotlines, reviewSuggestions string) (string, error) {
	if outline == "" {
		return "", fmt.Errorf("outline cannot be empty")
	}
	if genre == "" {
		return "", fmt.Errorf("genre cannot be empty")
	}
	if polishedPlotlines == "" {
		return "", fmt.Errorf("polished plotlines cannot be empty")
	}
	if reviewSuggestions == "" {
		return "", fmt.Errorf("review suggestions cannot be empty")
	}

	query := fmt.Sprintf(`请根据审核建议，在不偏离小说大纲的前提下修改润色后的剧情线集合，并输出最终完整的 plotlines。

小说类型：%s

小说大纲：%s

润色后的剧情线集合：
%s

审核建议：
%s

请只输出 JSON，顶层只能包含 plotlines。必须返回最终完整剧情线集合。`, genre, outline, polishedPlotlines, reviewSuggestions)

	return runPlotlineJSONQuery(ctx, agent, query, "revise plotlines")
}
