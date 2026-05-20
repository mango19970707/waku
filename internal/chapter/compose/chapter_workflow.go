package compose

import (
	"context"
	"fmt"
	"strings"

	"waku/internal/constant"
	characterrepository "waku/internal/character/repository"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	volumerepository "waku/internal/volume/repository"
	chapterrepository "waku/internal/chapter/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// ChapterWorkflowInput 是统一章节工作流的输入
type ChapterWorkflowInput struct {
	NovelID        uint
	VolumeID       uint
	ChapterID      uint
	UserSuggestion string
}

// ChapterWorkflowState 是统一章节工作流的状态
type ChapterWorkflowState struct {
	CurrentNode          string
	NovelID              uint
	VolumeID             uint
	ChapterID            uint
	NovelName            string
	Author               string
	Genre                string
	NovelOutline         string
	NovelWorldview       string
	CultivationSystem    string
	VolumeOutline        string
	VolumeTheme          string
	ExistingPlotlines    string
	ExistingCharacters   string
	ExistingChapters     string
	TargetChapterJSON    string
	UserSuggestion       string
	RevisionMode         bool
	GeneratedChapters    string
	PolishedChapters     string
	ReviewedChapters     string
	RevisedChapters      string
	SavedChapterIDs      []uint
	ErrorMsg             string
	RetryCount           int
	MaxRetries           int
}

// BuildChapterWorkflowGraph 构建统一章节工作流
func BuildChapterWorkflowGraph(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo novelrepository.Repository,
	volumeRepo volumerepository.VolumeRepository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	characterRepo characterrepository.CharacterRepository,
	chapterRepo chapterrepository.ChapterRepository,
) (compose.Runnable[*ChapterWorkflowInput, *ChapterWorkflowState], error) {
	if err := ensureAgentExecutor(); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[*ChapterWorkflowInput, *ChapterWorkflowState]()
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(novelRepo, volumeRepo, plotlineRepo, characterRepo, chapterRepo)))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(chapterRepo)))

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

// NewChapterWorkflow 创建统一章节工作流
func NewChapterWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo novelrepository.Repository,
	volumeRepo volumerepository.VolumeRepository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	characterRepo characterrepository.CharacterRepository,
	chapterRepo chapterrepository.ChapterRepository,
) (compose.Runnable[*ChapterWorkflowInput, *ChapterWorkflowState], error) {
	return BuildChapterWorkflowGraph(ctx, chatModel, novelRepo, volumeRepo, plotlineRepo, characterRepo, chapterRepo)
}

// ExecuteChapterWorkflow 执行统一章节工作流
func ExecuteChapterWorkflow(
	ctx context.Context,
	graph compose.Runnable[*ChapterWorkflowInput, *ChapterWorkflowState],
	input *ChapterWorkflowInput,
) (*ChapterWorkflowState, error) {
	if input == nil {
		return nil, fmt.Errorf("工作流输入不能为空")
	}
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}
	if input.VolumeID == 0 {
		return nil, fmt.Errorf("分卷ID不能为空")
	}
	return graph.Invoke(ctx, input)
}

func newWorkflowState(input *ChapterWorkflowInput) *ChapterWorkflowState {
	return &ChapterWorkflowState{
		CurrentNode:     "init_success",
		NovelID:         input.NovelID,
		VolumeID:        input.VolumeID,
		ChapterID:       input.ChapterID,
		UserSuggestion:  strings.TrimSpace(input.UserSuggestion),
		RevisionMode:    input.ChapterID > 0 && strings.TrimSpace(input.UserSuggestion) != "",
		SavedChapterIDs: []uint{},
		MaxRetries:      constant.MaxRetries,
	}
}
