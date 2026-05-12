package novel

import (
	"context"
	"log/slog"
	"testing"
	"waku/internal/config"
	"waku/internal/novel/agents"
	"waku/internal/utils"

	"github.com/cloudwego/eino/components/model"
)

var (
	chatModel model.BaseChatModel
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

func TestGenerateOutlineWithWorkflow(t *testing.T) {
	db := SetupTestDBForTools(t)
	repo := NewRepository(db)
	outlineAgent, err := agents.NewOutlineWorkflowAgent(context.Background(), chatModel, db, repo)
	if err != nil {
		t.Fatal(err)
	}

	input := &agents.OutlineGeneratorInput{
		OriginalOutline: "废柴流",
		Genre:           "玄幻",
	}

	rst, err := agents.GenerateOutlineWithWorkflow(context.Background(), outlineAgent, input)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(rst)
}
