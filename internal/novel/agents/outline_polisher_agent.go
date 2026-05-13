package agents

import (
	"context"
	"fmt"
	"log/slog"
	"waku/internal/novel/prompts"
	"waku/internal/novel/tools"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// OutlinePolisherInput 大纲润色器的输入参数
type OutlinePolisherInput struct {
	// Outline 需要润色的大纲内容
	Outline string
	// Genre 小说类型
	Genre string
}

// NewOutlinePolisherAgent 创建小说大纲润色 Agent
//
// 功能：对已有的小说大纲进行深度润色和优化，提升大纲的质量和吸引力
// 润色完成后可以使用 save_outline_to_database 工具将大纲保存到数据库
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接（可选，如果提供则添加保存工具）
//   - repo: 小说仓储实现（可选，如果提供 db 则必须提供）
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲润色 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlinePolisherAgent(ctx, chatModel, db, repo)
//	if err != nil {
//	    return err
//	}
//	result, err := PolishOutline(ctx, agent, input)
func NewOutlinePolisherAgent(ctx context.Context, chatModel model.BaseChatModel, db *gorm.DB, repo tools.Repository) (*adk.ChatModelAgent, error) {
	config := &adk.ChatModelAgentConfig{
		Name:        "OutlinePolisher",
		Description: "小说大纲润色专家，对大纲进行深度优化和提升，并可将大纲保存到数据库",
		Instruction: prompts.OutlinePolisherPrompt,
		Model:       chatModel,
	}

	// 如果提供了数据库连接和仓储，添加保存工具
	if db != nil && repo != nil {
		saveTool, err := tools.NewSaveOutlineTool(db, repo)
		if err != nil {
			return nil, fmt.Errorf("创建保存工具失败: %w", err)
		}

		config.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{saveTool},
			},
		}
	}

	polisherAgent, err := utils.NewChatModelAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	return polisherAgent, nil
}

// PolishOutline 润色小说大纲
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲润色 Agent
//   - input: 输入参数（大纲内容和小说类型）
//
// 返回值：
//   - string: 润色后的小说大纲
//   - error: 润色失败时返回错误
func PolishOutline(ctx context.Context, agent *adk.ChatModelAgent, input *OutlinePolisherInput) (string, error) {
	if input.Outline == "" {
		return "", fmt.Errorf("大纲内容不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请对以下小说大纲进行润色和优化：

**小说类型**：%s

**原始大纲**：
%s

请对大纲进行专业的润色，提升其质量和吸引力。`, input.Genre, input.Outline)

	// 创建 Runner 并执行查询（启用流式输出）
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	iter := runner.Query(ctx, query)

	slog.Info("[OutlinePolisher] 开始润色大纲...")

	// 收集响应内容
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			return "", fmt.Errorf("润色大纲失败: %w", event.Err)
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

	slog.Info("[OutlinePolisher] 大纲润色完成")

	return result, nil
}
