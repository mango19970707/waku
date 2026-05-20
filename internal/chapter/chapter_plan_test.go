package chapter

import (
	"context"
	"database/sql"
	"testing"
	"waku/internal/chapter/compose"
	"waku/internal/chapter/repository"
	characterRepoository "waku/internal/character/repository"
	"waku/internal/config"
	"waku/internal/model"
	novelRepoository "waku/internal/outline/repository"
	plotlineRepoository "waku/internal/plotline/repository"
	"waku/internal/utils"
	volumeRepository "waku/internal/volume/repository"

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
		&model.Plotline{}, &model.Character{}, &model.Volume{},
		&model.Chapter{}, &model.Event{}, &model.PlotlineProgress{}); err != nil {
		panic(err)
	}
}

func TestChapterWorkflowGraph_FullGeneration(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	volumeRepo := volumeRepository.NewVolumeRepository(db)
	plotlineRepo := plotlineRepoository.NewPlotlineRepository(db)
	characterRepo := characterRepoository.NewCharacterRepository(db)
	chapterRepo := repository.NewChapterRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewChapterWorkflow(ctx, chatModel, novelRepo, volumeRepo, plotlineRepo, characterRepo, chapterRepo)
	require.NoError(t, err)

	input := &compose.ChapterWorkflowInput{
		VolumeID: 1,
		NovelID:  1,
	}

	result, _ := compose.ExecuteChapterWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestChapterWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	volumeRepo := volumeRepository.NewVolumeRepository(db)
	plotlineRepo := plotlineRepoository.NewPlotlineRepository(db)
	characterRepo := characterRepoository.NewCharacterRepository(db)
	chapterRepo := repository.NewChapterRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewChapterWorkflow(ctx, chatModel, novelRepo, volumeRepo, plotlineRepo, characterRepo, chapterRepo)
	require.NoError(t, err)

	input := &compose.ChapterWorkflowInput{
		VolumeID: 1,
		NovelID:  1,
	}

	result, _ := compose.ExecuteChapterWorkflow(ctx, graph, input)
	pretty.Println(*result)
}
