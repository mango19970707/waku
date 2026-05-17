package compose

import (
	"context"
	"fmt"
	"waku/internal/character/repository"
	"waku/internal/constant"

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
	UserSuggestion      string
	ExistingCharacters  map[string]any
	ExtractedCharacters map[string]any
	PolishedCharacters  map[string]any
	ReviewedCharacters  map[string]any
	RevisedCharacters   map[string]any
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
) (compose.Runnable[*CharacterWorkflowInput, *CharacterWorkflowState], error) {
	if err := ensureAgentExecutor(); err != nil {
		return nil, err
	}

	graph := compose.NewGraph[*CharacterWorkflowInput, *CharacterWorkflowState]()
	graph.AddLambdaNode("init", compose.InvokableLambda(initNode(novelRepo, characterRepo)))
	graph.AddLambdaNode("extract", compose.InvokableLambda(extractNode(ctx, chatModel)))
	graph.AddLambdaNode("polish", compose.InvokableLambda(polishNode(ctx, chatModel)))
	graph.AddLambdaNode("review", compose.InvokableLambda(reviewNode(ctx, chatModel)))
	graph.AddLambdaNode("revise", compose.InvokableLambda(reviseNode(ctx, chatModel)))
	graph.AddLambdaNode("saveToDB", compose.InvokableLambda(saveToDBNode(characterRepo)))

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
		"revise":   true,
		"saveToDB": true,
	}))
	graph.AddBranch("saveToDB", compose.NewGraphBranch(routeAfterSave, map[string]bool{
		"saveToDB":  true,
		compose.END: true,
	}))

	return graph.Compile(ctx, compose.WithMaxRunSteps(50))
}

func NewCharacterWorkflow(
	ctx context.Context,
	chatModel einoModel.BaseChatModel,
	novelRepo repository.Repository,
	characterRepo repository.CharacterRepository,
) (compose.Runnable[*CharacterWorkflowInput, *CharacterWorkflowState], error) {
	return BuildCharacterWorkflowGraph(ctx, chatModel, novelRepo, characterRepo)
}

func ExecuteCharacterWorkflow(
	ctx context.Context,
	graph compose.Runnable[*CharacterWorkflowInput, *CharacterWorkflowState],
	input *CharacterWorkflowInput,
) (*CharacterWorkflowState, error) {
	if input == nil {
		return nil, fmt.Errorf("工作流输入不能为空")
	}
	if input.NovelID == 0 {
		return nil, fmt.Errorf("小说 ID 不能为空")
	}
	return graph.Invoke(ctx, input)
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
