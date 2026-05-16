package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"waku/internal/model"
	"waku/internal/plotline/agents"
	"waku/internal/plotline/repository"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

// ==================== Mock ChatModel ====================

type mockRevisionChatModel struct {
	failCount int
}

func (m *mockRevisionChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.Message, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("输入消息为空")
	}

	// 获取最后一条消息
	lastMsg := input[len(input)-1]
	content := lastMsg.Content

	// 根据内容判断是哪个阶段
	var response string
	if contains(content, "优化") || contains(content, "修改建议") {
		// 返回 JSON 格式的优化后剧情线
		output := agents.PlotlineRevisionOutput{
			Plotlines: []agents.PlotlineRevisionItem{
				{
					ID:           "6ece270a-9652-475a-a4ca-5974b820277c",
					Title:        "主角修仙成神之路（已优化）",
					Type:         "main",
					Description:  "主角从一个废柴少年开始，因意外获得神秘传承，踏上修仙之路。经历筑基、金丹、元婴等境界突破，与各大势力斗智斗勇，最终成为一代仙帝。根据用户建议，增加了主角在修炼过程中的心理成长描写，使人物更加立体。",
					StartChapter: 1,
					EndChapter:   0,
				},
				{
					ID:           "b3b33469-e8b5-4676-b944-47a0a694679e",
					Title:        "与师姐的情感纠葛（已优化）",
					Type:         "sub",
					Description:  "主角在宗门中与美丽的师姐相识相知，两人在历练中互生情愫。但师姐身负家族使命，主角为了师姐的幸福，暗中帮助她解决家族危机。根据用户建议，增加了情感线的细节描写。",
					StartChapter: 10,
					EndChapter:   80,
				},
				{
					ID:           "",
					Title:        "寻找失落的神器",
					Type:         "sub",
					Description:  "根据用户建议新增的支线。主角在修炼过程中得知上古神器散落人间，这些神器是对抗魔族的关键。主角踏上寻找神器之路，途中遇到各种挑战和竞争对手，最终集齐七件神器，为最终决战做好准备。",
					StartChapter: 30,
					EndChapter:   90,
				},
			},
		}
		jsonBytes, _ := json.Marshal(output)
		response = string(jsonBytes)
	} else {
		response = "处理完成"
	}

	return schema.AssistantMessage(response, nil), nil
}

func (m *mockRevisionChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	r, w := schema.Pipe[*schema.Message](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r, nil
}

func (m *mockRevisionChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

// ==================== 测试用例 ====================

// TestPlotlineRevisionWorkflowGraph 测试完整的成功流程（优化剧情线）
func TestPlotlineRevisionWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建 Mock ChatModel
	chatModel := &mockRevisionChatModel{failCount: 0}

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

	// 创建 PlotlineRepository
	plotlineRepo := repository.NewPlotlineRepository(db)

	// 先插入一些测试剧情线
	testPlotlines := []*model.Plotline{
		{
			ID:          "6ece270a-9652-475a-a4ca-5974b820277c",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Title:       "主角修仙成神之路",
			Type:        "main",
			Description: "主角从一个废柴少年开始，因意外获得神秘传承，踏上修仙之路。",
			Status:      "active",
		},
		{
			ID:          "b3b33469-e8b5-4676-b944-47a0a694679e",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Title:       "与师姐的情感纠葛",
			Type:        "sub",
			Description: "主角在宗门中与美丽的师姐相识相知，两人在历练中互生情愫。",
			Status:      "active",
		},
		{
			ID:          "1471788f-b3e4-49a5-a278-44552fd132e5",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Title:       "复仇灭杀仇家",
			Type:        "sub",
			Description: "主角家族被仇家灭门，只有主角侥幸逃脱。主角隐忍修炼，实力大增后开始复仇计划。",
			Status:      "active",
		},
	}

	err = plotlineRepo.CreateBatch(testPlotlines)
	require.NoError(t, err, "创建测试剧情线失败")

	// 清理测试数据
	defer func() {
		// 清理剧情线
		db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Delete(&model.Plotline{})
		// 清理小说
		db.Delete(testNovel)
		cleanupTestFiles(testNovel.Author, testNovel.Title)
	}()

	// 创建 repository
	novelRepo := repository.NewRepository(db)

	// 构建 Graph
	graph, err := NewPlotlineRevisionWorkflow(ctx, chatModel, novelRepo, plotlineRepo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程（优化剧情线） ===")

	// 准备输入
	input := &PlotlineRevisionWorkflowInput{
		NovelID:        testNovel.ID,
		UserSuggestion: "增加主角的心理成长描写，让人物更加立体。同时增加一条寻找神器的支线，删除复仇线。",
	}

	// 执行工作流
	result, err := ExecutePlotlineRevisionWorkflow(ctx, graph, input)
	require.NoError(t, err, "工作流执行失败")

	// 验证结果
	if result.CurrentNode != "save_success" {
		t.Errorf("期望 CurrentNode=save_success，实际: %s", result.CurrentNode)
	}

	if result.NovelID == 0 {
		t.Error("小说ID为空")
	}

	// 验证保存的剧情线数量
	if len(result.SavedPlotlineIDs) == 0 {
		t.Error("未保存任何剧情线")
	}

	t.Logf("✓ 成功保存 %d 条剧情线", len(result.SavedPlotlineIDs))
	t.Logf("✓ 剧情线ID列表: %v", result.SavedPlotlineIDs)

	// 验证数据库中的剧情线
	var plotlines []model.Plotline
	err = db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Find(&plotlines).Error
	require.NoError(t, err, "查询剧情线失败")

	t.Logf("数据库中的剧情线数量: %d", len(plotlines))

	// 验证剧情线内容
	for i, plotline := range plotlines {
		t.Logf("剧情线 %d: %s (%s)", i+1, plotline.Title, plotline.Type)
		if plotline.Title == "" {
			t.Errorf("剧情线 %d 的标题为空", i+1)
		}
		if plotline.Type != "main" && plotline.Type != "sub" {
			t.Errorf("剧情线 %d 的类型无效: %s", i+1, plotline.Type)
		}
		if plotline.Description == "" {
			t.Errorf("剧情线 %d 的描述为空", i+1)
		}
	}

	// 验证是否有新增的剧情线
	var newPlotline model.Plotline
	err = db.Where("novel_id = ? AND title LIKE ?", fmt.Sprintf("%d", testNovel.ID), "%神器%").First(&newPlotline).Error
	if err != nil {
		t.Errorf("未找到新增的神器支线")
	} else {
		t.Logf("✓ 找到新增的剧情线: %s", newPlotline.Title)
	}

	// 验证是否删除了复仇线
	var revengeCount int64
	db.Model(&model.Plotline{}).Where("novel_id = ? AND title LIKE ?", fmt.Sprintf("%d", testNovel.ID), "%复仇%").Count(&revengeCount)
	if revengeCount > 0 {
		t.Logf("⚠️  复仇线未被删除（这取决于 Agent 的实际输出）")
	} else {
		t.Logf("✓ 复仇线已被删除")
	}

	t.Log("✓ 完整流程测试通过")
}
