package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// routeAfterInit 初始化后的路由决策
func routeAfterInit(ctx context.Context, state *ChapterWorkflowState) (string, error) {
	// 快速新增模式：有小说ID、分卷ID和用户建议，无章节ID（用于新增章节）
	if state.NovelID > 0 && state.VolumeID > 0 && state.UserSuggestion != "" && state.ChapterID == 0 {
		return "polish", nil
	}
	// 快速修改模式：有小说ID、分卷ID、章节ID和用户建议（用于修改已有章节）
	if state.NovelID > 0 && state.VolumeID > 0 && state.ChapterID > 0 && state.UserSuggestion != "" {
		return "polish", nil
	}
	// 完整生成模式：有小说ID和分卷ID，从生成节点开始
	if state.NovelID > 0 && state.VolumeID > 0 {
		return "generate", nil
	}
	return compose.END, fmt.Errorf("输入参数不足，无法确定章节工作流模式")
}

// routeAfterGenerate 生成后的路由决策
func routeAfterGenerate(ctx context.Context, state *ChapterWorkflowState) (string, error) {
	if state.CurrentNode == "generate_success" {
		state.RetryCount = 0
		return "polish", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("章节生成失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "generate", nil
}

// routeAfterPolish 润色后的路由决策
func routeAfterPolish(ctx context.Context, state *ChapterWorkflowState) (string, error) {
	if state.CurrentNode == "polish_success" {
		state.RetryCount = 0
		return "review", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("章节润色失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "polish", nil
}

// routeAfterReview 审核后的路由决策
func routeAfterReview(ctx context.Context, state *ChapterWorkflowState) (string, error) {
	if state.CurrentNode == "review_success" {
		state.RetryCount = 0
		return "revise", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("章节审核失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "review", nil
}

// routeAfterRevise 修改后的路由决策
func routeAfterRevise(ctx context.Context, state *ChapterWorkflowState) (string, error) {
	if state.CurrentNode == "revise_success" {
		state.RetryCount = 0
		return "saveToDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("章节修改失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "revise", nil
}

// routeAfterSave 保存后的路由决策
func routeAfterSave(ctx context.Context, state *ChapterWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("章节保存失败，已达到最大重试次数: %s", state.ErrorMsg)
	}
	state.RetryCount++
	return "saveToDB", nil
}
