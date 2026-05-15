package compose

import (
	"context"
	"fmt"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"

	"waku/internal/character/repository"
)

// CharacterExtractWorkflowInput 人物提取工作流输入
type CharacterExtractWorkflowInput struct {
	NovelID uint // 小说ID
}

// CharacterExtractWorkflowState 人物提取工作流状态
type CharacterExtractWorkflowState struct {
	CurrentNode         string   // 当前节点
	NovelID             uint     // 小说ID
	NovelName           string   // 小说名称
	Author              string   // 作者
	Genre               string   // 小说类型
	Outline             string   // 小说大纲
	ExtractedCharacters string   // 提取的人物信息（JSON）
	SavedCharacterIDs   []string // 保存后的人物ID列表
	ErrorMsg            string   // 错误信息
	RetryCount          int      // 重试次数
	MaxRetries          int      // 最大重试次数
}

// NewCharacterExtractWorkflow 创建人物提取工作流
//
// 参数：
//   - ctx: 上下文对象
//   - db: 数据库连接
//   - chatModel: 聊天模型
//   - characterRepo: 人物仓储
//
// 返回：
//   - compose.Runnable: 工作流实例
//   - error: 错误信息
func NewCharacterExtractWorkflow(
	ctx context.Context,
	db *gorm.DB,
	chatModel einoModel.BaseChatModel,
	characterRepo repository.CharacterRepository,
) (compose.Runnable[*CharacterExtractWorkflowInput, *CharacterExtractWorkflowState], error) {
	// 创建工作流图
	graph := compose.NewGraph[*CharacterExtractWorkflowInput, *CharacterExtractWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initExtractNode(db)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(db, characterRepo)))

	// 添加边
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "extract")

	// 添加条件边
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

	// 编译工作流
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// RunCharacterExtractWorkflow 运行人物提取工作流
//
// 参数：
//   - ctx: 上下文对象
//   - workflow: 工作流实例
//   - input: 输入参数
//
// 返回：
//   - *CharacterExtractWorkflowState: 最终状态
//   - error: 错误信息
func RunCharacterExtractWorkflow(
	ctx context.Context,
	workflow compose.Runnable[*CharacterExtractWorkflowInput, *CharacterExtractWorkflowState],
	input *CharacterExtractWorkflowInput,
) (*CharacterExtractWorkflowState, error) {
	fmt.Println("\n==================== 开始人物提取工作流 ====================")

	// 执行工作流
	state, err := workflow.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("工作流执行失败: %w", err)
	}

	fmt.Println("\n==================== 人物提取工作流完成 ====================")
	fmt.Printf("最终状态: %s\n", state.CurrentNode)
	fmt.Printf("保存的人物数量: %d\n", len(state.SavedCharacterIDs))

	if state.ErrorMsg != "" {
		fmt.Printf("错误信息: %s\n", state.ErrorMsg)
	}

	return state, nil
}
