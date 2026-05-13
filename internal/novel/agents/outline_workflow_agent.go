package agents

import (
	"context"
	"fmt"
	"waku/internal/novel/compose"
	"waku/internal/novel/tools"
	"waku/internal/utils"

	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	einoCompose "github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// ==================== Agent Executor 实现 ====================

// agentExecutorImpl 实现 compose.AgentExecutor 接口
type agentExecutorImpl struct{}

func (e *agentExecutorImpl) GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error) {
	agent, err := NewOutlineGeneratorAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建生成 Agent 失败: %w", err)
	}

	input := &OutlineGeneratorInput{
		OriginalOutline: originalOutline,
		Genre:           genre,
	}

	return GenerateOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error) {
	agent, err := NewOutlineExpanderAgent(ctx, chatModel)
	if err != nil {
		return "", fmt.Errorf("创建扩写 Agent 失败: %w", err)
	}

	input := &OutlineExpanderInput{
		BriefOutline: briefOutline,
		Genre:        genre,
	}

	return ExpandOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	agent, err := NewOutlinePolisherAgent(ctx, chatModel, nil, nil)
	if err != nil {
		return "", fmt.Errorf("创建润色 Agent 失败: %w", err)
	}

	input := &OutlinePolisherInput{
		Outline: outline,
		Genre:   genre,
	}

	return PolishOutline(ctx, agent, input)
}

func (e *agentExecutorImpl) ExtractOutlineInfo(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, novelName string) (*compose.OutlineExtractorOutput, error) {
	agent, err := NewOutlineExtractorAgent(ctx, chatModel)
	if err != nil {
		return nil, fmt.Errorf("创建提取 Agent 失败: %w", err)
	}

	input := &OutlineExtractorInput{
		Outline:   outline,
		Genre:     genre,
		NovelName: novelName,
	}

	result, err := ExtractOutlineInfo(ctx, agent, input)
	if err != nil {
		return nil, err
	}

	// 转换为 compose 包的输出类型
	return &compose.OutlineExtractorOutput{
		NovelTitle:        result.NovelTitle,
		Synopsis:          result.Synopsis,
		Outline:           result.Outline,
		Worldview:         result.Worldview,
		CultivationSystem: result.CultivationSystem,
	}, nil
}

// 在包初始化时注册 Agent 执行器
func init() {
	compose.RegisterAgentExecutor(&agentExecutorImpl{})
}

// ==================== 兼容旧接口 ====================

// OutlineWorkflowInput 大纲工作流输入（兼容旧接口）
type OutlineWorkflowInput = compose.OutlineWorkflowInput

// OutlineWorkflowState 大纲工作流状态（兼容旧接口）
type OutlineWorkflowState = compose.OutlineWorkflowState

// BuildOutlineWorkflowGraph 构建大纲工作流 Graph（兼容旧接口）
//
// 注意：此函数已迁移到 compose 包，这里保留用于向后兼容
func BuildOutlineWorkflowGraph(ctx context.Context, chatModel einoModel.BaseChatModel, db *gorm.DB, repo tools.Repository) (interface{}, error) {
	return compose.BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
}

// ExecuteOutlineWorkflow 执行大纲工作流（兼容旧接口）
//
// 注意：此函数已迁移到 compose 包，这里保留用于向后兼容
func ExecuteOutlineWorkflow(ctx context.Context, graph interface{}, input *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	// 类型断言
	if g, ok := graph.(einoCompose.Runnable[*compose.OutlineWorkflowInput, *compose.OutlineWorkflowState]); ok {
		return compose.ExecuteOutlineWorkflow(ctx, g, input)
	}
	return nil, fmt.Errorf("无效的 graph 类型")
}

// NewOutlineWorkflowAgent 创建小说大纲工作流 Agent（兼容旧接口）
//
// 注意：此函数保留用于向后兼容，新代码应使用 compose.BuildOutlineWorkflowGraph
func NewOutlineWorkflowAgent(ctx context.Context, chatModel einoModel.BaseChatModel, db *gorm.DB, repo tools.Repository) (adk.ResumableAgent, error) {
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

// GenerateOutlineWithWorkflow 使用工作流生成完整的小说大纲（兼容旧接口）
//
// 注意：此函数保留用于向后兼容，新代码应使用 compose.ExecuteOutlineWorkflow
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
