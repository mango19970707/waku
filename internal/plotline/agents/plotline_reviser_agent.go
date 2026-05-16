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
		Description: "根据用户建议和审核意见输出最终剧情线集合",
		Instruction: prompts.PlotlineReviserPrompt,
		Model:       chatModel,
	})
}

// RevisePlotlines 根据用户建议与审核结果修改剧情线集合。
func RevisePlotlines(ctx context.Context, agent *adk.ChatModelAgent, outline, genre, reviewedPlotlines, userSuggestion string) (string, error) {
	if outline == "" {
		return "", fmt.Errorf("小说大纲不能为空")
	}
	if genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}
	if reviewedPlotlines == "" {
		return "", fmt.Errorf("审核后的剧情线集合不能为空")
	}

	query := fmt.Sprintf(`请根据用户建议和审核结果，输出最终剧情线集合。

小说类型：
%s

小说大纲：
%s

用户建议：
%s

审核后的剧情线集合：
%s

请只输出 JSON，顶层只能包含 plotlines。必须返回最终完整剧情线集合。`, genre, outline, userSuggestion, reviewedPlotlines)

	return runPlotlineJSONQuery(ctx, agent, query, "修改剧情线")
}
