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
