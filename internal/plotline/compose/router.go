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
		return "polish", nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 提取失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("提取剧情线失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  提取失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "extract", nil
}

// routeAfterPolish 润色节点后的路由逻辑
func routeAfterPolish(ctx context.Context, state *PlotlineExtractWorkflowState) (string, error) {
	if state.CurrentNode == "polish_success" {
		return "saveToDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 润色失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("润色剧情线失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  润色失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "polish", nil
}

// routeAfterSave 保存节点后的路由逻辑
func routeAfterSave(ctx context.Context, state *PlotlineExtractWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}
	if state.CurrentNode == "save_rejected" {
		fmt.Printf("  ❌ 用户拒绝保存操作: %s\n", state.DisapproveReason)
		return compose.END, fmt.Errorf("用户拒绝保存: %s", state.DisapproveReason)
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

// routeAfterPolishRevision 润色节点后的路由逻辑（修改工作流）
func routeAfterPolishRevision(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "polish_success" {
		return "updateDB", nil
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 润色失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("润色剧情线失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  润色失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "polish", nil
}

// routeAfterUpdate 更新节点后的路由逻辑
func routeAfterUpdate(ctx context.Context, state *PlotlineRevisionWorkflowState) (string, error) {
	if state.CurrentNode == "update_success" {
		return compose.END, nil
	}
	if state.CurrentNode == "update_rejected" {
		fmt.Printf("  ❌ 用户拒绝更新操作: %s\n", state.DisapproveReason)
		return compose.END, fmt.Errorf("用户拒绝更新: %s", state.DisapproveReason)
	}
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 更新失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("更新数据库失败: %s", state.ErrorMsg)
	}
	state.RetryCount++
	fmt.Printf("  ⚠️  更新失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "updateDB", nil
}
