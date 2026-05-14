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

// OutlineReviserInput 大纲修改器的输入参数
type OutlineReviserInput struct {
	// OriginalOutline 原始大纲内容
	OriginalOutline string
	// UserSuggestion 用户的修改建议
	UserSuggestion string
	// Genre 小说类型
	Genre string
}

// NewOutlineReviserAgent 创建小说大纲修改 Agent
//
// 功能：根据用户的建议调整和修改现有的大纲内容
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲修改 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlineReviserAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ReviseOutline(ctx, agent, input)
func NewOutlineReviserAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineReviser",
		Description: "小说大纲修改专家，根据用户建议调整和优化大纲内容",
		Instruction: prompts.OutlineReviserPrompt,
		Model:       chatModel,
	})
}

// ReviseOutline 根据用户建议修改大纲
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲修改 Agent
//   - input: 输入参数（原始大纲、用户建议、小说类型）
//
// 返回值：
//   - string: 修改后的大纲内容
//   - error: 修改失败时返回错误
func ReviseOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineReviserInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.UserSuggestion == "" {
		return "", fmt.Errorf("用户建议不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据用户的建议修改以下小说大纲：

**小说类型**：%s

**原始大纲**：
%s

**用户建议**：
%s

请根据用户的建议对大纲进行调整和优化，保持大纲的整体结构和逻辑性，同时充分考虑用户的意见。`, input.Genre, input.OriginalOutline, input.UserSuggestion)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlineReviser] 开始修改大纲...")

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("修改大纲失败: %w", event.Err)
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

	slog.Info("[OutlineReviser] 大纲修改完成")

	if result == "" {
		return "", fmt.Errorf("修改大纲失败：未生成任何内容")
	}

	return result, nil
}
