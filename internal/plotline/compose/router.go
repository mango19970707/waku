package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// ==================== 提取工作流路由 ====================

// routeAfterExtract 提取节点后的路由逻辑
func routeAfterExtract(ctx context.Context, state *PlotlineExtractWorkflowState) (string, error) {
	if state.CurrentNode == "extract_success" {
		return "saveToDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 提取失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("提取剧情线失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  提取失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "extract", nil
}

// routeAfterSave 保存节点后的路由逻辑
func routeAfterSave(ctx context.Context, state *PlotlineExtractWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 保存失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("保存到数据库失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  保存失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "saveToDB", nil
}

// ==================== 修改工作流路由 ====================

// routeAfterRevise 优化节点后的路由逻辑（修改工作流）
func routeAfterRevise(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "revise_success" {
		// 如果是单个剧情线修改，进入大纲冲突检测
		if state.TargetPlotlineID != "" {
			return "detectOutlineConflict", nil
		}
		// 否则直接保存
		return "saveOrUpdateDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 优化失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("优化剧情线失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  优化失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "revise", nil
}

// routeAfterSaveOrUpdate 保存或更新节点后的路由逻辑
func routeAfterSaveOrUpdate(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 保存失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("保存到数据库失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  保存失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "saveOrUpdateDB", nil
}

// ==================== 冲突检测和解决路由 ====================

// routeAfterReviseV2 优化节点后的路由逻辑（支持冲突检测）
func routeAfterReviseV2(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "revise_success" {
		// 如果是单个剧情线修改，进入冲突检测
		if state.TargetPlotlineID != "" {
			return "detectConflict", nil
		}
		// 否则直接保存
		return "saveOrUpdateDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 优化失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("优化剧情线失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  优化失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "revise", nil
}

// routeAfterDetectConflict 冲突检测节点后的路由逻辑
func routeAfterDetectConflict(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "conflict_detection_failed" {
		if state.RetryCount >= state.MaxRetries {
			fmt.Printf("  ❌ 冲突检测失败，已达到最大重试次数 %d\n", state.MaxRetries)
			return compose.END, fmt.Errorf("冲突检测失败: %s", state.ErrorMsg)
		}
		state.RetryCount++
		fmt.Printf("  ⚠️  冲突检测失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
		return "detectConflict", nil
	}

	if state.CurrentNode == "no_conflict" || state.CurrentNode == "conflict_detection_skipped" {
		return "saveOrUpdateDB", nil
	}

	if state.CurrentNode == "conflict_detected" {
		return "reviseOutline", nil
	}

	return compose.END, fmt.Errorf("未知状态: %s", state.CurrentNode)
}

// routeAfterConflictDetection 冲突检测节点后的路由逻辑（别名，保持兼容）
func routeAfterConflictDetection(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	return routeAfterDetectConflict(ctx, state)
}

// routeAfterReviseOutline 修改大纲节点后的路由逻辑
func routeAfterReviseOutline(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "revise_outline_failed" {
		if state.RetryCount >= state.MaxRetries {
			fmt.Printf("  ❌ 修改大纲失败，已达到最大重试次数 %d\n", state.MaxRetries)
			return compose.END, fmt.Errorf("修改大纲失败: %s", state.ErrorMsg)
		}
		state.RetryCount++
		fmt.Printf("  ⚠️  修改大纲失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
		return "reviseOutline", nil
	}

	if state.CurrentNode == "revise_outline_success" {
		// 大纲冲突解决后，继续检测剧情线冲突
		return "detectPlotlinesConflict", nil
	}

	return compose.END, fmt.Errorf("未知状态: %s", state.CurrentNode)
}

// routeAfterResolveOutlineConflict 解决大纲冲突节点后的路由逻辑（别名，保持兼容）
func routeAfterResolveOutlineConflict(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	return routeAfterReviseOutline(ctx, state)
}

// routeAfterReviseOtherPlotlines 修改其他剧情线节点后的路由逻辑
func routeAfterReviseOtherPlotlines(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "revise_other_plotlines_failed" {
		if state.RetryCount >= state.MaxRetries {
			fmt.Printf("  ❌ 修改其他剧情线失败，已达到最大重试次数 %d\n", state.MaxRetries)
			return compose.END, fmt.Errorf("修改其他剧情线失败: %s", state.ErrorMsg)
		}
		state.RetryCount++
		fmt.Printf("  ⚠️  修改其他剧情线失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
		return "reviseOtherPlotlines", nil
	}

	if state.CurrentNode == "revise_other_plotlines_success" {
		return "saveOrUpdateDB", nil
	}

	return compose.END, fmt.Errorf("未知状态: %s", state.CurrentNode)
}

// routeAfterResolvePlotlinesConflict 解决剧情线冲突节点后的路由逻辑（别名，保持兼容）
func routeAfterResolvePlotlinesConflict(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	return routeAfterReviseOtherPlotlines(ctx, state)
}

// ==================== 拆分的冲突检测路由 ====================

// routeAfterDetectOutlineConflict 大纲冲突检测节点后的路由逻辑
func routeAfterDetectOutlineConflict(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "outline_conflict_detection_failed" {
		if state.RetryCount >= state.MaxRetries {
			fmt.Printf("  ❌ 大纲冲突检测失败，已达到最大重试次数 %d\n", state.MaxRetries)
			return compose.END, fmt.Errorf("大纲冲突检测失败: %s", state.ErrorMsg)
		}
		state.RetryCount++
		fmt.Printf("  ⚠️  大纲冲突检测失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
		return "detectOutlineConflict", nil
	}

	if state.CurrentNode == "outline_conflict_detection_skipped" || state.CurrentNode == "no_outline_conflict" {
		// 没有大纲冲突，继续检测剧情线冲突
		return "detectPlotlinesConflict", nil
	}

	if state.CurrentNode == "outline_conflict_detected" {
		// 有大纲冲突，先解决大纲冲突
		return "reviseOutline", nil
	}

	return compose.END, fmt.Errorf("未知状态: %s", state.CurrentNode)
}

// routeAfterDetectPlotlinesConflict 剧情线冲突检测节点后的路由逻辑
func routeAfterDetectPlotlinesConflict(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "plotlines_conflict_detection_failed" {
		if state.RetryCount >= state.MaxRetries {
			fmt.Printf("  ❌ 剧情线冲突检测失败，已达到最大重试次数 %d\n", state.MaxRetries)
			return compose.END, fmt.Errorf("剧情线冲突检测失败: %s", state.ErrorMsg)
		}
		state.RetryCount++
		fmt.Printf("  ⚠️  剧情线冲突检测失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
		return "detectPlotlinesConflict", nil
	}

	if state.CurrentNode == "plotlines_conflict_detection_skipped" || state.CurrentNode == "no_other_plotlines" || state.CurrentNode == "no_plotlines_conflict" {
		// 没有剧情线冲突，直接保存
		return "saveOrUpdateDB", nil
	}

	if state.CurrentNode == "plotlines_conflict_detected" {
		// 有剧情线冲突，解决剧情线冲突
		return "reviseOtherPlotlines", nil
	}

	return compose.END, fmt.Errorf("未知状态: %s", state.CurrentNode)
}
