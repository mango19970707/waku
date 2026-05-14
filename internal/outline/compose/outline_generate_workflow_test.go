package compose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"waku/internal/outline/repository"

	"waku/internal/constant"
	"waku/internal/model"
	"waku/internal/utils"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func init() {
	db = setupTestDBForTools()
}

// setupTestDBForTools 创建测试数据库
func setupTestDBForTools() *gorm.DB {
	// 使用项目根目录的 data 目录存储测试数据库
	dbPath := "../../../data/waku.db"

	db, _ := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})

	_ = db.AutoMigrate(&model.Novel{})

	return db
}

// ==================== Mock ChatModel ====================

type mockChatModel struct {
	failCount    int // 模拟失败次数
	currentCalls int // 当前调用次数
}

func (m *mockChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...einoModel.Option) (*schema.Message, error) {
	m.currentCalls++

	// 模拟前 failCount 次调用失败
	if m.currentCalls <= m.failCount {
		return nil, fmt.Errorf("模拟失败 %d/%d", m.currentCalls, m.failCount)
	}

	// 获取最后一条消息
	lastMsg := messages[len(messages)-1]
	content := lastMsg.Content

	// 根据内容判断是哪个阶段
	var response string
	if contains(content, "生成小说大纲") || contains(content, "原始大纲") {
		response = "# 生成的大纲\n\n这是一个生成的小说大纲。"
	} else if contains(content, "扩写") || contains(content, "简略大纲") {
		response = "# 扩写的大纲\n\n这是一个扩写后的详细大纲。"
	} else if contains(content, "润色") {
		response = `# 润色的大纲,这是一个润色后的详细大纲。`
	} else if contains(content, "提取") {
		response = `{
  "novel_title": "小说标题",
  "synopsis": "小说简介（200-500字）",
  "outline": "完整的大纲内容",
  "worldview": "世界观设定",
  "cultivation_system": "修行体系"
}`
	} else if contains(content, "调整和优化") {
		response = `# 调整和优化的大纲,这是一个调整和优化后的详细大纲。`
	} else {
		response = "处理完成"
	}

	return schema.AssistantMessage(response, nil), nil
}

func (m *mockChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...einoModel.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}
	r, w := schema.Pipe[*schema.Message](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModel) GetType(ctx context.Context) string {
	return "mock_chat_model"
}

func (m *mockChatModel) IsCallbacksEnabled() bool {
	return false
}

// ==================== 辅助函数 ====================

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用项目根目录的 data 目录存储测试数据库
	dbPath := "../../../data/waku.db"

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	err = db.AutoMigrate(&model.Novel{})
	require.NoError(t, err, "迁移表结构失败")

	return db
}

// cleanupTestFiles 清理测试文件
func cleanupTestFiles(author, novelName string) {
	// 清理作者目录下的所有测试文件
	dir := filepath.Join("data", author)
	os.RemoveAll(dir)
}

// ==================== 测试用例 ====================

// TestOutlineWorkflowGraph_Success 测试完整的成功流程
func TestOutlineWorkflowGcommit raph_Success(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	chatModel := &mockChatModel{failCount: 0}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程 ===")

	// 准备输入
	input := &OutlineWorkflowInput{
		NovelName:       "测试小说",
		Author:          "测试作者",
		Genre:           "玄幻",
		OriginalOutline: "一个少年的修仙之路",
	}

	// 清理测试文件
	defer cleanupTestFiles(input.Author, input.NovelName)

	// 执行工作流
	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	// 验证结果
	if result.CurrentNode != "save_success" {
		t.Errorf("期望 CurrentNode=save_success，实际: %s", result.CurrentNode)
	}

	if result.GeneratedOutline == "" {
		t.Error("生成的大纲为空")
	}

	if result.ExpandedOutline == "" {
		t.Error("扩写的大纲为空")
	}

	if result.PolishedOutline == "" {
		t.Error("润色的大纲为空")
	}

	if result.NovelID == 0 {
		t.Error("小说ID为空")
	}

	t.Log("✓ 完整流程测试通过")
}

// TestOutlineWorkflowGraph_RetryOnFailure 测试失败重试机制
func TestOutlineWorkflowGraph_RetryOnFailure(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel（前2次调用失败）
	chatModel := &mockChatModel{failCount: 2}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：失败重试机制（前2次失败，第3次成功） ===")

	// 准备输入
	input := &OutlineWorkflowInput{
		NovelName:       "重试测试小说",
		Author:          "测试作者",
		Genre:           "都市",
		OriginalOutline: "一个程序员的逆袭之路",
	}

	// 清理测试文件
	defer cleanupTestFiles(input.Author, input.NovelName)

	// 执行工作流
	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	// 验证结果
	if result.CurrentNode != "save_success" {
		t.Errorf("期望 CurrentNode=save_success，实际: %s", result.CurrentNode)
	}

	t.Logf("✓ 重试机制测试通过，总调用次数: %d", chatModel.currentCalls)
}

// TestOutlineWorkflowGraph_MaxRetriesExceeded 测试超过最大重试次数
func TestOutlineWorkflowGraph_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel（一直失败）
	chatModel := &mockChatModel{failCount: 100}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：超过最大重试次数 ===")

	// 准备输入
	input := &OutlineWorkflowInput{
		NovelName:       "失败测试小说",
		Author:          "测试作者",
		Genre:           "科幻",
		OriginalOutline: "星际探险故事",
	}

	// 清理测试文件
	defer cleanupTestFiles(input.Author, input.NovelName)

	// 执行工作流（应该失败）
	_, err = ExecuteOutlineWorkflow(ctx, graph, input)
	if err == nil {
		t.Fatal("期望执行失败，但成功了")
	}

	t.Logf("✓ 如预期失败: %v", err)
}

// TestOutlineWorkflowGraph_ResumeFromCheckpoint 测试断点续传
func TestOutlineWorkflowGraph_ResumeFromCheckpoint(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	chatModel := &mockChatModel{failCount: 0}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：断点续传 ===")

	// 准备输入
	input := &OutlineWorkflowInput{
		NovelName:       "断点续传测试",
		Author:          "测试作者",
		Genre:           "武侠",
		OriginalOutline: "江湖恩怨录",
	}

	// 清理测试文件
	defer cleanupTestFiles(input.Author, input.NovelName)

	// 第一次执行：完整流程
	t.Log("\n--- 第一次执行：完整流程 ---")
	result1, err := ExecuteOutlineWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("第一次执行失败: %v", err)
	}

	firstCallCount := chatModel.currentCalls
	t.Logf("第一次执行完成，调用次数: %d", firstCallCount)

	// 重置调用计数
	chatModel.currentCalls = 0

	// 第二次执行：应该从文件加载，不调用模型
	t.Log("\n--- 第二次执行：断点续传 ---")
	result2, err := ExecuteOutlineWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("第二次执行失败: %v", err)
	}

	secondCallCount := chatModel.currentCalls
	t.Logf("第二次执行完成，调用次数: %d", secondCallCount)

	// 验证结果一致
	if result1.GeneratedOutline != result2.GeneratedOutline {
		t.Error("两次执行的生成大纲不一致")
	}

	if result1.ExpandedOutline != result2.ExpandedOutline {
		t.Error("两次执行的扩写大纲不一致")
	}

	if result1.PolishedOutline != result2.PolishedOutline {
		t.Error("两次执行的润色大纲不一致")
	}

	// 验证第二次执行的调用次数应该减少（只有提取阶段会调用，因为提取没有文件缓存）
	if secondCallCount >= firstCallCount {
		t.Errorf("期望第二次执行调用次数少于第一次，第一次: %d，第二次: %d", firstCallCount, secondCallCount)
	}

	t.Log("✓ 断点续传测试通过")
}

// TestOutlineWorkflowGraph_PartialResume 测试部分断点续传
func TestOutlineWorkflowGraph_PartialResume(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	chatModel := &mockChatModel{failCount: 0}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	t.Log("=== 测试场景：部分断点续传（只有生成阶段的文件） ===")

	// 准备输入
	input := &OutlineWorkflowInput{
		NovelName:       "部分断点测试",
		Author:          "测试作者",
		Genre:           "奇幻",
		OriginalOutline: "魔法世界的冒险",
	}

	// 清理测试文件
	defer cleanupTestFiles(input.Author, input.NovelName)

	// 预先创建"生成"阶段的文件
	generatedContent := "# 预先生成的大纲\n\n这是预先保存的生成大纲。"
	// 注意：这里使用 NovelID=1 来模拟已有小说的情况
	if err := utils.SaveToFile(input.Author, constant.FileTypeOutline, constant.StageGenerate, generatedContent); err != nil {
		t.Fatalf("预先保存文件失败: %v", err)
	}

	// 构建 Graph
	graph, err := BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	// 执行工作流
	result, err := ExecuteOutlineWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	// 验证结果
	if result.GeneratedOutline != generatedContent {
		t.Error("生成大纲应该从文件加载")
	}

	if result.ExpandedOutline == "" {
		t.Error("扩写大纲不应该为空")
	}

	if result.PolishedOutline == "" {
		t.Error("润色大纲不应该为空")
	}

	t.Logf("✓ 部分断点续传测试通过，模型调用次数: %d", chatModel.currentCalls)
}

// TestOutlineWorkflowGraph_InvalidInput 测试无效输入
func TestOutlineWorkflowGraph_InvalidInput(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	chatModel := &mockChatModel{failCount: 0}

	// 创建测试数据库
	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：无效输入 ===")

	testCases := []struct {
		name  string
		input *OutlineWorkflowInput
	}{
		{
			name: "作者名称为空",
			input: &OutlineWorkflowInput{
				NovelName:       "测试小说",
				Author:          "",
				Genre:           "玄幻",
				OriginalOutline: "测试大纲",
			},
		},
		{
			name: "小说名称为空",
			input: &OutlineWorkflowInput{
				NovelName:       "",
				Author:          "测试作者",
				Genre:           "玄幻",
				OriginalOutline: "测试大纲",
			},
		},
		{
			name: "小说类型为空",
			input: &OutlineWorkflowInput{
				NovelName:       "测试小说",
				Author:          "测试作者",
				Genre:           "",
				OriginalOutline: "测试大纲",
			},
		},
		{
			name: "原始大纲为空",
			input: &OutlineWorkflowInput{
				NovelName:       "测试小说",
				Author:          "测试作者",
				Genre:           "玄幻",
				OriginalOutline: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ExecuteOutlineWorkflow(ctx, graph, tc.input)
			if err == nil {
				t.Error("期望执行失败，但成功了")
			}
			t.Logf("✓ 如预期失败: %v", err)
		})
	}
}
