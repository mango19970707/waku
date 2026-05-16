package compose

import (
	"context"
	"fmt"
	"strings"

	"waku/internal/constant"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	volumerepository "waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// VolumeWorkflowInput 是统一分卷工作流的输入。
type VolumeWorkflowInput struct {
	NovelID        uint
	VolumeID       uint
	UserSuggestion string
}

// VolumeWorkflowState 是统一分卷工作流的状态。
type VolumeWorkflowState struct {
	CurrentNode       string
	NovelID           uint
	VolumeID          uint
	NovelName         string
	Author            string
	Genre             string
	Outline           string
	ExistingPlotlines string
	ExistingVolumes   string
	TargetVolumeJSON  string
	UserSuggestion    string
	RevisionMode      bool
	GeneratedVolumes  string
	PolishedVolumes   string
	ReviewedVolumes   string
	RevisedVolumes    string
	SavedVolumeIDs    []uint
	ErrorMsg          string
	RetryCount        int
	MaxRetries        int
}

// BuildVolumeWorkflowGraph 构建统一分卷工作流。
func BuildVolumeWorkflowGraph(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo novelrepository.Repository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	volumeRepo volumerepository.VolumeRepository,
) (compose.Runnable[*VolumeWorkflowInput, *VolumeWorkflowState], error) {
	if err := ensureAgentExecutor(); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[*VolumeWorkflowInput, *VolumeWorkflowState]()
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(novelRepo, plotlineRepo, volumeRepo)))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(volumeRepo)))

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

// NewVolumeWorkflow 创建统一分卷工作流。
func NewVolumeWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo novelrepository.Repository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	volumeRepo volumerepository.VolumeRepository,
) (compose.Runnable[*VolumeWorkflowInput, *VolumeWorkflowState], error) {
	return BuildVolumeWorkflowGraph(ctx, chatModel, novelRepo, plotlineRepo, volumeRepo)
}

// ExecuteVolumeWorkflow 执行统一分卷工作流。
func ExecuteVolumeWorkflow(
	ctx context.Context,
	graph compose.Runnable[*VolumeWorkflowInput, *VolumeWorkflowState],
	input *VolumeWorkflowInput,
) (*VolumeWorkflowState, error) {
	if input == nil {
		return nil, fmt.Errorf("工作流输入不能为空")
	}
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}
	return graph.Invoke(ctx, input)
}

func newWorkflowState(input *VolumeWorkflowInput) *VolumeWorkflowState {
	return &VolumeWorkflowState{
		CurrentNode:    "init_success",
		NovelID:        input.NovelID,
		VolumeID:       input.VolumeID,
		UserSuggestion: strings.TrimSpace(input.UserSuggestion),
		RevisionMode:   input.VolumeID > 0 && strings.TrimSpace(input.UserSuggestion) != "",
		SavedVolumeIDs: []uint{},
		MaxRetries:     constant.MaxRetries,
	}
}
