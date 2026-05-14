package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	einoModel "github.com/cloudwego/eino/components/model"
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
//   - *compose.Graph[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]: 工作流图
//   - error: 创建失败时返回错误
func NewPlotlineRevisionWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	db *gorm.DB,
) (*compose.Graph[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], error) {
	// 创建工作流图
	workflow := compose.NewGraph[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]()

	// 添加节点
	if err := workflow.AddNode("init", compose.InvokableLambda(initRevisionNode(db))); err != nil {
		return nil, fmt.Errorf("添加 init 节点失败: %w", err)
	}

	if err := workflow.AddNode("polish", compose.InvokableLambda(polishRevisionNode(ctx, chatModel))); err != nil {
		return nil, fmt.Errorf("添加 polish 节点失败: %w", err)
	}

	if err := workflow.AddNode("updateDB", compose.InvokableLambda(updateDBNode(db))); err != nil {
		return nil, fmt.Errorf("添加 updateDB 节点失败: %w", err)
	}

	// 添加边（定义节点之间的连接）
	if err := workflow.AddEdge(compose.START, "init"); err != nil {
		return nil, fmt.Errorf("添加 START -> init 边失败: %w", err)
	}

	if err := workflow.AddEdge("init", "polish"); err != nil {
		return nil, fmt.Errorf("添加 init -> polish 边失败: %w", err)
	}

	// 添加条件边（根据状态决定下一步）
	if err := workflow.AddConditionalEdges("polish", routeAfterPolishRevision, map[string]string{
		"updateDB": "updateDB",
		"polish":   "polish",
	}); err != nil {
		return nil, fmt.Errorf("添加 polish 条件边失败: %w", err)
	}

	if err := workflow.AddConditionalEdges("updateDB", routeAfterUpdate, map[string]string{}); err != nil {
		return nil, fmt.Errorf("添加 updateDB 条件边失败: %w", err)
	}

	// 编译工作流
	if err := workflow.Compile(ctx); err != nil {
		return nil, fmt.Errorf("编译工作流失败: %w", err)
	}

	return workflow, nil
}
