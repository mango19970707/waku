package compose

import (
	"context"
	"testing"
	"waku/internal/outline/repository"
)

// TestOutlineRevisionWorkflow 测试完整的成功流程
func TestOutlineRevisionWorkflow(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	chatModel := &mockChatModel{failCount: 0}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := BuildOutlineRevisionWorkflowGraph(ctx, chatModel, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程 ===")

	// 准备输入
	input := &OutlineRevisionWorkflowInput{
		NovelID:        2,
		UserSuggestion: "一个少年的修仙之路",
	}

	// 执行工作流
	result, err := ExecuteOutlineRevisionWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	// 验证结果
	if result.CurrentNode != "update_success" {
		t.Errorf("期望 CurrentNode=update_success，实际: %s", result.CurrentNode)
	}

	if result.NovelID == 0 {
		t.Error("小说ID为空")
	}

	t.Log("✓ 完整流程测试通过")
}
