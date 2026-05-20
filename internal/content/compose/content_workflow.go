package compose

import (
	"context"
	"fmt"
	"strings"

	characterrepository "waku/internal/character/repository"
	"waku/internal/constant"
	contentrepository "waku/internal/content/repository"
	novelrepository "waku/internal/outline/repository"
	plotlinerepository "waku/internal/plotline/repository"
	volumerepository "waku/internal/volume/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

// ContentWorkflowInput 是统一正文工作流的输入。
type ContentWorkflowInput struct {
	NovelID        uint
	VolumeID       uint
	ChapterID      uint
	ChapterIDs     []uint
	UserSuggestion string
}

// ContentWorkflowState 保存正文工作流跨节点状态，正文内容只在阶段字段中流转。
type ContentWorkflowState struct {
	CurrentNode       string
	NovelID           uint
	VolumeID          uint
	ChapterID         uint
	PendingChapterIDs []uint
	NovelName         string
	Author            string
	Genre             string
	NovelContext      string
	VolumeContext     string
	ChapterContext    string
	PreviousContext   string
	NextContext       string
	Characters        string
	Plotlines         string
	TargetContent     string
	UserSuggestion    string
	FastPolishMode    bool
	GeneratedContent  string
	PolishedContent   string
	ReviewedContent   string
	RevisedContent    string
	NeedRevise        bool
	SavedChapterIDs   []uint
	ErrorMsg          string
	RetryCount        int
	MaxRetries        int
}

// BuildContentWorkflowGraph 构建统一正文工作流图。
func BuildContentWorkflowGraph(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo novelrepository.Repository,
	volumeRepo volumerepository.VolumeRepository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	characterRepo characterrepository.CharacterRepository,
	contentRepo contentrepository.ContentRepository,
) (compose.Runnable[*ContentWorkflowInput, *ContentWorkflowState], error) {
	if err := ensureAgentExecutor(); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[*ContentWorkflowInput, *ContentWorkflowState]()
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(novelRepo, volumeRepo, plotlineRepo, characterRepo, contentRepo)))
	graph.AddLambdaNode("generate", compose.InvokableLambda(generateNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(contentRepo)))
	graph.AddLambdaNode("prepareNext", compose.InvokableLambda(prepareNextNode(contentRepo)))

	graph.AddEdge(compose.START, "init")
	graph.AddBranch("init", compose.NewGraphBranch(routeAfterInit, map[string]bool{"generate": true, "polish": true, compose.END: true}))
	graph.AddBranch("generate", compose.NewGraphBranch(routeAfterGenerate, map[string]bool{"generate": true, "polish": true, compose.END: true}))
	graph.AddBranch("polish", compose.NewGraphBranch(routeAfterPolish, map[string]bool{"polish": true, "review": true, compose.END: true}))
	graph.AddBranch("review", compose.NewGraphBranch(routeAfterReview, map[string]bool{"review": true, "revise": true, "saveToDB": true, compose.END: true}))
	graph.AddBranch("revise", compose.NewGraphBranch(routeAfterRevise, map[string]bool{"revise": true, "saveToDB": true, compose.END: true}))
	graph.AddBranch("saveToDB", compose.NewGraphBranch(routeAfterSave, map[string]bool{"saveToDB": true, "prepareNext": true, compose.END: true}))
	graph.AddBranch("prepareNext", compose.NewGraphBranch(routeAfterPrepareNext, map[string]bool{"generate": true, compose.END: true}))

	return graph.Compile(ctx, compose.WithMaxRunSteps(200))
}

// NewContentWorkflow 创建统一正文工作流。
func NewContentWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo novelrepository.Repository,
	volumeRepo volumerepository.VolumeRepository,
	plotlineRepo plotlinerepository.PlotlineRepository,
	characterRepo characterrepository.CharacterRepository,
	contentRepo contentrepository.ContentRepository,
) (compose.Runnable[*ContentWorkflowInput, *ContentWorkflowState], error) {
	return BuildContentWorkflowGraph(ctx, chatModel, novelRepo, volumeRepo, plotlineRepo, characterRepo, contentRepo)
}

// ExecuteContentWorkflow 执行统一正文工作流，并在入口校验必填参数。
func ExecuteContentWorkflow(ctx context.Context, graph compose.Runnable[*ContentWorkflowInput, *ContentWorkflowState], input *ContentWorkflowInput) (*ContentWorkflowState, error) {
	if input == nil {
		return nil, fmt.Errorf("工作流输入不能为空")
	}
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说ID不能为空")
	}
	if input.VolumeID == 0 {
		return nil, fmt.Errorf("分卷ID不能为空")
	}
	if input.ChapterID > 0 && len(input.ChapterIDs) > 0 {
		return nil, fmt.Errorf("chapter_id 与 chapter_ids 不能同时传入")
	}
	return graph.Invoke(ctx, input)
}

func newWorkflowState(input *ContentWorkflowInput) *ContentWorkflowState {
	return &ContentWorkflowState{
		CurrentNode:       "init_success",
		NovelID:           input.NovelID,
		VolumeID:          input.VolumeID,
		ChapterID:         input.ChapterID,
		PendingChapterIDs: append([]uint{}, input.ChapterIDs...),
		UserSuggestion:    strings.TrimSpace(input.UserSuggestion),
		SavedChapterIDs:   []uint{},
		MaxRetries:        constant.MaxRetries,
	}
}
