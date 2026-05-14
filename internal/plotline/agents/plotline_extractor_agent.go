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

// PlotlineExtractorInput 剧情线提取器的输入参数
type PlotlineExtractorInput struct {
	// Outline 小说大纲
	Outline string
	// Genre 小说类型
	Genre string
}

// NewPlotlineExtractorAgent 创建剧情线提取 Agent
//
// 功能：从小说大纲中提取剧情线
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 剧情线提取 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewPlotlineExtractorAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ExtractPlotline(ctx, agent, input)
func NewPlotlineExtractorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "PlotlineExtractor",
		Description: "剧情线提取专家，从小说大纲中提取完整的剧情线",
		Instruction: prompts.PlotlineExtractorPrompt,
		Model:       chatModel,
	})
}

// ExtractPlotline 从大纲中提取剧情线
//
// 参数：
//   - ctx: 上下文
//   - agent: 剧情线提取 Agent
//   - input: 输入参数（大纲和类型）
//
// 返回值：
//   - string: 提取的剧情线
//   - error: 提取失败时返回错误
func ExtractPlotline(ctx context.Context, agent *adk.ChatModelAgent, input *PlotlineExtractorInput) (string, error) {
	if input.Outline == "" {
		return "", fmt.Errorf("大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请从以下小说大纲中提取完整的剧情线：

**小说类型**：%s

**小说大纲**：
%s

请提取主线剧情、支线剧情、转折点、冲突、伏笔等关键信息，形成完整的剧情线。`, input.Genre, input.Outline)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[PlotlineExtractor] 开始从大纲提取剧情线...")

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("提取剧情线失败: %w", event.Err)
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

	slog.Info("[PlotlineExtractor] 剧情线提取完成")

	return result, nil
}
