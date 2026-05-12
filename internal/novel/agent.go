package novel

import (
	"context"
	"fmt"

	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// OutlineGeneratorInput 大纲生成器的输入参数
type OutlineGeneratorInput struct {
	// OriginalOutline 用户提供的原始大纲
	OriginalOutline string
	// Genre 小说类型（玄幻、都市、科幻、武侠、历史、悬疑推理、言情、奇幻等）
	Genre string
}

// OutlinePolisherInput 大纲润色器的输入参数
type OutlinePolisherInput struct {
	// Outline 需要润色的大纲内容
	Outline string
	// Genre 小说类型
	Genre string
}

// OutlineExpanderInput 大纲扩写器的输入参数
type OutlineExpanderInput struct {
	// BriefOutline 简略的大纲内容
	BriefOutline string
	// Genre 小说类型
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
//	result, err := agent.Generate(ctx, input)
func NewOutlineGeneratorAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineGenerator",
		Description: "小说大纲生成专家，将简单的创意扩展为完整的小说大纲",
		Instruction: OutlineGeneratorPrompt,
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
			return "", fmt.Errorf("生成大纲失败: %w", event.Err)
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

// NewOutlinePolisherAgent 创建小说大纲润色 Agent
//
// 功能：对已有的小说大纲进行深度润色和优化，提升大纲的质量和吸引力
// 润色完成后可以使用 save_outline_to_database 工具将大纲保存到数据库
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接（可选，如果提供则添加保存工具）
//
// 返回值：
//   - *adk.ChatModelAgent: 大纲润色 Agent
//   - error: 创建失败时返回错误
//
// 使用示例：
//
//	agent, err := NewOutlinePolisherAgent(ctx, chatModel, db)
//	if err != nil {
//	    return err
//	}
//	result, err := PolishOutline(ctx, agent, input)
func NewOutlinePolisherAgent(ctx context.Context, chatModel model.BaseChatModel, db *gorm.DB) (*adk.ChatModelAgent, error) {
	config := &adk.ChatModelAgentConfig{
		Name:        "OutlinePolisher",
		Description: "小说大纲润色专家，对大纲进行深度优化和提升，并可将大纲保存到数据库",
		Instruction: OutlinePolisherPrompt,
		Model:       chatModel,
	}

	// 如果提供了数据库连接，添加保存工具
	if db != nil {
		saveTool, err := NewSaveOutlineTool(db)
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
			return "", fmt.Errorf("润色大纲失败: %w", event.Err)
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
//	result, err := agent.Generate(ctx, input)
func NewOutlineExpanderAgent(ctx context.Context, chatModel model.BaseChatModel) (*adk.ChatModelAgent, error) {
	return utils.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OutlineExpander",
		Description: "小说大纲扩写专家，将简略大纲扩展为详细的故事框架",
		Instruction: OutlineExpanderPrompt,
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

// NewOutlineWorkflowAgent 创建小说大纲工作流 Agent（串行执行：生成 -> 扩写 -> 润色）
//
// 功能：将大纲生成、扩写、润色三个步骤串行组合，一次性完成从原始创意到精品大纲的全流程
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接（可选，如果提供则润色 Agent 可以保存大纲到数据库）
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
//	agent, err := NewOutlineWorkflowAgent(ctx, chatModel, db)
//	if err != nil {
//	    return err
//	}
//	result, err := agent.Generate(ctx, input)
func NewOutlineWorkflowAgent(ctx context.Context, chatModel model.BaseChatModel, db *gorm.DB) (adk.ResumableAgent, error) {
	// 创建三个子 Agent
	generatorAgent, err := NewOutlineGeneratorAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建大纲生成 Agent 失败: %w", err)
	}

	expanderAgent, err := NewOutlineExpanderAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建大纲扩写 Agent 失败: %w", err)
	}

	polisherAgent, err := NewOutlinePolisherAgent(ctx, chatModel, db)
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
