package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// routeAfterInit 在初始化后决定工作流模式。
func routeAfterInit(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.NovelID > 0 && state.UserSuggestion != "" {
		return "polish", nil
	}
	if state.OriginalOutline != "" {
		return "generate", nil
	}
	return compose.END, fmt.Errorf("输入参数不足，无法确定工作流模式")
}

// routeAfterGenerate 处理生成节点后的路由。
func routeAfterGenerate(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	return routeWithRetry(state, "generate_success", "expand", "generate")
}

// routeAfterExpand 处理扩写节点后的路由。
func routeAfterExpand(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	return routeWithRetry(state, "expand_success", "polish", "expand")
}

// routeAfterPolish 处理润色节点后的路由。
func routeAfterPolish(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	return routeWithRetry(state, "polish_success", "review", "polish")
}

// routeAfterReview 处理审核节点后的路由。
func routeAfterReview(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	return routeWithRetry(state, "review_success", "revise", "review")
}

// routeAfterRevise 处理修改节点后的路由。
func routeAfterRevise(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	return routeWithRetry(state, "revise_success", "saveToDB", "revise")
}

// routeAfterSave 处理保存节点后的路由。
func routeAfterSave(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	return routeWithRetry(state, "save_success", compose.END, "saveToDB")
}

func routeWithRetry(state *OutlineWorkflowState, successNode, nextNode, retryNode string) (string, error) {
	if state.CurrentNode == successNode {
		return nextNode, nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("%s 失败: %s", retryNode, state.ErrorMsg)
	}

	state.RetryCount++
	return retryNode, nil
}
