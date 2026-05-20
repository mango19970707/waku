package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// routeAfterInit 根据初始化结果决定走快速修改还是完整生成。
func routeAfterInit(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode != "init_success" {
		return compose.END, fmt.Errorf("正文初始化失败: %s", state.ErrorMsg)
	}
	if state.FastPolishMode {
		return "polish", nil
	}
	return "generate", nil
}

// routeAfterGenerate 根据生成节点状态决定进入润色或重试生成。
func routeAfterGenerate(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode == "generate_success" {
		state.RetryCount = 0
		return "polish", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("正文生成失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "generate", nil
}

// routeAfterPolish 根据润色节点状态决定进入审核或重试润色。
func routeAfterPolish(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode == "polish_success" {
		state.RetryCount = 0
		return "review", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("正文润色失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "polish", nil
}

// routeAfterReview 审核通过时直接保存，有建议时进入修改节点。
func routeAfterReview(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode == "review_success" {
		state.RetryCount = 0
		if state.NeedRevise {
			return "revise", nil
		}
		return "saveToDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("正文审核失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "review", nil
}

// routeAfterRevise 根据修改节点状态决定进入保存或重试修改。
func routeAfterRevise(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode == "revise_success" {
		state.RetryCount = 0
		return "saveToDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("正文修改失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "revise", nil
}

// routeAfterSave 单章保存后，如仍有待处理章节则准备下一章，否则结束工作流。
func routeAfterSave(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		state.RetryCount = 0
		if len(state.PendingChapterIDs) > 0 {
			return "prepareNext", nil
		}
		return compose.END, nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("正文保存失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "saveToDB", nil
}

// routeAfterPrepareNext 下一章准备完成后总是进入完整生成流程。
func routeAfterPrepareNext(ctx context.Context, state *ContentWorkflowState) (string, error) {
	if state.CurrentNode == "prepare_next_success" {
		state.RetryCount = 0
		return "generate", nil
	}
	return compose.END, fmt.Errorf("准备下一章失败: %s", state.ErrorMsg)
}
