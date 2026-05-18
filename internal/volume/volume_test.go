package volume

import (
	"context"
	"database/sql"
	"testing"
	"waku/internal/config"
	"waku/internal/model"
	novelRepoository "waku/internal/outline/repository"
	plotlineRepoository "waku/internal/plotline/repository"
	"waku/internal/utils"
	"waku/internal/volume/compose"
	"waku/internal/volume/repository"

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
		&model.Plotline{}, &model.Character{}, &model.Volume{}); err != nil {
		panic(err)
	}
}

func TestCharaterWorkflowGraph_FullGeneration(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	volumeRepo := repository.NewVolumeRepository(db)
	plotlineRepo := plotlineRepoository.NewPlotlineRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewVolumeWorkflow(ctx, chatModel, novelRepo, plotlineRepo, volumeRepo)
	require.NoError(t, err)

	input := &compose.VolumeWorkflowInput{
		NovelID: 1,
	}

	result, _ := compose.ExecuteVolumeWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestCharaterWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	volumeRepo := repository.NewVolumeRepository(db)
	plotlineRepo := plotlineRepoository.NewPlotlineRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewVolumeWorkflow(ctx, chatModel, novelRepo, plotlineRepo, volumeRepo)
	require.NoError(t, err)

	input := &compose.VolumeWorkflowInput{
		NovelID: 1,
	}

	result, _ := compose.ExecuteVolumeWorkflow(ctx, graph, input)
	pretty.Println(*result)
}
