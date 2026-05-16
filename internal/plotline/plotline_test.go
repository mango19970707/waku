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
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // 使用纯 Go 的 SQLite 驱动（不需要 CGO）
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
		NovelID: 1,
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
		NovelID:        1,
		UserSuggestion: "着重突出情感线和成长线",
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
	err = db.AutoMigrate(&model.Plotline{})
	require.NoError(t, err, "迁移表结构失败")

	return db
}
