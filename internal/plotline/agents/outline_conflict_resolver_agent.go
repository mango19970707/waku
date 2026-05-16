package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/plotline/prompts"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// OutlineConflictResolverInput 大纲冲突解决的输入参数
type OutlineConflictResolverInput struct {
	// OriginalOutline 原始大纲
	OriginalOutline string
	// Genre 小说类型
	Genre string
	// TargetPlotline 修改后的目标剧情线（JSON 格式）
	TargetPlotline string
	// Conflicts 冲突列表（JSON 格式）
	Conflicts string
}

// NewOutlineConflictResolverAgent 创建大纲冲突解决 Agent
//
// 功能：根据目标剧情线和冲突信息，修改大纲以解决冲突
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲冲突解决 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlineConflictResolverAgent(ctx, chatModel)
//	if err != nil {
//	    return err
//	}
//	result, err := ResolveOutlineConflicts(ctx, agent, input)
func NewOutlineConflictResolverAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineConflictResolver",
		Description: "大纲冲突解决专家，根据剧情线变化调整大纲以解决冲突",
		Instruction: prompts.OutlineConflictResolverPrompt,
		Model:       chatModel,
	})
}

// ResolveOutlineConflicts 解决大纲冲突
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲冲突解决 Agent
//   - input: 输入参数（原始大纲、目标剧情线、冲突信息）
//
// 返回值：
//   - string: 修改后的完整大纲
//   - error: 解决失败时返回错误
func ResolveOutlineConflicts(ctx context.Context, agent *adk.ChatModelAgent, input *OutlineConflictResolverInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}
	if input.TargetPlotline == "" {
		return "", fmt.Errorf("目标剧情线不能为空")
	}
	if input.Conflicts == "" {
		return "", fmt.Errorf("冲突信息不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息修改大纲，解决与目标剧情线的冲突：

**小说类型**：%s

**原始大纲**：
%s

**修改后的目标剧情线**：
%s

**检测到的冲突**：
%s

请直接输出修改后的完整大纲，不要添加任何额外的说明或注释。`, input.Genre, input.OriginalOutline, input.TargetPlotline, input.Conflicts)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlineConflictResolver] 开始修改大纲以解决冲突...")

	// 收集响应内容
	var lastMessage *schema.Message
	var fullContent string
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
				}
				lastMessage = msg
				fullContent = msg.Content
			}
		}
	}

	if lastMessage == nil {
		return "", fmt.Errorf("未获取到 Agent 响应")
	}

	if fullContent == "" {
		return "", fmt.Errorf("修改后的大纲为空")
	}

	slog.Info(fmt.Sprintf("[OutlineConflictResolver] 大纲修改完成，长度: %d 字符", len(fullContent)))

	return fullContent, nil
}
