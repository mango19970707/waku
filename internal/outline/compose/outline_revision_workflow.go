package compose

import (
	"context"
	"fmt"
	"waku/internal/outline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// OutlineRevisionWorkflowState 大纲修改工作流状态
type OutlineRevisionWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string
	// NovelID 小说ID（必填）
	NovelID uint
	// NovelName 小说名称
	NovelName string
	// Author 作者名称
	Author string
	// Genre 小说类型
	Genre string
	// OriginalOutline 原始大纲（从数据库加载）
	OriginalOutline string
	// UserSuggestion 用户的修改建议
	UserSuggestion string
	// RevisedOutline 修改后的大纲
	RevisedOutline string
	// ExtractedInfo 提取的结构化信息
	ExtractedInfo *OutlineExtractorOutput
	// ErrorMsg 错误信息
	ErrorMsg string
	// RetryCount 当前节点的重试次数
	RetryCount int
	// MaxRetries 最大重试次数
	MaxRetries int
}

// OutlineRevisionWorkflowInput 大纲修改工作流输入
type OutlineRevisionWorkflowInput struct {
	// NovelID 小说ID（必填）
	NovelID uint
	// UserSuggestion 用户的修改建议（必填）
	UserSuggestion string
}

// BuildOutlineRevisionWorkflowGraph 构建大纲修改工作流 Graph
//
// 功能：创建一个完整的大纲修改工作流，包括加载、修改、提取、更新四个步骤
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - repo: 小说仓储实现
//
// 返回值：
//   - compose.Runnable: 可执行的 Graph
//   - error: 创建失败时返回错误
//
// 工作流程：
//  1. 加载：从数据库加载现有小说的大纲信息
//  2. 修改：根据用户建议调整大纲内容（失败重试3次）
//  3. 提取：从修改后的大纲中提取结构化信息（失败重试3次）
//  4. 更新：将提取的信息更新到数据库（失败重试3次）
//
// 特性：
//   - 修改后的大纲保存到 data/content/{小说ID-名称}/abstract/修改.md
//   - 工具调用失败自动重试3次
func BuildOutlineRevisionWorkflowGraph(ctx context.Context, chatModel einoModel.BaseChatModel, repo repository.Repository) (compose.Runnable[*OutlineRevisionWorkflowInput, *OutlineRevisionWorkflowState], error) {
	graph := compose.NewGraph[*OutlineRevisionWorkflowInput, *OutlineRevisionWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initRevisionNode(repo)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractRevisionNode(ctx, chatModel)))
	graph.AddLambdaNode("update", compose.InvokableLambda(updateDBNode(repo)))

	// 添加边
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "revise")

	// 添加条件分支
	graph.AddBranch("revise", compose.NewGraphBranch(
		routeAfterRevise,
		map[string]bool{
			"revise":  true,
			"extract": true,
		},
	))

	graph.AddBranch("extract", compose.NewGraphBranch(
		routeAfterExtractRevision,
		map[string]bool{
			"extract": true,
			"update":  true,
		},
	))

	graph.AddBranch("update", compose.NewGraphBranch(
		routeAfterUpdate,
		map[string]bool{
			"update":    true,
			compose.END: true,
		},
	))

	// 编译 Graph，设置最大步数以支持重试
	return graph.Compile(ctx, compose.WithMaxRunSteps(30))
}

// ExecuteOutlineRevisionWorkflow 执行大纲修改工作流
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *OutlineRevisionWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecuteOutlineRevisionWorkflow(ctx context.Context, graph compose.Runnable[*OutlineRevisionWorkflowInput, *OutlineRevisionWorkflowState], input *OutlineRevisionWorkflowInput) (*OutlineRevisionWorkflowState, error) {
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}
	if input.UserSuggestion == "" {
		return nil, fmt.Errorf("用户建议不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📝 开始执行大纲修改工作流\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说ID: %d\n", input.NovelID)
	fmt.Printf("用户建议: %s\n", input.UserSuggestion)
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
