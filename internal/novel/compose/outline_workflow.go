package compose

import (
	"context"
	"fmt"
	"waku/internal/novel/tools"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// OutlineWorkflowState 大纲工作流状态
type OutlineWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string
	// NovelID 小说ID（用于更新已有小说，非必填）
	NovelID uint
	// NovelName 小说名称（用于文件保存路径）
	NovelName string
	// Genre 小说类型
	Genre string
	// OriginalOutline 原始大纲
	OriginalOutline string
	// GeneratedOutline 生成的大纲
	GeneratedOutline string
	// ExpandedOutline 扩写后的大纲
	ExpandedOutline string
	// PolishedOutline 润色后的大纲
	PolishedOutline string
	// ExtractedInfo 提取的结构化信息
	ExtractedInfo *OutlineExtractorOutput
	// SavedNovelID 保存到数据库后的小说ID
	SavedNovelID uint
	// ErrorMsg 错误信息
	ErrorMsg string
	// RetryCount 当前节点的重试次数
	RetryCount int
	// MaxRetries 最大重试次数
	MaxRetries int
	// WaitingForApproval 是否正在等待用户审批
	WaitingForApproval bool
	// UserApproved 用户是否批准（true=批准，false=拒绝）
	UserApproved bool
	// DisapproveReason 拒绝原因
	DisapproveReason string
}

// OutlineWorkflowInput 大纲工作流输入
type OutlineWorkflowInput struct {
	// NovelID 小说ID（用于更新已有小说，非必填）
	NovelID uint
	// NovelName 小说名称（非必填，第一次生成时可为空）
	NovelName string
	// Genre 小说类型
	Genre string
	// OriginalOutline 原始大纲
	OriginalOutline string
}

// OutlineExtractorOutput 大纲提取器的输出结构
type OutlineExtractorOutput struct {
	NovelTitle        string `json:"novel_title"`
	Synopsis          string `json:"synopsis"`
	Outline           string `json:"outline"`
	Worldview         string `json:"worldview"`
	CultivationSystem string `json:"cultivation_system"`
}

// AgentExecutor 定义 Agent 执行接口，用于解耦
type AgentExecutor interface {
	GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error)
	ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error)
	PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error)
	ExtractOutlineInfo(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, novelName string) (*OutlineExtractorOutput, error)
}

// defaultAgentExecutor 默认的 Agent 执行器实现
type defaultAgentExecutor struct{}

// SetAgentExecutor 设置自定义的 Agent 执行器（用于测试或扩展）
var agentExecutor AgentExecutor = &defaultAgentExecutor{}

func (e *defaultAgentExecutor) GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error) {
	// 这里需要调用 agents 包的函数，但为了避免循环依赖，我们将在 agents 包中注册实现
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ExtractOutlineInfo(ctx context.Context, chatModel einoModel.BaseChatModel, outline, genre, novelName string) (*OutlineExtractorOutput, error) {
	return nil, fmt.Errorf("agent executor not implemented")
}

// RegisterAgentExecutor 注册 Agent 执行器（由 agents 包调用）
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// ==================== 构建 Graph ====================

// BuildOutlineWorkflowGraph 构建大纲工作流 Graph
//
// 功能：创建一个完整的大纲生成工作流，包括生成、扩写、润色、提取、保存五个步骤
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接
//   - repo: 小说仓储实现
//
// 返回值：
//   - compose.Runnable: 可执行的 Graph
//   - error: 创建失败时返回错误
//
// 工作流程：
//  1. 生成：将原始创意扩展为完整大纲（失败重试3次）
//  2. 扩写：将大纲扩展为详细的故事框架（失败重试3次）
//  3. 润色：对大纲进行深度优化和提升（失败重试3次）
//  4. 提取：从大纲中提取结构化信息（失败重试3次）
//  5. 保存：将提取的信息保存到数据库（失败重试3次）
//
// 特性：
//   - 每个步骤的输出保存到 data/content/{小说名称}/abstract/{阶段}.md
//   - 支持断点续传：如果文件已存在，直接加载而不重新生成
//   - 工具调用失败自动重试3次
func BuildOutlineWorkflowGraph(ctx context.Context, chatModel einoModel.BaseChatModel, db *gorm.DB, repo tools.Repository) (compose.Runnable[*OutlineWorkflowInput, *OutlineWorkflowState], error) {
	graph := compose.NewGraph[*OutlineWorkflowInput, *OutlineWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("expand", compose.InvokableLambda(expandNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(db, repo)))

	// 添加边
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "generate")

	// 添加条件分支
	graph.AddBranch("generate", compose.NewGraphBranch(
		routeAfterGenerate,
		map[string]bool{
			"generate": true,
			"expand":   true,
		},
	))

	graph.AddBranch("expand", compose.NewGraphBranch(
		routeAfterExpand,
		map[string]bool{
			"expand": true,
			"polish": true,
		},
	))

	graph.AddBranch("polish", compose.NewGraphBranch(
		routeAfterPolish,
		map[string]bool{
			"polish":  true,
			"extract": true,
		},
	))

	graph.AddBranch("extract", compose.NewGraphBranch(
		routeAfterExtract,
		map[string]bool{
			"extract":  true,
			"saveToDB": true,
		},
	))

	graph.AddBranch("saveToDB", compose.NewGraphBranch(
		routeAfterSave,
		map[string]bool{
			"saveToDB":  true,
			compose.END: true,
		},
	))

	// 编译 Graph，设置最大步数以支持重试
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// ExecuteOutlineWorkflow 执行大纲工作流
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *OutlineWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecuteOutlineWorkflow(ctx context.Context, graph compose.Runnable[*OutlineWorkflowInput, *OutlineWorkflowState], input *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	if input.NovelName == "" {
		return nil, fmt.Errorf("小说名称不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.OriginalOutline == "" {
		return nil, fmt.Errorf("原始大纲不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📚 开始执行大纲工作流\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说名称: %s\n", input.NovelName)
	fmt.Printf("小说类型: %s\n", input.Genre)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 执行 Graph
	result, err := graph.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("工作流执行失败: %w", err)
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ 工作流执行完成\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("最终状态: %s\n", result.CurrentNode)
	if result.NovelID != 0 {
		fmt.Printf("小说ID: %d\n", result.NovelID)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}
