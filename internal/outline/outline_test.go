package outline

import (
	"context"
	"database/sql"
	"testing"
	"waku/internal/config"
	"waku/internal/model"
	"waku/internal/outline/compose"
	"waku/internal/outline/repository"
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

	if err = db.AutoMigrate(&model.Novel{}); err != nil {
		panic(err)
	}
}

func TestOutlineWorkflowGraph_FullGeneration(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(db)
	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)
	if err != nil {
		t.Fatal("new chatModel", err)
	}

	graph, err := compose.BuildOutlineWorkflowGraph(ctx, chatModel, repo)
	require.NoError(t, err)

	input := &compose.OutlineWorkflowInput{
		Author:          "哐啷小牛",
		Genre:           "玄幻",
		OriginalOutline: "草根崛起退婚流",
	}

	result, _ := compose.ExecuteOutlineWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestOutlineWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(db)
	chatModel, err := utils.NewChatModel(ctx, &cfg.Model)

	graph, err := compose.BuildOutlineWorkflowGraph(ctx, chatModel, repo)
	require.NoError(t, err)

	input := &compose.OutlineWorkflowInput{
		NovelID:        6,
		Genre:          "玄幻",
		UserSuggestion: "优化妖兽境界，和人族修炼境界一一对应；主角不是双修，身为妖兽，只修炼妖兽的功法和境界",
	}

	result, _ := compose.ExecuteOutlineWorkflow(ctx, graph, input)
	pretty.Println(*result)
}
