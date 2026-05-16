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
		Description: "优化剧情线表述、节奏和可读性",
		Instruction: prompts.PlotlinePolisherPrompt,
		Model:       chatModel,
	})
}

// PolishPlotlines 润色剧情线集合并返回统一 JSON。
func PolishPlotlines(ctx context.Context, agent *adk.ChatModelAgent, plotlines, genre string) (string, error) {
	if plotlines == "" {
		return "", fmt.Errorf("剧情线集合不能为空")
	}
	if genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请对以下剧情线集合进行润色，优化表述、节奏和清晰度。

小说类型：
%s

剧情线集合：
%s

请只输出 JSON，顶层只能包含 plotlines。必须保留全部剧情线。每条剧情线至少包含 title、type、description、status。`, genre, plotlines)

	return runPlotlineJSONQuery(ctx, agent, query, "润色剧情线")
}
