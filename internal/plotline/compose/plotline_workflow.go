package compose

import (
	"context"
	"fmt"
	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// PlotlineWorkflowInput 是统一剧情线工作流输入。
type PlotlineWorkflowInput struct {
	NovelID        uint
	PlotlineID     uint
	UserSuggestion string
}

// PlotlineWorkflowState 是统一剧情线工作流状态。
type PlotlineWorkflowState struct {
	CurrentNode        string
	NovelID            uint
	PlotlineID         uint
	NovelName          string
	Author             string
	Genre              string
	Outline            string
	UserSuggestion     string
	ExistingPlotlines  string
	GeneratedPlotlines string
	PolishedPlotlines  string
	ReviewedPlotlines  string
	RevisedPlotlines   string
	SavedPlotlineIDs   []uint
	ErrorMsg           string
	RetryCount         int
	MaxRetries         int
}

type PlotlineExtractWorkflowInput = PlotlineWorkflowInput
type PlotlineExtractWorkflowState = PlotlineWorkflowState
type PlotlineRevisionWorkflowInput = PlotlineWorkflowInput
type PlotlineRevisionWorkflowState = PlotlineWorkflowState

// BuildPlotlineWorkflowGraph 构建统一剧情线工作流。
func BuildPlotlineWorkflowGraph(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineWorkflowInput, *PlotlineWorkflowState], error) {
	if err := ensureAgentExecutor(); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[*PlotlineWorkflowInput, *PlotlineWorkflowState]()

	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(novelRepo, plotlineRepo)))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(plotlineRepo)))

	graph.AddEdge(compose.START, "init")
	graph.AddBranch("init", compose.NewGraphBranch(routeAfterInit, map[string]bool{
		"generate": true,
		"polish":   true,
	}))
	graph.AddBranch("generate", compose.NewGraphBranch(routeAfterGenerate, map[string]bool{
		"generate": true,
		"polish":   true,
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

// ExecutePlotlineWorkflow 执行统一剧情线工作流。
func ExecutePlotlineWorkflow(
	ctx context.Context,
	graph compose.Runnable[*PlotlineWorkflowInput, *PlotlineWorkflowState],
	input *PlotlineWorkflowInput,
) (*PlotlineWorkflowState, error) {
	logWorkflowInput(ctx, input)

	var (
		state *PlotlineWorkflowState
		err   error
	)
	defer func() {
		logWorkflowOutput(ctx, state, err)
	}()

	if input == nil {
		err = fmt.Errorf("工作流输入不能为空")
		return nil, err
	}

	if input.NovelID == 0 && input.PlotlineID == 0 {
		err = fmt.Errorf("小说 ID 和剧情线 ID 不能同时为空")
		return nil, err
	}

	if input.NovelID > 0 && input.PlotlineID == 0 {
		state, err = graph.Invoke(ctx, input)
		return state, err
	}

	if input.PlotlineID > 0 && input.UserSuggestion != "" && input.NovelID == 0 {
		state, err = graph.Invoke(ctx, input)
		return state, err
	}

	err = fmt.Errorf("输入不符合工作流模式要求")
	return nil, err
}

// NewPlotlineWorkflow 创建统一剧情线工作流。
func NewPlotlineWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineWorkflowInput, *PlotlineWorkflowState], error) {
	return BuildPlotlineWorkflowGraph(ctx, chatModel, novelRepo, plotlineRepo)
}

// NewPlotlineExtractWorkflow 保留旧入口，映射到统一工作流。
func NewPlotlineExtractWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState], error) {
	return BuildPlotlineWorkflowGraph(ctx, chatModel, novelRepo, plotlineRepo)
}

// ExecutePlotlineExtractWorkflow 保留旧入口，映射到统一工作流。
func ExecutePlotlineExtractWorkflow(
	ctx context.Context,
	graph compose.Runnable[*PlotlineExtractWorkflowInput, *PlotlineExtractWorkflowState],
	input *PlotlineExtractWorkflowInput,
) (*PlotlineExtractWorkflowState, error) {
	return ExecutePlotlineWorkflow(ctx, graph, input)
}

// NewPlotlineRevisionWorkflow 保留旧入口，映射到统一工作流。
func NewPlotlineRevisionWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], error) {
	return BuildPlotlineWorkflowGraph(ctx, chatModel, novelRepo, plotlineRepo)
}

// ExecutePlotlineRevisionWorkflow 保留旧入口，映射到统一工作流。
func ExecutePlotlineRevisionWorkflow(
	ctx context.Context,
	graph compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState],
	input *PlotlineRevisionWorkflowInput,
) (*PlotlineRevisionWorkflowState, error) {
	return ExecutePlotlineWorkflow(ctx, graph, input)
}

// NewPlotlineRevisionWorkflowV2 保留旧入口，映射到统一工作流。
func NewPlotlineRevisionWorkflowV2(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	plotlineRepo repository.PlotlineRepository,
) (compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState], error) {
	return BuildPlotlineWorkflowGraph(ctx, chatModel, novelRepo, plotlineRepo)
}

// ExecutePlotlineRevisionWorkflowV2 保留旧入口，映射到统一工作流。
func ExecutePlotlineRevisionWorkflowV2(
	ctx context.Context,
	graph compose.Runnable[*PlotlineRevisionWorkflowInput, *PlotlineRevisionWorkflowState],
	input *PlotlineRevisionWorkflowInput,
) (*PlotlineRevisionWorkflowState, error) {
	return ExecutePlotlineWorkflow(ctx, graph, input)
}
