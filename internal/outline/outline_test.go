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
		OriginalOutline: "亿万富豪叶景穿越到万妖大陆，重生为神兽麒麟幼崽，被一个大家族偶然得到，消息泄露后，家族遭隐世世家灭族，唯有一幼女逃脱，男主契约为女孩本命灵兽，助其修炼，复仇",
	}

	result, _ := compose.ExecuteOutlineWorkflow(ctx, graph, input)
	pretty.Println(*result)
}

func TestOutlineWorkflowGraph_QuickRevision(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewRepository(db)

	graph, err := compose.BuildOutlineWorkflowGraph(ctx, nil, repo)
	require.NoError(t, err)

	input := &compose.OutlineWorkflowInput{
		NovelID:        1,
		Genre:          "玄幻",
		UserSuggestion: "增强主角的成长线。",
	}

	compose.ExecuteOutlineWorkflow(ctx, graph, input)
}
