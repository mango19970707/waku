package agents

import (
	"context"
	"fmt"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewPlotlinePolisherAgent 创建剧情线润色 Agent。
func NewPlotlinePolisherAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlinePolisher",
		Description: "优化剧情线表达、节奏，并吸收用户建议",
		Instruction: prompts.PlotlinePolisherPrompt,
		Model:       chatModel,
	})
}

// PolishPlotlines 润色剧情线集合，并在此阶段吸收用户建议。
func PolishPlotlines(ctx context.Context, agent *adk.ChatModelAgent, plotlines, genre, userSuggestion string) (string, error) {
	if plotlines == "" {
		return "", fmt.Errorf("plotlines cannot be empty")
	}
	if genre == "" {
		return "", fmt.Errorf("genre cannot be empty")
	}

	query := fmt.Sprintf(`请对以下剧情线集合进行润色，优化表述、节奏和清晰度。
如果用户建议不为空，请在本阶段直接吸收这些建议；如果建议要求新增剧情线，也请直接产出新增后的完整剧情线集合。

小说类型：%s

用户建议：%s

剧情线集合：
%s

请只输出 JSON，顶层只能包含 plotlines。必须返回完整剧情线集合，每条剧情线至少包含 title、type、description、status。`, genre, userSuggestion, plotlines)

	return runPlotlineJSONQuery(ctx, agent, query, "polish plotlines")
}
