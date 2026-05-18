package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

func routeAfterInit(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	if state.NovelID > 0 && state.UserSuggestion != "" {
		return "polish", nil
	}
	if state.NovelID > 0 {
		return "extract", nil
	}
	return compose.END, fmt.Errorf("输入参数不足，无法确定角色工作流模式")
}

func routeAfterExtract(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	return routeWithRetry(state, "extract_success", "polish", "extract")
}

func routeAfterPolish(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	return routeWithRetry(state, "polish_success", "review", "polish")
}

func routeAfterReview(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	return routeWithRetry(state, "review_success", "revise", "review")
}

func routeAfterRevise(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	return routeWithRetry(state, "revise_success", "preSave", "revise")
}

func routeAfterPreSave(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	if state.CurrentNode == "preSave_success" {
		return "mapIDs", nil
	}
	if state.CurrentNode == "save_success" {
		// 全新生成场景，已经完成所有保存工作
		return compose.END, nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("preSave 失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "preSave", nil
}

func routeAfterMapIDs(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	return routeWithRetry(state, "mapIDs_success", "updateRelationships", "mapIDs")
}

func routeAfterUpdateRelationships(ctx context.Context, state *CharacterWorkflowState) (string, error) {
	return routeWithRetry(state, "updateRelationships_success", compose.END, "updateRelationships")
}

func routeWithRetry(state *CharacterWorkflowState, successNode, nextNode, retryNode string) (string, error) {
	if state.CurrentNode == successNode {
		return nextNode, nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("%s 失败: %s", retryNode, state.ErrorMsg)
	}
	state.RetryCount++
	return retryNode, nil
}
