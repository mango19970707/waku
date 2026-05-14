package compose

import (
	"context"
	"fmt"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// NewPlotlineRevisionWorkflow 创建剧情线修改工作流
//
// 功能：根据用户建议修改剧情线，并进行润色，最后更新到数据库
//
// 工作流程：
//   1. init - 从数据库加载小说信息和剧情线
//   2. polish - 根据用户建议润色剧情线
//   3. updateDB - 更新到数据库（需要用户审批）
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库实例
//
// 返回值：
//   - compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]: 可执行的工作流
//   - error: 创建失败时返回错误
func NewPlotlineRevisionWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	db *gorm.DB,
) (compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], error) {
	// 创建工作流图
	graph := compose.NewGraph[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initRevisionNode(db)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishRevisionNode(ctx, chatModel)))
	graph.AddLambdaNode("updateDB", compose.InvokableLambda(updateDBNode(db)))

	// 添加边（定义节点之间的连接）
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "polish")

	// 添加条件边（根据状态决定下一步）
	graph.AddBranch("polish", compose.NewGraphBranch(
		routeAfterPolishRevision,
		map[string]bool{
			"updateDB": true,
			"polish":   true,
		},
	))

	graph.AddBranch("updateDB", compose.NewGraphBranch(
		routeAfterUpdate,
		map[string]bool{
			"updateDB":  true,
			compose.END: true,
		},
	))

	// 编译工作流，设置最大步数以支持重试
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// ExecutePlotlineRevisionWorkflow 执行剧情线修改工作流
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *PlotlineRevisionWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecutePlotlineRevisionWorkflow(ctx context.Context, graph compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], input *PlotlineRevisionWorkflowInput) (*PlotlineRevisionWorkflowState, error) {
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}
	if input.UserSuggestion == "" {
		return nil, fmt.Errorf("用户修改建议不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✏️  开始执行剧情线修改工作流\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说ID: %d\n", input.NovelID)
	fmt.Printf("修改建议: %s\n", input.UserSuggestion)
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
