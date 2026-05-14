package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// PlotlinePolisherInput 剧情线润色器的输入参数
type PlotlinePolisherInput struct {
	// Plotline 待润色的剧情线
	Plotline string
	// Genre 小说类型
	Genre string
}

// NewPlotlinePolisherAgent 创建剧情线润色 Agent
//
// 功能：对剧情线进行全面润色和优化，提升质量
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 剧情线润色 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewPlotlinePolisherAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := PolishPlotline(ctx, agent, input)
func NewPlotlinePolisherAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlinePolisher",
		Description: "剧情线润色专家，对剧情线进行深度优化和提升",
		Instruction: prompts.PlotlinePolisherPrompt,
		Model:       chatModel,
	})
}

// PolishPlotline 润色剧情线
//
// 参数：
//   - ctx: 上下文
//   - agent: 剧情线润色 Agent
//   - input: 输入参数（剧情线和类型）
//
// 返回值：
//   - string: 润色后的剧情线
//   - error: 润色失败时返回错误
func PolishPlotline(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlinePolisherInput) (string, error) {
	if input.Plotline == "" {
		return "", fmt.Errorf("剧情线不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请对以下剧情线进行全面润色和优化：

**小说类型**：%s

**剧情线**：
%s

请提升剧情线的质量、逻辑性和可读性。`, input.Genre, input.Plotline)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[PlotlinePolisher] 开始润色剧情线...")

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("润色剧情线失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				// 打印流式输出的内容
				if msg.Content != "" {
					slog.Info(msg.Content)
					result += msg.Content
				}
			}
		}
	}

	slog.Info("[PlotlinePolisher] 剧情线润色完成")

	return result, nil
}
