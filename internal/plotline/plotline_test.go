package plotline

import (
	"context"
	"database/sql"
	"testing"
	"waku/internal/config"
	"waku/internal/model"
	novelRepoository "waku/internal/outline/repository"
	"waku/internal/plotline/compose"
	"waku/internal/plotline/repository"
	"waku/internal/utils"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

var (
	cfg *config.Config
	db  *gorm.DB
)

func init() {
	// 加载配置
	cfg, _ = config.Load("../../config/config.yaml")

	var err error
	db, err = gorm.Open(sqlite.New(sqlite.Config{
		DriverName: "sqlite",
		DSN:        "../../data/waku.db?cache=shared",
	}), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err = db.AutoMigrate(&model.Novel{},
		&model.Plotline{}); err != nil {
		panic(err)
	}
}

func TestPlotlineWorkflowGraph_FullGeneration(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	plotLineRepo := repository.NewPlotlineRepository(db)
	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.BuildPlotlineWorkflowGraph(ctx, chatModel, novelRepo, plotLineRepo)
	require.NoError(t, err)

	input := &compose.PlotlineWorkflowInput{
		NovelID: 6,
	}

	result, _ := compose.ExecutePlotlineWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestPlotlineWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	plotLineRepo := repository.NewPlotlineRepository(db)
	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.BuildPlotlineWorkflowGraph(ctx, chatModel, novelRepo, plotLineRepo)
	require.NoError(t, err)

	input := &compose.PlotlineWorkflowInput{
		NovelID: 6,
	}

	result, _ := compose.ExecutePlotlineWorkflow(ctx, graph, input)
	pretty.Println(*result)
}
