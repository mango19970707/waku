package compose

import (
	"context"
	"fmt"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"gorm.io/gorm"

	"waku/internal/character/repository"
)

// CharacterRevisionWorkflowInput 人物修改工作流输入
type CharacterRevisionWorkflowInput struct {
	NovelID        uint   // 小说ID
	CharacterID    string // 角色ID（可选，指定时只优化该角色）
	UserSuggestion string // 用户建议
}

// CharacterRevisionWorkflowState 人物修改工作流状态
type CharacterRevisionWorkflowState struct {
	CurrentNode        string   // 当前节点
	NovelID            uint     // 小说ID
	CharacterID        string   // 角色ID（可选，指定时只优化该角色）
	NovelName          string   // 小说名称
	Author             string   // 作者
	Genre              string   // 小说类型
	Outline            string   // 小说大纲
	ExistingCharacters string   // 现有人物信息（JSON）
	UserSuggestion     string   // 用户建议
	ReviewSuggestions  string   // 审核意见（JSON）
	RevisedCharacters  string   // 优化后的人物信息（JSON）
	SavedCharacterIDs  []string // 保存后的人物ID列表
	ErrorMsg           string   // 错误信息
	RetryCount         int      // 重试次数
	MaxRetries         int      // 最大重试次数
}

// NewCharacterRevisionWorkflow 创建人物修改工作流
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
func NewCharacterRevisionWorkflow(
	ctx context.Context,
	db *gorm.DB,
	chatModel einoModel.BaseChatModel,
	characterRepo repository.CharacterRepository,
) (compose.Runnable[*CharacterRevisionWorkflowInput, *CharacterRevisionWorkflowState], error) {
	// 创建工作流图
	graph := compose.NewGraph[*CharacterRevisionWorkflowInput, *CharacterRevisionWorkflowState]()

	// 添加节点
	graph.AddLambdaNode("init", compose.InvokableLambda(initRevisionNode(db, characterRepo)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveOrUpdateDB", compose.InvokableLambda(saveOrUpdateDBNode(db, characterRepo)))

	// 添加边
	graph.AddEdge(compose.START, "init")
	graph.AddEdge("init", "review")

	// 添加条件边
	graph.AddBranch("review", compose.NewGraphBranch(
		routeAfterReview,
		map[string]bool{
			"revise": true,
		},
	))

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

	// 编译工作流
	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// RunCharacterRevisionWorkflow 运行人物修改工作流
//
// 参数：
//   - ctx: 上下文对象
//   - workflow: 工作流实例
//   - input: 输入参数
//
// 返回：
//   - *CharacterRevisionWorkflowState: 最终状态
//   - error: 错误信息
func RunCharacterRevisionWorkflow(
	ctx context.Context,
	workflow compose.Runnable[*CharacterRevisionWorkflowInput, *CharacterRevisionWorkflowState],
	input *CharacterRevisionWorkflowInput,
) (*CharacterRevisionWorkflowState, error) {
	fmt.Println("\n==================== 开始人物修改工作流 ====================")

	// 执行工作流
	state, err := workflow.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("工作流执行失败: %w", err)
	}

	fmt.Println("\n==================== 人物修改工作流完成 ====================")
	fmt.Printf("最终状态: %s\n", state.CurrentNode)
	fmt.Printf("保存的人物数量: %d\n", len(state.SavedCharacterIDs))

	if state.ErrorMsg != "" {
		fmt.Printf("错误信息: %s\n", state.ErrorMsg)
	}

	return state, nil
}
