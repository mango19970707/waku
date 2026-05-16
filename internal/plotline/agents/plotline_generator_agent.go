package agents

import (
	"context"
	"fmt"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// NewPlotlineGeneratorAgent 创建剧情线生成 Agent。
func NewPlotlineGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlineGenerator",
		Description: "根据小说大纲生成完整剧情线集合",
		Instruction: prompts.PlotlineGeneratorPrompt,
		Model:       chatModel,
	})
}

// GeneratePlotlines 根据小说大纲生成剧情线集合。
func GeneratePlotlines(ctx context.Context, agent *adk.ChatModelAgent, outline, genre string) (string, error) {
	if outline == "" {
		return "", fmt.Errorf("小说大纲不能为空")
	}
	if genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	query := fmt.Sprintf(`请根据以下小说大纲生成剧情线集合。

小说类型：
%s

小说大纲：
%s

请只输出 JSON，顶层只能包含 plotlines。每条剧情线至少包含 title、type、description、status。`, genre, outline)

	return runPlotlineJSONQuery(ctx, agent, query, "生成剧情线")
}
