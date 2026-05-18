package compose

import (
	"context"
	"fmt"
	"waku/internal/character/repository"
	"waku/internal/constant"
	plotlineRepo "waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
)

type CharacterWorkflowInput struct {
	NovelID        uint
	CharacterID    uint
	UserSuggestion string
}

type CharacterWorkflowState struct {
	CurrentNode         string
	NovelID             uint
	CharacterID         uint
	NovelName           string
	Author              string
	Genre               string
	Outline             string
	Plotlines           string // 所有剧情线的 JSON 格式数据
	UserSuggestion      string
	ExistingCharacters  map[string]any
	ExtractedCharacters map[string]any
	PolishedCharacters  map[string]any
	ReviewedCharacters  map[string]any
	RevisedCharacters   map[string]any
	IDMapping           map[uint]uint // 临时ID → 真实ID 的映射
	MappedCharacters    map[string]any
	SavedCharacterIDs   []uint
	ErrorMsg            string
	RetryCount          int
	MaxRetries          int
}

func BuildCharacterWorkflowGraph(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	characterRepo repository.CharacterRepository,
	plotlineRepository plotlineRepo.PlotlineRepository,
) (compose.Runnable[*CharacterWorkflowInput, *CharacterWorkflowState], error) {
	if err := ensureAgentExecutor(); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[*CharacterWorkflowInput, *CharacterWorkflowState]()
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(novelRepo, characterRepo, plotlineRepository)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("preSave", compose.InvokableLambda(preSaveNode(characterRepo)))
	graph.AddLambdaNode("mapIDs", compose.InvokableLambda(mapIDsNode()))
	graph.AddLambdaNode("updateRelationships", compose.InvokableLambda(updateRelationshipsNode(characterRepo)))

	graph.AddEdge(compose.START, "init")
	graph.AddBranch("init", compose.NewGraphBranch(routeAfterInit, map[string]bool{
		"extract": true,
		"polish":  true,
	}))
	graph.AddBranch("extract", compose.NewGraphBranch(routeAfterExtract, map[string]bool{
		"extract": true,
		"polish":  true,
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
		"revise":  true,
		"preSave": true,
	}))
	graph.AddBranch("preSave", compose.NewGraphBranch(routeAfterPreSave, map[string]bool{
		"preSave":    true,
		"mapIDs":     true,
		compose.END:  true,
	}))
	graph.AddBranch("mapIDs", compose.NewGraphBranch(routeAfterMapIDs, map[string]bool{
		"mapIDs":              true,
		"updateRelationships": true,
	}))
	graph.AddBranch("updateRelationships", compose.NewGraphBranch(routeAfterUpdateRelationships, map[string]bool{
		"updateRelationships": true,
		compose.END:           true,
	}))

	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

func NewCharacterWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	characterRepo repository.CharacterRepository,
	plotlineRepository plotlineRepo.PlotlineRepository,
) (compose.Runnable[*CharacterWorkflowInput, *CharacterWorkflowState], error) {
	return BuildCharacterWorkflowGraph(ctx, chatModel, novelRepo, characterRepo, plotlineRepository)
}

func ExecuteCharacterWorkflow(
	ctx context.Context,
	graph compose.Runnable[*CharacterWorkflowInput, *CharacterWorkflowState],
	input *CharacterWorkflowInput,
) (*CharacterWorkflowState, error) {
	logWorkflowInput(ctx, input)

	var (
		state *CharacterWorkflowState
		err   error
	)
	defer func() {
		logWorkflowOutput(ctx, state, err)
	}()

	if input == nil {
		err = fmt.Errorf("工作流输入不能为空")
		return nil, err
	}
	if input.NovelID == 0 {
		err = fmt.Errorf("小说 ID 不能为空")
		return nil, err
	}
	state, err = graph.Invoke(ctx, input)
	return state, err
}

func newWorkflowState(input *CharacterWorkflowInput) *CharacterWorkflowState {
	return &CharacterWorkflowState{
		CurrentNode:       "init_success",
		NovelID:           input.NovelID,
		CharacterID:       input.CharacterID,
		UserSuggestion:    input.UserSuggestion,
		SavedCharacterIDs: []uint{},
		MaxRetries:        constant.MaxRetries,
	}
}
