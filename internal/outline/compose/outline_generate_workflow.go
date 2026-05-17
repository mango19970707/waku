package compose

import (
	"context"
	"fmt"
	"waku/internal/outline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// OutlineWorkflowState 是统一大纲工作流的状态载体。
type OutlineWorkflowState struct {
	CurrentNode       string
	NovelID           uint
	NovelName         string
	Author            string
	Genre             string
	OriginalOutline   string
	UserSuggestion    string
	GeneratedOutline  map[string]any
	ExpandedOutline   map[string]any
	PolishedOutline   map[string]any
	ReviewSuggestions string
	RevisedOutline    map[string]any
	NovelTitle        string
	Synopsis          string
	Outline           string
	SavedNovelID      uint
	ErrorMsg          string
	RetryCount        int
	MaxRetries        int
}

// OutlineWorkflowInput 是统一大纲工作流的输入。
type OutlineWorkflowInput struct {
	NovelID         uint
	NovelName       string
	Author          string
	Genre           string
	OriginalOutline string
	UserSuggestion  string
}

// AgentExecutor 定义工作流依赖的 Agent 执行接口。
type AgentExecutor interface {
	GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error)
	ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error)
	PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, userSuggestion, genre string) (string, error)
	ReviewOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, userSuggestion, genre string) (string, error)
	ReviseOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, suggestions, genre string) (string, error)
}

type defaultAgentExecutor struct{}

var agentExecutor AgentExecutor = &defaultAgentExecutor{}

func (e *defaultAgentExecutor) GenerateOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ExpandOutline(ctx context.Context, chatModel einoModel.BaseChatModel, briefOutline, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) PolishOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, userSuggestion, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ReviewOutline(ctx context.Context, chatModel einoModel.BaseChatModel, outline, userSuggestion, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

func (e *defaultAgentExecutor) ReviseOutline(ctx context.Context, chatModel einoModel.BaseChatModel, originalOutline, suggestions, genre string) (string, error) {
	return "", fmt.Errorf("agent executor not implemented")
}

// RegisterAgentExecutor 用于替换默认 Agent 执行器，便于测试和扩展。
func RegisterAgentExecutor(executor AgentExecutor) {
	agentExecutor = executor
}

// BuildOutlineWorkflowGraph 构建统一大纲工作流。
func BuildOutlineWorkflowGraph(ctx context.Context, chatModel einoModel.BaseChatModel, repo repository.Repository) (compose.Runnable[*OutlineWorkflowInput, *OutlineWorkflowState], error) {
	graph := compose.NewGraph[*OutlineWorkflowInput, *OutlineWorkflowState]()

	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(repo)))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("expand", compose.InvokableLambda(expandNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(repo)))

	graph.AddEdge(compose.START, "init")

	graph.AddBranch("init", compose.NewGraphBranch(routeAfterInit, map[string]bool{
		"generate": true,
		"polish":   true,
	}))
	graph.AddBranch("generate", compose.NewGraphBranch(routeAfterGenerate, map[string]bool{
		"generate": true,
		"expand":   true,
	}))
	graph.AddBranch("expand", compose.NewGraphBranch(routeAfterExpand, map[string]bool{
		"expand": true,
		"polish": true,
	}))
	graph.AddBranch("polish", compose.NewGraphBranch(routeAfterPolish, map[string]bool{
		"polish": true,
		"review": true,
	}))
	graph.AddBranch("review", compose.NewGraphBranch(routeAfterReview, map[string]bool{
		"review": true,
		"revise": true,
	}))
	graph.AddBranch("revise", compose.NewGraphBranch(routeAfterRevise, map[string]bool{
		"revise":   true,
		"saveToDB": true,
	}))
	graph.AddBranch("saveToDB", compose.NewGraphBranch(routeAfterSave, map[string]bool{
		"saveToDB":  true,
		compose.END: true,
	}))

	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

// ExecuteOutlineWorkflow 执行统一大纲工作流。
func ExecuteOutlineWorkflow(ctx context.Context, graph compose.Runnable[*OutlineWorkflowInput, *OutlineWorkflowState], input *OutlineWorkflowInput) (*OutlineWorkflowState, error) {
	logWorkflowInput(ctx, input)

	if input == nil {
		return nil, fmt.Errorf("工作流输入不能为空")
	}

	var (
		state *OutlineWorkflowState
		err   error
	)
	defer func() {
		logWorkflowOutput(ctx, state, err)
	}()

	isQuickRevision := input.NovelID > 0 && input.UserSuggestion != ""
	if isQuickRevision {
		if input.Genre == "" {
			err = fmt.Errorf("快速预修改模式下小说类型不能为空")
			return nil, err
		}
		state, err = graph.Invoke(ctx, input)
		return state, err
	}

	if input.Author == "" {
		err = fmt.Errorf("完整生成模式下作者名称不能为空")
		return nil, err
	}
	if input.Genre == "" {
		err = fmt.Errorf("完整生成模式下小说类型不能为空")
		return nil, err
	}
	if input.OriginalOutline == "" {
		err = fmt.Errorf("完整生成模式下原始大纲不能为空")
		return nil, err
	}

	state, err = graph.Invoke(ctx, input)
	return state, err
}
