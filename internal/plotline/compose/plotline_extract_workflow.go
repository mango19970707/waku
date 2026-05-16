package compose

import (
	"context"
	"fmt"

	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// PlotlineExtractWorkflowInput 剧情线提取工作流的输入参数
type PlotlineExtractWorkflowInput struct {
	// NovelID 小说 ID
	NovelID uint
}

// PlotlineExtractWorkflowState 剧情线提取工作流的状态
type PlotlineExtractWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string

	// NovelID 小说 ID
	NovelID uint

	// NovelName 小说名称
	NovelName string

	// Author 作者
	Author string

	// Genre 小说类型
	Genre string

	// Outline 小说大纲
	Outline string

	// ExtractedPlotlines 提取的剧情线列表（JSON 格式）
	ExtractedPlotlines string

	// SavedPlotlineIDs 保存到数据库后的剧情线 ID 列表
	SavedPlotlineIDs []string

	// ErrorMsg 错误信息
	ErrorMsg string

	// RetryCount 重试次数
	RetryCount int

	// MaxRetries 最大重试次数
	MaxRetries int
}

// PlotlineRevisionWorkflowInput 剧情线修改工作流输入
type PlotlineRevisionWorkflowInput struct {
	// NovelID 小说ID（必填）
	NovelID uint
	// UserSuggestion 用户的修改建议（必填）
	UserSuggestion string
}

// PlotlineRevisionWorkflowState 剧情线修改工作流状态
type PlotlineRevisionWorkflowState struct {
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
	// Outline 小说大纲
	Outline string
	// ExistingPlotlines 现有剧情线列表（JSON 格式）
	ExistingPlotlines string
	// UserSuggestion 用户的修改建议
	UserSuggestion string
	// RevisedPlotlines 优化后的剧情线列表（JSON 格式）
	RevisedPlotlines string
	// SavedPlotlineIDs 保存到数据库后的剧情线 ID 列表
	SavedPlotlineIDs []string
	// ErrorMsg 错误信息
	ErrorMsg string
	// RetryCount 当前节点的重试次数
	RetryCount int
	// MaxRetries 最大重试次数
	MaxRetries int
}

// NewPlotlineExtractWorkflow 创建剧情线提取工作流
//
// 功能：从小说大纲中提取剧情线，并保存到数据库
//
// 工作流程：
//  1. init - 初始化工作流状态
//  2. extract - 从大纲提取剧情线（使用 JSON parser 严格校验）
//  3. saveToDB - 保存到数据库
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - novelRepo: 小说仓库实例
//   - plotlineRepo: 剧情线仓库实例
//
// 返回值：
//   - compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState]: 可执行的工作流
//   - error: 创建失败时返回错误
func NewPlotlineExtractWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState], error) {

	// 创建工作流图
	graph := compose.NewGraph[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initExtractNode(novelRepo)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(plotlineRepo)))

	// 添加边（定义节点之间的连接）
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "extract")

	// 添加条件边（根据状态决定下一步）
	graph.AddBranch("extract", compose.NewGraphBranch(
		routeAfterExtract,
		map[string]bool{
			"saveToDB": true,
			"extract":  true,
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
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📖 开始执行剧情线提取工作流\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("小说ID: %d\n", input.NovelID)
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
	if len(result.SavedPlotlineIDs) > 0 {
		fmt.Printf("保存的剧情线数量: %d\n", len(result.SavedPlotlineIDs))
		fmt.Printf("剧情线ID列表: %v\n", result.SavedPlotlineIDs)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}
