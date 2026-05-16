package compose

import (
	"context"
	"encoding/json"
	"fmt"

	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// NewPlotlineRevisionWorkflow 创建剧情线修改工作流
//
// 功能：根据用户建议优化剧情线，并保存或更新到数据库
//
// 工作流程：
//   1. init - 从数据库加载小说信息、大纲和现有剧情线
//   2. revise - 根据用户建议优化剧情线（可以修改、删除、新增）
//   3. saveOrUpdateDB - 保存或更新到数据库
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - novelRepo: 小说仓库实例
//   - plotlineRepo: 剧情线仓库实例
//
// 返回值：
//   - compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]: 可执行的工作流
//   - error: 创建失败时返回错误
func NewPlotlineRevisionWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], error) {
	// 创建工作流图
	graph := compose.NewGraph[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initRevisionNode(novelRepo, plotlineRepo)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveOrUpdateDB", compose.InvokableLambda(saveOrUpdateDBNode(plotlineRepo)))

	// 添加边（定义节点之间的连接）
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "revise")

	// 添加条件边（根据状态决定下一步）
	graph.AddBranch("revise", compose.NewGraphBranch(
		routeAfterRevise,
		map[string]bool{
			"saveOrUpdateDB": true,
			"revise":         true,
		},
	))

	graph.AddBranch("saveOrUpdateDB", compose.NewGraphBranch(
		routeAfterSaveOrUpdate,
		map[string]bool{
			"saveOrUpdateDB": true,
			compose.END:      true,
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
	if len(result.SavedPlotlineIDs) > 0 {
		fmt.Printf("保存的剧情线数量: %d\n", len(result.SavedPlotlineIDs))
		fmt.Printf("剧情线ID列表: %v\n", result.SavedPlotlineIDs)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}

// NewPlotlineRevisionWorkflowV2 创建剧情线修改工作流（支持冲突检测和解决）
//
// 功能：根据用户建议优化剧情线，支持单个剧情线修改和冲突自动解决
//
// 工作流程：
//   1. init - 从数据库加载小说信息、大纲和剧情线
//   2. revise - 根据用户建议优化剧情线
//   3. detectOutlineConflict - 检测与大纲的冲突（仅单个剧情线修改时）
//   4. detectPlotlinesConflict - 检测与其他剧情线的冲突（仅单个剧情线修改时）
//   5. reviseOutline - 修改大纲以解决冲突（如果有大纲冲突）
//   6. reviseOtherPlotlines - 修改其他剧情线以解决冲突（如果有剧情线冲突）
//   7. saveOrUpdateDB - 保存或更新到数据库
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - novelRepo: 小说仓库实例
//   - plotlineRepo: 剧情线仓库实例
//
// 返回值：
//   - compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]: 可执行的工作流
//   - error: 创建失败时返回错误
func NewPlotlineRevisionWorkflowV2(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], error) {
	// 创建工作流图
	graph := compose.NewGraph[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initRevisionNodeV2(novelRepo, plotlineRepo)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("detectOutlineConflict", compose.InvokableLambda(detectOutlineConflictNode(ctx, chatModel)))
	graph.AddLambdaNode("detectPlotlinesConflict", compose.InvokableLambda(detectPlotlinesConflictNode(ctx, chatModel)))
	graph.AddLambdaNode("reviseOutline", compose.InvokableLambda(reviseOutlineNode(ctx, chatModel)))
	graph.AddLambdaNode("reviseOtherPlotlines", compose.InvokableLambda(reviseOtherPlotlinesNode(ctx, chatModel)))
	graph.AddLambdaNode("saveOrUpdateDB", compose.InvokableLambda(saveOrUpdateDBNodeV2(novelRepo, plotlineRepo)))

	// 添加边（定义节点之间的连接）
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "revise")

	// 添加条件边（根据状态决定下一步）
	graph.AddBranch("revise", compose.NewGraphBranch(
		routeAfterRevise,
		map[string]bool{
			"detectOutlineConflict": true,
			"saveOrUpdateDB":        true,
			"revise":                true,
		},
	))

	graph.AddBranch("detectOutlineConflict", compose.NewGraphBranch(
		routeAfterDetectOutlineConflict,
		map[string]bool{
			"detectPlotlinesConflict": true,
			"reviseOutline":           true,
			"detectOutlineConflict":   true,
		},
	))

	graph.AddBranch("detectPlotlinesConflict", compose.NewGraphBranch(
		routeAfterDetectPlotlinesConflict,
		map[string]bool{
			"reviseOtherPlotlines":    true,
			"saveOrUpdateDB":          true,
			"detectPlotlinesConflict": true,
		},
	))

	graph.AddBranch("reviseOutline", compose.NewGraphBranch(
		routeAfterReviseOutline,
		map[string]bool{
			"detectPlotlinesConflict": true,
			"reviseOutline":           true,
		},
	))

	graph.AddBranch("reviseOtherPlotlines", compose.NewGraphBranch(
		routeAfterReviseOtherPlotlines,
		map[string]bool{
			"saveOrUpdateDB":      true,
			"reviseOtherPlotlines": true,
		},
	))

	graph.AddBranch("saveOrUpdateDB", compose.NewGraphBranch(
		routeAfterSaveOrUpdate,
		map[string]bool{
			"saveOrUpdateDB": true,
			compose.END:      true,
		},
	))

	// 编译工作流，设置最大步数以支持重试和新节点
	return graph.Compile(ctx, compose.WithMaxRunSteps(100))
}

// ExecutePlotlineRevisionWorkflowV2 执行剧情线修改工作流（V2版本，支持冲突检测）
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *PlotlineRevisionWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecutePlotlineRevisionWorkflowV2(ctx context.Context, graph compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], input *PlotlineRevisionWorkflowInput) (*PlotlineRevisionWorkflowState, error) {
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}
	if input.UserSuggestion == "" {
		return nil, fmt.Errorf("用户修改建议不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✏️  开始执行剧情线修改工作流 (V2 - 支持冲突检测)\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说ID: %d\n", input.NovelID)
	if input.PlotlineID != "" {
		fmt.Printf("指定剧情线ID: %s\n", input.PlotlineID)
	} else {
		fmt.Printf("修改模式: 全部剧情线\n")
	}
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
	// 显示大纲冲突检测结果
	if result.OutlineConflictDetectionResult != "" {
		var outlineConflictResult map[string]interface{}
		if err := json.Unmarshal([]byte(result.OutlineConflictDetectionResult), &outlineConflictResult); err == nil {
			if hasConflicts, ok := outlineConflictResult["has_conflicts"].(bool); ok {
				if hasConflicts {
					if conflicts, ok := outlineConflictResult["conflicts"].([]interface{}); ok {
						fmt.Printf("检测到大纲冲突: %d 个\n", len(conflicts))
					}
					if result.RevisedOutline != "" {
						fmt.Printf("✓ 已解决大纲冲突\n")
					}
				} else {
					fmt.Printf("✓ 未检测到大纲冲突\n")
				}
			}
		}
	}

	// 显示剧情线冲突检测结果
	if result.PlotlinesConflictDetectionResults != "" {
		var plotlinesConflictResults []map[string]interface{}
		if err := json.Unmarshal([]byte(result.PlotlinesConflictDetectionResults), &plotlinesConflictResults); err == nil {
			totalConflicts := 0
			conflictPlotlines := 0
			for _, plotlineResult := range plotlinesConflictResults {
				if hasConflicts, ok := plotlineResult["has_conflicts"].(bool); ok && hasConflicts {
					conflictPlotlines++
					if conflicts, ok := plotlineResult["conflicts"].([]interface{}); ok {
						totalConflicts += len(conflicts)
					}
				}
			}
			if totalConflicts > 0 {
				fmt.Printf("检测到剧情线冲突: %d 个剧情线共 %d 个冲突\n", conflictPlotlines, totalConflicts)
				if result.RevisedOtherPlotlines != "" {
					fmt.Printf("✓ 已解决剧情线冲突\n")
				}
			} else {
				fmt.Printf("✓ 未检测到剧情线冲突\n")
			}
		}
	}
	if len(result.SavedPlotlineIDs) > 0 {
		fmt.Printf("保存的剧情线数量: %d\n", len(result.SavedPlotlineIDs))
		fmt.Printf("剧情线ID列表: %v\n", result.SavedPlotlineIDs)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}
