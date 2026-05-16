package outline

import (
	"context"
	"log/slog"
	"testing"

	"waku/internal/config"
	"waku/internal/model"
	"waku/internal/plotline/compose"
	plotlineRepo "waku/internal/plotline/repository"
	"waku/internal/utils"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	_ "modernc.org/sqlite" // 使用纯 Go 的 SQLite 驱动（不需要 CGO）
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

func TestPlotlineGenerateWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建测试数据库
	db := setupTestDBForTools(t)

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
		db.Delete(testNovel)
	}()

	// 创建 repository
	novelRepo := plotlineRepo.NewRepository(db)
	plotlineRepository := plotlineRepo.NewPlotlineRepository(db)

	// 构建 Graph
	graph, err := compose.NewPlotlineExtractWorkflow(ctx, chatModel, novelRepo, plotlineRepository)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程 ===")

	// 准备输入（只需要小说ID）
	input := &compose.PlotlineExtractWorkflowInput{
		NovelID: testNovel.ID,
	}

	// 执行工作流
	_, err = compose.ExecutePlotlineExtractWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	t.Log("✓ 完整流程测试通过")
}

func TestPlotlineRevisionWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建测试数据库
	db := setupTestDBForTools(t)

	// 创建 repository
	novelRepo := plotlineRepo.NewRepository(db)
	plotlineRepository := plotlineRepo.NewPlotlineRepository(db)

	// 构建 Graph
	graph, err := compose.NewPlotlineRevisionWorkflow(ctx, chatModel, novelRepo, plotlineRepository)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程 ===")

	// 准备输入
	input := &compose.PlotlineRevisionWorkflowInput{
		NovelID:        2,
		UserSuggestion: "完全模仿斗破苍穹的套路，修行体系啥都一样",
	}

	// 执行工作流
	_, err = compose.ExecutePlotlineRevisionWorkflow(ctx, graph, input)
	if err != nil {
		t.Fatalf("执行工作流失败: %v", err)
	}

	t.Log("✓ 完整流程测试通过")
}

// setupTestDBForTools 创建测试数据库
func setupTestDBForTools(t *testing.T) *gorm.DB {
	// 使用项目根目录的 data 目录存储测试数据库
	dbPath := "../../data/waku.db"

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
	}, &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	err = db.AutoMigrate(&model.Novel{})
	require.NoError(t, err, "迁移表结构失败")

	return db
}
