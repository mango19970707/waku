package compose

import (
	"context"
	"fmt"

	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// NewPlotlineExtractWorkflow 创建剧情线提取工作流
//
// 功能：从小说大纲中提取剧情线，并进行润色，最后保存到数据库
//
// 工作流程：
//  1. init - 初始化工作流状态
//  2. extract - 从大纲提取剧情线
//  3. polish - 润色剧情线
//  4. saveToDB - 保存到数据库（需要用户审批）
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库实例
//
// 返回值：
//   - compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState]: 可执行的工作流
//   - error: 创建失败时返回错误
func NewPlotlineExtractWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	db *gorm.DB,
) (compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState], error) {
	// 创建数据仓库
	repo := repository.NewRepository(db)

	// 创建工作流图
	graph := compose.NewGraph[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initExtractNode))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(db, repo)))

	// 添加边（定义节点之间的连接）
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "extract")

	// 添加条件边（根据状态决定下一步）
	graph.AddBranch("extract", compose.NewGraphBranch(
		routeAfterExtract,
		map[string]bool{
			"polish":  true,
			"extract": true,
		},
	))

	graph.AddBranch("polish", compose.NewGraphBranch(
		routeAfterPolish,
		map[string]bool{
			"saveToDB": true,
			"polish":   true,
		},
	))

	graph.AddBranch("saveToDB", compose.NewGraphBranch(
		routeAfterSave,
		map[string]bool{
			"saveToDB":  true,
			compose.END: true,
		},
	))

	// 编译工作流，设置最大步数以支持重试
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// ExecutePlotlineExtractWorkflow 执行剧情线提取工作流
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *PlotlineExtractWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecutePlotlineExtractWorkflow(ctx context.Context, graph compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState], input *PlotlineExtractWorkflowInput) (*PlotlineExtractWorkflowState, error) {
	if input.Author == "" {
		return nil, fmt.Errorf("作者名称不能为空")
	}
	if input.NovelName == "" {
		return nil, fmt.Errorf("小说名称不能为空")
	}
	if input.Genre == "" {
		return nil, fmt.Errorf("小说类型不能为空")
	}
	if input.Outline == "" {
		return nil, fmt.Errorf("小说大纲不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📖 开始执行剧情线提取工作流\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说名称: %s\n", input.NovelName)
	fmt.Printf("作者: %s\n", input.Author)
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
	if result.SavedNovelID != 0 {
		fmt.Printf("小说ID: %d\n", result.SavedNovelID)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}
