package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/outline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlineGeneratorInput 大纲生成器的输入参数
type OutlineGeneratorInput struct {
	// OriginalOutline 用户提供的原始大纲
	OriginalOutline string
	// Genre 小说类型（玄幻、都市、科幻、武侠、历史、悬疑推理、言情、奇幻等）
	Genre string
}

// NewOutlineGeneratorAgent 创建小说大纲生成 Agent
//
// 功能：根据用户提供的原始大纲和小说类型，生成一个详细、结构完整的小说大纲
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲生成 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlineGeneratorAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := GenerateOutline(ctx, agent, input)
func NewOutlineGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineGenerator",
		Description: "小说大纲生成专家，将简单的创意扩展为完整的小说大纲",
		Instruction: prompts.OutlineGeneratorPrompt,
		Model:       chatModel,
	})
}

// GenerateOutline 生成小说大纲
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲生成 Agent
//   - input: 输入参数（原始大纲和小说类型）
//
// 返回值：
//   - string: 生成的小说大纲
//   - error: 生成失败时返回错误
func GenerateOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineGeneratorInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息生成小说大纲：

**小说类型**：%s

**原始大纲**：
%s

请生成一个详细、结构完整的小说大纲。`, input.Genre, input.OriginalOutline)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlineGenerator] 开始生成大纲...")

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("生成大纲失败: %w", event.Err)
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

	slog.Info("[OutlineGenerator] 大纲生成完成")

	return result, nil
}
