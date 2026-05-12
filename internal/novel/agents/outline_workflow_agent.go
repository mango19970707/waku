package agents

import (
	"context"
	"fmt"
	"waku/internal/novel/tools"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"
)

// NewOutlineWorkflowAgent 创建小说大纲工作流 Agent（串行执行：生成 -> 扩写 -> 润色）
//
// 功能：将大纲生成、扩写、润色三个步骤串行组合，一次性完成从原始创意到精品大纲的全流程
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接（可选，如果提供则润色 Agent 可以保存大纲到数据库）
//   - repo: 小说仓储实现（可选，如果提供 db 则必须提供）
//
// 返回值：
//   - adk.ResumableAgent: 大纲工作流 Agent
//   - error: 创建失败时返回错误
//
// 工作流程：
//  1. 大纲生成：将原始创意扩展为完整大纲
//  2. 大纲扩写：将大纲扩展为详细的故事框架
//  3. 大纲润色：对大纲进行深度优化和提升（可选保存到数据库）
//
// 使用示例：
//
//	agent, err := NewOutlineWorkflowAgent(ctx, chatModel, db, repo)
//	if err != nil {
//	    return err
//	}
//	result, err := GenerateOutlineWithWorkflow(ctx, agent, input)
func NewOutlineWorkflowAgent(ctx context.Context, chatModel model.BaseChatModel, db *gorm.DB, repo tools.Repository) (adk.ResumableAgent, error) {
	// 创建三个子 Agent
	generatorAgent, err := NewOutlineGeneratorAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建大纲生成 Agent 失败: %w", err)
	}

	expanderAgent, err := NewOutlineExpanderAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建大纲扩写 Agent 失败: %w", err)
	}

	polisherAgent, err := NewOutlinePolisherAgent(ctx, chatModel, db, repo)
	if err != nil {
		return nil, fmt.Errorf("创建大纲润色 Agent 失败: %w", err)
	}

	// 创建串行工作流 Agent
	return utils.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "OutlineWorkflow",
		Description: "小说大纲完整工作流：生成 -> 扩写 -> 润色",
		SubAgents: []adk.Agent{
			generatorAgent,
			expanderAgent,
			polisherAgent,
		},
	})
}

// GenerateOutlineWithWorkflow 使用工作流生成完整的小说大纲
//
// 参数：
//   - ctx: 上下文
//   - agent: 大纲工作流 Agent
//   - input: 输入参数（原始大纲和小说类型）
//
// 返回值：
//   - string: 最终生成的精品大纲
//   - error: 生成失败时返回错误
func GenerateOutlineWithWorkflow(ctx context.Context, agent adk.ResumableAgent, input *OutlineGeneratorInput) (string, error) {
	if input.OriginalOutline == "" {
		return "", fmt.Errorf("原始大纲不能为空")
	}
	if input.Genre == "" {
		return "", fmt.Errorf("小说类型不能为空")
	}

	// 构建查询消息
	query := fmt.Sprintf(`请根据以下信息生成、扩写并润色小说大纲：

**小说类型**：%s

**原始大纲**：
%s

请完成完整的大纲创作流程。`, input.Genre, input.OriginalOutline)

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
			return "", fmt.Errorf("工作流执行失败: %w", event.Err)
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
