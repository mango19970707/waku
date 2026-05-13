package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// routeAfterGenerate 生成后的路由
func routeAfterGenerate(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "generate_success" {
		return "expand", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 生成失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("生成失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  生成失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "generate", nil
}

// routeAfterExpand 扩写后的路由
func routeAfterExpand(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "expand_success" {
		return "polish", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 扩写失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("扩写失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  扩写失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "expand", nil
}

// routeAfterPolish 润色后的路由
func routeAfterPolish(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "polish_success" {
		return "extract", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 润色失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("润色失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  润色失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "polish", nil
}

// routeAfterExtract 提取后的路由
func routeAfterExtract(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "extract_success" {
		return "saveToDB", nil
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 提取失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("提取失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  提取失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "extract", nil
}

// routeAfterSave 保存后的路由
func routeAfterSave(ctx context.Context, state *OutlineWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}

	// 如果用户拒绝保存，直接结束流程
	if state.CurrentNode == "save_rejected" {
		fmt.Printf("  ❌ 用户拒绝保存操作: %s\n", state.ErrorMsg)
		return compose.END, fmt.Errorf("用户拒绝保存: %s", state.ErrorMsg)
	}

	// 检查是否超过最大重试次数
	if state.RetryCount >= state.MaxRetries {
		fmt.Printf("  ❌ 保存失败，已达到最大重试次数 %d\n", state.MaxRetries)
		return compose.END, fmt.Errorf("保存失败: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  保存失败，重试 %d/%d: %s\n", state.RetryCount, state.MaxRetries, state.ErrorMsg)
	return "saveToDB", nil
}
