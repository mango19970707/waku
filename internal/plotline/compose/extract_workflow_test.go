package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"waku/internal/model"
	"waku/internal/plotline/agents"
	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	_ "modernc.org/sqlite" // 使用纯 Go 的 SQLite 驱动（不需要 CGO）
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func init() {
	db = setupTestDBForTools()
}

// ==================== 测试用例 ====================

// TestPlotlineExtractWorkflowGraph 测试完整的成功流程（多条剧情线）
func TestPlotlineExtractWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel（返回 JSON 格式的多条剧情线）
	chatModel := &mockChatModel{failCount: 0}

	// 先创建一个测试小说（包含大纲）
	testNovel := &model.Novel{
		Title:   "测试小说",
		Author:  "测试作者",
		Genre:   "玄幻",
		Outline: "完全模仿斗破苍穹的套路，修行体系啥都一样。主角从一个废柴开始，逐步修炼成为强者。",
		Status:  "draft",
	}
	err := db.Create(testNovel).Error
	require.NoError(t, err, "创建测试小说失败")

	// 清理测试数据
	defer func() {
		// 清理剧情线
		db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Delete(&model.Plotline{})
		// 清理小说
		db.Delete(testNovel)
		cleanupTestFiles(testNovel.Author, testNovel.Title)
	}()

	// 创建 repository
	novelRepo := repository.NewRepository(db)
	plotlineRepo := repository.NewPlotlineRepository(db)

	// 构建 Graph
	graph, err := NewPlotlineExtractWorkflow(ctx, chatModel, novelRepo, plotlineRepo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程（多条剧情线） ===")

	// 准备输入（只需要小说ID）
	input := &PlotlineExtractWorkflowInput{
		NovelID: testNovel.ID,
	}

	// 执行工作流
	result, err := ExecutePlotlineExtractWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	// 验证结果
	if result.CurrentNode != "save_success" {
		t.Errorf("期望 CurrentNode=save_success，实际: %s", result.CurrentNode)
	}

	if result.NovelID == 0 {
		t.Error("小说ID为空")
	}

	// 验证保存的剧情线数量
	if len(result.SavedPlotlineIDs) == 0 {
		t.Error("未保存任何剧情线")
	}

	t.Logf("✓ 成功保存 %d 条剧情线", len(result.SavedPlotlineIDs))
	t.Logf("✓ 剧情线ID列表: %v", result.SavedPlotlineIDs)

	// 验证数据库中的剧情线
	var plotlines []model.Plotline
	err = db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Find(&plotlines).Error
	require.NoError(t, err, "查询剧情线失败")

	if len(plotlines) != len(result.SavedPlotlineIDs) {
		t.Errorf("数据库中的剧情线数量 (%d) 与返回的ID数量 (%d) 不一致", len(plotlines), len(result.SavedPlotlineIDs))
	}

	// 验证剧情线内容
	for i, plotline := range plotlines {
		t.Logf("剧情线 %d: %s (%s)", i+1, plotline.Title, plotline.Type)
		if plotline.Title == "" {
			t.Errorf("剧情线 %d 的标题为空", i+1)
		}
		if plotline.Type != "main" && plotline.Type != "sub" {
			t.Errorf("剧情线 %d 的类型无效: %s", i+1, plotline.Type)
		}
		if plotline.Description == "" {
			t.Errorf("剧情线 %d 的描述为空", i+1)
		}
	}

	t.Log("✓ 完整流程测试通过")
}

// setupTestDBForTools 创建测试数据库
func setupTestDBForTools() *gorm.DB {
	// 使用项目根目录的 data 目录存储测试数据库
	dbPath := "../../../data/waku.db"

	db, _ := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
	}, &gorm.Config{})

	_ = db.AutoMigrate(&model.Novel{})
	_ = db.AutoMigrate(&model.Plotline{})

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
	if contains(content, "提取") || contains(content, "JSON") {
		// 返回 JSON 格式的多条剧情线
		output := agents.PlotlineExtractorOutput{
			Plotlines: []agents.PlotlineItem{
				{
					Title:        "主角修仙成神之路",
					Type:         "main",
					Description:  "主角从一个废柴少年开始，因意外获得神秘传承，踏上修仙之路。经历筑基、金丹、元婴等境界突破，与各大势力斗智斗勇，最终成为一代仙帝。",
					StartChapter: 1,
					EndChapter:   0,
				},
				{
					Title:        "与师姐的情感纠葛",
					Type:         "sub",
					Description:  "主角在宗门中与美丽的师姐相识相知，两人在历练中互生情愫。但师姐身负家族使命，主角为了师姐的幸福，暗中帮助她解决家族危机。",
					StartChapter: 10,
					EndChapter:   80,
				},
				{
					Title:        "复仇灭杀仇家",
					Type:         "sub",
					Description:  "主角家族被仇家灭门，只有主角侥幸逃脱。主角隐忍修炼，实力大增后开始复仇计划，最终灭杀仇家满门。",
					StartChapter: 5,
					EndChapter:   60,
				},
			},
		}
		jsonBytes, _ := json.Marshal(output)
		response = string(jsonBytes)
	} else if contains(content, "优化") {
		response = `# 调整和优化的剧情线,这是一个调整和优化后的详细剧情线。`
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

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
	}, &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	err = db.AutoMigrate(&model.Novel{})
	require.NoError(t, err, "迁移表结构失败")

	err = db.AutoMigrate(&model.Plotline{})
	require.NoError(t, err, "迁移 Plotline 表结构失败")

	return db
}

// cleanupTestFiles 清理测试文件
func cleanupTestFiles(author, novelName string) {
	// 清理作者目录下的所有测试文件
	dir := filepath.Join("data", author)
	os.RemoveAll(dir)
}
