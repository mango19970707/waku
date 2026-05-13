package compose

import (
	"context"
	"fmt"
	"waku/internal/volume/agents"
	"waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"
)

// VolumePartitionWorkflowState 分卷工作流状态
type VolumePartitionWorkflowState struct {
	// CurrentNode 当前节点名称
	CurrentNode string
	// NovelID 小说ID
	NovelID uint
	// NovelName 小说名称
	NovelName string
	// Genre 小说类型
	Genre string
	// NovelOutline 小说大纲
	NovelOutline string
	// PartitionedVolumes 分卷结果（文本）
	PartitionedVolumes string
	// ExtractedVolumes 提取的结构化卷信息
	ExtractedVolumes *agents.VolumePartitionOutput
	// SavedVolumeIDs 保存后的卷ID列表
	SavedVolumeIDs []uint
	// ChapterPartitionMap 卷ID -> 章节列表
	ChapterPartitionMap map[uint][]agents.ChapterInfo
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

// VolumePartitionWorkflowInput 分卷工作流输入
type VolumePartitionWorkflowInput struct {
	// NovelID 小说ID
	NovelID uint
}

// AgentExecutor 定义 Agent 执行接口，用于解耦
type AgentExecutor interface {
	PartitionVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, novelOutline, genre string, suggestedVolumeCount int) (string, error)
	ExtractVolumeInfo(ctx context.Context, chatModel einoModel.BaseChatModel, partitionedText, genre string) (*agents.VolumePartitionOutput, error)
	PartitionChapters(ctx context.Context, chatModel einoModel.BaseChatModel, volumeOutline, genre string, volumeNumber, suggestedChapterCount int) (string, error)
	ExtractChapterInfo(ctx context.Context, chatModel einoModel.BaseChatModel, partitionedText, genre string) (*agents.ChapterPartitionOutput, error)
}

// defaultAgentExecutor 默认的 Agent 执行器实现
type defaultAgentExecutor struct{}

// SetAgentExecutor 设置自定义的 Agent 执行器（用于测试或扩展）
var agentExecutor AgentExecutor = &defaultAgentExecutor{}

func (e *defaultAgentExecutor) PartitionVolumes(ctx context.Context, chatModel einoModel.BaseChatModel, novelOutline, genre string, suggestedVolumeCount int) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ExtractVolumeInfo(ctx context.Context, chatModel einoModel.BaseChatModel, partitionedText, genre string) (*agents.VolumePartitionOutput, error) {
	return nil, fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) PartitionChapters(ctx context.Context, chatModel einoModel.BaseChatModel, volumeOutline, genre string, volumeNumber, suggestedChapterCount int) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ExtractChapterInfo(ctx context.Context, chatModel einoModel.BaseChatModel, partitionedText, genre string) (*agents.ChapterPartitionOutput, error) {
	return nil, fmt.Errorf("agent executor not implemented")
}

// RegisterAgentExecutor 注册 Agent 执行器（由 agents 包调用）
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// ==================== 构建 Graph ====================

// BuildVolumePartitionWorkflowGraph 构建分卷工作流 Graph
//
// 功能：创建一个完整的分卷工作流，包括分卷、提取、保存、分章四个步骤
//
// 参数：
//   - ctx: 上下文
//   - chatModel: 聊天模型实例
//   - db: 数据库连接
//   - volumeRepo: 卷仓储实现
//   - chapterRepo: 章节仓储实现
//
// 返回值：
//   - compose.Runnable: 可执行的 Graph
//   - error: 创建失败时返回错误
//
// 工作流程：
//  1. 初始化：从数据库加载小说信息
//  2. 分卷：将小说大纲划分为卷（失败重试3次）
//  3. 提取：提取结构化的卷信息（失败重试3次）
//  4. 保存：保存卷到数据库（失败重试3次，需用户审批）
//  5. 分章：为每个卷划分章节并保存（失败重试3次）
//
// 特性：
//   - 每个步骤的输出保存到 data/content/{小说名称}/volume/{阶段}.md
//   - 支持断点续传：如果文件已存在，直接加载而不重新生成
//   - 工具调用失败自动重试3次
//   - 保存前需要用户审批
func BuildVolumePartitionWorkflowGraph(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	db *gorm.DB,
	volumeRepo repository.VolumeRepository,
	chapterRepo repository.ChapterRepository,
) (compose.Runnable[*VolumePartitionWorkflowInput, *VolumePartitionWorkflowState], error) {
	graph := compose.NewGraph[*VolumePartitionWorkflowInput, *VolumePartitionWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initVolumePartitionNode(db)))
	graph.AddLambdaNode("partition", compose.InvokableLambda(partitionVolumesNode(ctx, chatModel)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractVolumesNode(ctx, chatModel)))
	graph.AddLambdaNode("saveVolumes", compose.InvokableLambda(saveVolumesToDBNode(db, volumeRepo)))
	graph.AddLambdaNode("partitionChapters", compose.InvokableLambda(partitionChaptersNode(ctx, chatModel, chapterRepo)))

	// 添加边
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "partition")

	// 添加条件分支
	graph.AddBranch("partition", compose.NewGraphBranch(
		routeAfterPartition,
		map[string]bool{
			"partition": true,
			"extract":   true,
		},
	))

	graph.AddBranch("extract", compose.NewGraphBranch(
		routeAfterExtract,
		map[string]bool{
			"extract":     true,
			"saveVolumes": true,
		},
	))

	graph.AddBranch("saveVolumes", compose.NewGraphBranch(
		routeAfterSaveVolumes,
		map[string]bool{
			"saveVolumes":       true,
			"partitionChapters": true,
		},
	))

	graph.AddBranch("partitionChapters", compose.NewGraphBranch(
		routeAfterPartitionChapters,
		map[string]bool{
			"partitionChapters": true,
			compose.END:         true,
		},
	))

	// 编译 Graph，设置最大步数以支持重试
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// ExecuteVolumePartitionWorkflow 执行分卷工作流
//
// 参数：
//   - ctx: 上下文
//   - graph: 工作流 Graph
//   - input: 输入参数
//
// 返回值：
//   - *VolumePartitionWorkflowState: 最终状态
//   - error: 执行失败时返回错误
func ExecuteVolumePartitionWorkflow(
	ctx context.Context,
	graph compose.Runnable[*VolumePartitionWorkflowInput, *VolumePartitionWorkflowState],
	input *VolumePartitionWorkflowInput,
) (*VolumePartitionWorkflowState, error) {
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📚 开始执行分卷工作流\n")
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
	if len(result.SavedVolumeIDs) > 0 {
		fmt.Printf("创建卷数: %d\n", len(result.SavedVolumeIDs))
		totalChapters := 0
		for _, chapters := range result.ChapterPartitionMap {
			totalChapters += len(chapters)
		}
		fmt.Printf("创建章节数: %d\n", totalChapters)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return result, nil
}
