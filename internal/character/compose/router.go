package compose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

// ==================== 人物提取工作流路由 ====================

// routeAfterExtract 提取节点后的路由逻辑
//
// 参数：
//   - ctx: 上下文对象
//   - state: 工作流状态
//
// 返回：
//   - string: 下一个节点名称
//   - error: 错误信息
func routeAfterExtract(ctx context.Context, state *CharacterExtractWorkflowState) (string, error) {
	// 提取成功，进入保存节点
	if state.CurrentNode == "extract_success" {
		return "saveToDB", nil
	}

	// 达到最大重试次数，结束工作流
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("人物提取失败，已重试 %d 次: %s", state.MaxRetries, state.ErrorMsg)
	}

	// 重试提取节点
	state.RetryCount++
	return "extract", nil
}

// routeAfterSave 保存节点后的路由逻辑
//
// 参数：
//   - ctx: 上下文对象
//   - state: 工作流状态
//
// 返回：
//   - string: 下一个节点名称
//   - error: 错误信息
func routeAfterSave(ctx context.Context, state *CharacterExtractWorkflowState) (string, error) {
	// 保存成功，结束工作流
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}

	// 达到最大重试次数，结束工作流
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("保存人物失败，已重试 %d 次: %s", state.MaxRetries, state.ErrorMsg)
	}

	// 重试保存节点
	state.RetryCount++
	return "saveToDB", nil
}

// ==================== 人物修改工作流路由 ====================

// routeAfterReview 审核节点后的路由逻辑
//
// 参数：
//   - ctx: 上下文对象
//   - state: 工作流状态
//
// 返回：
//   - string: 下一个节点名称
//   - error: 错误信息
func routeAfterReview(ctx context.Context, state *CharacterRevisionWorkflowState) (string, error) {
	// 审核成功，进入优化节点
	if state.CurrentNode == "review_success" {
		return "revise", nil
	}

	// 达到最大重试次数，结束工作流
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("人物审核失败，已重试 %d 次: %s", state.MaxRetries, state.ErrorMsg)
	}

	// 重试审核节点
	state.RetryCount++
	return "review", nil
}

// routeAfterRevise 优化节点后的路由逻辑
//
// 参数：
//   - ctx: 上下文对象
//   - state: 工作流状态
//
// 返回：
//   - string: 下一个节点名称
//   - error: 错误信息
func routeAfterRevise(ctx context.Context, state *CharacterRevisionWorkflowState) (string, error) {
	// 优化成功，进入保存/更新节点
	if state.CurrentNode == "revise_success" {
		return "saveOrUpdateDB", nil
	}

	// 达到最大重试次数，结束工作流
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("人物优化失败，已重试 %d 次: %s", state.MaxRetries, state.ErrorMsg)
	}

	// 重试优化节点
	state.RetryCount++
	return "revise", nil
}

// routeAfterSaveOrUpdate 保存/更新节点后的路由逻辑
//
// 参数：
//   - ctx: 上下文对象
//   - state: 工作流状态
//
// 返回：
//   - string: 下一个节点名称
//   - error: 错误信息
func routeAfterSaveOrUpdate(ctx context.Context, state *CharacterRevisionWorkflowState) (string, error) {
	// 保存/更新成功，结束工作流
	if state.CurrentNode == "save_success" {
		return compose.END, nil
	}

	// 达到最大重试次数，结束工作流
	if state.RetryCount >= state.MaxRetries {
		return compose.END, fmt.Errorf("保存/更新人物失败，已重试 %d 次: %s", state.MaxRetries, state.ErrorMsg)
	}

	// 重试保存/更新节点
	state.RetryCount++
	return "saveOrUpdateDB", nil
}
