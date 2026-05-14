package compose

import (
	"context"
	"fmt"

	"waku/internal/plotline/repository"

	"github.com/cloudwego/eino/compose"
	einoModel "github.com/cloudwego/eino/components/model"
	"gorm.io/gorm"
)

// NewPlotlineExtractWorkflow 创建剧情线提取工作流
//
// 功能：从小说大纲中提取剧情线，并进行润色，最后保存到数据库
//
// 工作流程：
//   1. init - 初始化工作流状态
//   2. extract - 从大纲提取剧情线
//   3. polish - 润色剧情线
//   4. saveToDB - 保存到数据库（需要用户审批）
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库实例
//
// 返回值：
//   - *compose.Graph[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState]: 工作流图
//   - error: 创建失败时返回错误
func NewPlotlineExtractWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	db *gorm.DB,
) (*compose.Graph[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState], error) {
	// 创建数据仓库
	repo := repository.NewRepository(db)

	// 创建工作流图
	workflow := compose.NewGraph[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState]()

	// 添加节点
	if err := workflow.AddNode("init", compose.InvokableLambda(initExtractNode)); err != nil {
		return nil, fmt.Errorf("添加 init 节点失败: %w", err)
	}

	if err := workflow.AddNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel))); err != nil {
		return nil, fmt.Errorf("添加 extract 节点失败: %w", err)
	}

	if err := workflow.AddNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel))); err != nil {
		return nil, fmt.Errorf("添加 polish 节点失败: %w", err)
	}

	if err := workflow.AddNode("saveToDB", compose.InvokableLambda(saveToDBNode(db, repo))); err != nil {
		return nil, fmt.Errorf("添加 saveToDB 节点失败: %w", err)
	}

	// 添加边（定义节点之间的连接）
	if err := workflow.AddEdge(compose.START, "init"); err != nil {
		return nil, fmt.Errorf("添加 START -> init 边失败: %w", err)
	}

	if err := workflow.AddEdge("init", "extract"); err != nil {
		return nil, fmt.Errorf("添加 init -> extract 边失败: %w", err)
	}

	// 添加条件边（根据状态决定下一步）
	if err := workflow.AddConditionalEdges("extract", routeAfterExtract, map[string]string{
		"polish":  "polish",
		"extract": "extract",
	}); err != nil {
		return nil, fmt.Errorf("添加 extract 条件边失败: %w", err)
	}

	if err := workflow.AddConditionalEdges("polish", routeAfterPolish, map[string]string{
		"saveToDB": "saveToDB",
		"polish":   "polish",
	}); err != nil {
		return nil, fmt.Errorf("添加 polish 条件边失败: %w", err)
	}

	if err := workflow.AddConditionalEdges("saveToDB", routeAfterSave, map[string]string{}); err != nil {
		return nil, fmt.Errorf("添加 saveToDB 条件边失败: %w", err)
	}

	// 编译工作流
	if err := workflow.Compile(ctx); err != nil {
		return nil, fmt.Errorf("编译工作流失败: %w", err)
	}

	return workflow, nil
}
