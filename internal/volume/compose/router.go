package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// routeAfterPartition 分卷后的路由逻辑
func routeAfterPartition(ctx context.Context, state *VolumePartitionWorkflowState) (string, error) {
	if state.CurrentNode == "partition_success" {
		return "extract", nil // 成功，进入提取节点
	}

	// 失败，检查是否需要重试
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("分卷失败，已达到最大重试次数: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  分卷失败，重试 %d/%d\n", state.RetryCount, state.MaxRetries)
	return "partition", nil
}

// routeAfterExtract 提取后的路由逻辑
func routeAfterExtract(ctx context.Context, state *VolumePartitionWorkflowState) (string, error) {
	if state.CurrentNode == "extract_success" {
		return "saveVolumes", nil // 成功，进入保存节点
	}

	// 失败，检查是否需要重试
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("提取卷信息失败，已达到最大重试次数: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  提取失败，重试 %d/%d\n", state.RetryCount, state.MaxRetries)
	return "extract", nil
}

// routeAfterSaveVolumes 保存卷后的路由逻辑
func routeAfterSaveVolumes(ctx context.Context, state *VolumePartitionWorkflowState) (string, error) {
	if state.CurrentNode == "save_success" {
		return "partitionChapters", nil // 成功，进入分章节点
	}

	if state.CurrentNode == "save_rejected" {
		return compose.END, fmt.Errorf("用户拒绝保存: %s", state.DisapproveReason)
	}

	// 失败，检查是否需要重试
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("保存卷失败，已达到最大重试次数: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  保存失败，重试 %d/%d\n", state.RetryCount, state.MaxRetries)
	return "saveVolumes", nil
}

// routeAfterPartitionChapters 分章后的路由逻辑
func routeAfterPartitionChapters(ctx context.Context, state *VolumePartitionWorkflowState) (string, error) {
	if state.CurrentNode == "partition_chapters_success" {
		return compose.END, nil // 成功，结束工作流
	}

	// 失败，检查是否需要重试
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("分章失败，已达到最大重试次数: %s", state.ErrorMsg)
	}

	// 重试
	state.RetryCount++
	fmt.Printf("  ⚠️  分章失败，重试 %d/%d\n", state.RetryCount, state.MaxRetries)
	return "partitionChapters", nil
}
