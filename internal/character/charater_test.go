package character

import (
	"context"
	"database/sql"
	"testing"
	"waku/internal/config"
	"waku/internal/model"
	novelRepoository "waku/internal/outline/repository"
	"waku/internal/character/compose"
	//plotlineRepoository "waku/internal/plotline/repository"
	"waku/internal/character/repository"
	"waku/internal/utils"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

func TestCharaterWorkflowGraph_FullGeneration(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	characterRepo := repository.NewCharacterRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewCharacterWorkflow(ctx, chatModel, novelRepo, characterRepo)
	require.NoError(t, err)

	input := &compose.CharacterWorkflowInput{
		NovelID: 1,
	}

	result, _ := compose.ExecuteCharacterWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestCharaterWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	novelRepo := novelRepoository.NewRepository(db)
	characterRepo := repository.NewCharacterRepository(db)

	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.NewCharacterWorkflow(ctx, chatModel, novelRepo, characterRepo)
	require.NoError(t, err)

	input := &compose.CharacterWorkflowInput{
		NovelID: 1,
	}

	result, _ := compose.ExecuteCharacterWorkflow(ctx, graph, input)
	pretty.Println(*result)
}
