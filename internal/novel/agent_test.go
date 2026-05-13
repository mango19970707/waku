package novel

import (
	"context"
	"log/slog"
	"testing"

	"waku/internal/config"
	"waku/internal/model"
	"waku/internal/novel/compose"
	"waku/internal/novel/repository"
	"waku/internal/utils"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	chatModel einoModel.BaseChatModel
)

func init() {
	cfg, err := config.Load("../../config/config.yaml")
	if err != nil {
		slog.Error("加载配置失败", "err", err)
	}

	chatModel, err = utils.NewChatModel(context.Background(), &cfg.Model)
	if err != nil {
		slog.Error("创建chat model失败", "err", err)
	}
}

func TestOutlineGenerateWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建测试数据库
	db := setupTestDBForTools(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := compose.BuildOutlineWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程 ===")

	// 准备输入
	input := &compose.OutlineWorkflowInput{
		NovelName:       "测试小说",
		Genre:           "玄幻",
		OriginalOutline: "一个少年的修仙之路",
	}

	// 执行工作流
	_, err = compose.ExecuteOutlineWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	t.Log("✓ 完整流程测试通过")
}

func TestOutlineRevisionWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建测试数据库
	db := setupTestDBForTools(t)
	repo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := compose.BuildOutlineRevisionWorkflowGraph(ctx, chatModel, db, repo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程 ===")

	// 准备输入
	input := &compose.OutlineRevisionWorkflowInput{
		NovelID:        1,
		UserSuggestion: "修行体系改为和《斗破苍穹》一样",
	}

	// 执行工作流
	_, err = compose.ExecuteOutlineRevisionWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	t.Log("✓ 完整流程测试通过")
}

// setupTestDBForTools 创建测试数据库
func setupTestDBForTools(t *testing.T) *gorm.DB {
	// 使用项目根目录的 data 目录存储测试数据库
	dbPath := "../../data/waku.db"

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	err = db.AutoMigrate(&model.Novel{})
	require.NoError(t, err, "迁移表结构失败")

	return db
}
