package content

import (
	"context"
	"database/sql"
	"testing"
	charactereRepo "waku/internal/character/repository"
	"waku/internal/config"
	"waku/internal/content/compose"
	"waku/internal/content/repository"
	"waku/internal/model"
	novelRepoository "waku/internal/outline/repository"
	plotlineRepoository "waku/internal/plotline/repository"
	"waku/internal/utils"
	volumeRepo "waku/internal/volume/repository"

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
	volumeRepo := volumeRepo.NewVolumeRepository(db)
	plotlineRepo := plotlineRepoository.NewPlotlineRepository(db)
	characterRepo := charactereRepo.NewCharacterRepository(db)
	contentRepo := repository.NewContentRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewContentWorkflow(ctx, chatModel, novelRepo, volumeRepo, plotlineRepo, characterRepo, contentRepo)
	require.NoError(t, err)

	input := &compose.ContentWorkflowInput{
		NovelID:  1,
		VolumeID: 1,
		//ChapterID: 91,
		ChapterIDs: []uint{102, 103},
	}

	result, _ := compose.ExecuteContentWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestCharaterWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	volumeRepo := volumeRepo.NewVolumeRepository(db)
	plotlineRepo := plotlineRepoository.NewPlotlineRepository(db)
	characterRepo := charactereRepo.NewCharacterRepository(db)
	contentRepo := repository.NewContentRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewContentWorkflow(ctx, chatModel, novelRepo, volumeRepo, plotlineRepo, characterRepo, contentRepo)
	require.NoError(t, err)

	input := &compose.ContentWorkflowInput{
		NovelID: 1,
	}

	result, _ := compose.ExecuteContentWorkflow(ctx, graph, input)
	pretty.Println(*result)
}
