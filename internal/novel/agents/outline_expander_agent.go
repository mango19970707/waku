package agents

import (
	"context"
	"fmt"
	"waku/internal/novel/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

// OutlineExpanderInput 大纲扩写器的输入参数
type OutlineExpanderInput struct {
	// BriefOutline 简略的大纲内容
	BriefOutline string
	// Genre 小说类型
	Genre string
}

// NewOutlineExpanderAgent 创建小说大纲扩写 Agent
//
// 功能：将简略的大纲扩展为详细的故事框架，为后续章节创作提供充分指导
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲扩写 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlineExpanderAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ExpandOutline(ctx, agent, input)
func NewOutlineExpanderAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineExpander",
		Description: "小说大纲扩写专家，将简略大纲扩展为详细的故事框架",
		Instruction: prompts.OutlineExpanderPrompt,
		Model:       chatModel,
	})
}

// ExpandOutline 扩写小说大纲
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲扩写 Agent
//   - input: 输入参数（简略大纲和小说类型）
//
// 返回值：
//   - string: 扩写后的详细大纲
//   - error: 扩写失败时返回错误
func ExpandOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineExpanderInput) (string, error) {
	if input.BriefOutline == "" {
		return "", fmt.Errorf("简略大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请对以下简略大纲进行详细扩写：

**小说类型**：%s

**简略大纲**：
%s

请将大纲扩展为详细的故事框架，为后续创作提供充分指导。`, input.Genre, input.BriefOutline)

	// 创建 Runner 并执行查询
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("扩写大纲失败: %w", event.Err)
		}

		// 获取消息内容
		if event.Output != nil {
			if msg, _, err := adk.GetMessage(event); err == nil {
				result += msg.Content
			}
		}
	}

	return result, nil
}
