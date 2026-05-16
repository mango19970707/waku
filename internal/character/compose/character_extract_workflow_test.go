package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"waku/internal/character/agents"
	"waku/internal/character/repository"
	"waku/internal/model"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	_ "modernc.org/sqlite" // 使用纯 Go 的 SQLite 驱动（不需要 CGO）
	"gorm.io/gorm"
)

// ==================== Mock ChatModel ====================

type mockCharacterExtractChatModel struct {
	failCount int
}

func (m *mockCharacterExtractChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.Message, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("输入消息为空")
	}

	// 获取最后一条消息
	lastMsg := input[len(input)-1]
	content := lastMsg.Content

	// 根据内容判断是提取请求
	var response string
	if contains(content, "请从以下小说大纲中提取人物信息") || contains(content, "大纲") {
		// 返回提取结果
		output := agents.CharacterExtractorOutput{
			Characters: []agents.CharacterItem{
				{
					Name:        "张三",
					Role:        "主角",
					Age:         25,
					Gender:      "男",
					Appearance:  "身材修长，面容俊朗，眼神深邃",
					Personality: "冷静理智，但有时冲动易怒",
					Background:  "出身普通家庭，因一次意外获得神秘力量",
					Relationships: map[string]string{
						"character-2": "师姐，互生情愫",
						"character-3": "宿敌，多次交手",
					},
				},
				{
					Name:        "李四",
					Role:        "配角",
					Age:         23,
					Gender:      "女",
					Appearance:  "容貌清秀，气质出尘",
					Personality: "温柔善良，但在关键时刻能展现出坚强的一面",
					Background:  "宗门长老之女，从小在宗门长大",
					Relationships: map[string]string{
						"character-1": "师弟，暗生情愫",
					},
				},
				{
					Name:        "王五",
					Role:        "反派",
					Age:         30,
					Gender:      "男",
					Appearance:  "面容阴鸷，眼神凶狠",
					Personality: "心狠手辣，为达目的不择手段",
					Background:  "曾经是正道弟子，因遭受不公对待而堕入魔道",
					Relationships: map[string]string{
						"character-1": "主角，企图夺取其力量",
					},
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

func (m *mockCharacterExtractChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	r, w := schema.Pipe[*schema.Message](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r, nil
}

func (m *mockCharacterExtractChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

// ==================== 测试用例 ====================

// TestCharacterExtractWorkflowGraph 测试完整的成功流程（提取人物）
func TestCharacterExtractWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        ":memory:",
	}, &gorm.Config{})
	require.NoError(t, err, "创建数据库失败")

	// 自动迁移
	err = db.AutoMigrate(&model.Novel{}, &model.Character{})
	require.NoError(t, err, "数据库迁移失败")

	// 创建 Mock ChatModel
	chatModel := &mockCharacterExtractChatModel{failCount: 0}

	// 先创建一个测试小说（包含大纲）
	testNovel := &model.Novel{
		Title:   "测试小说",
		Author:  "测试作者",
		Genre:   "玄幻",
		Outline: "主角张三是一个普通少年，因意外获得神秘力量，踏上修仙之路。在修炼过程中，他结识了师姐李四，两人互生情愫。但反派王五却一直在暗中谋划，企图夺取主角的力量。",
		Status:  "draft",
	}
	err = db.Create(testNovel).Error
	require.NoError(t, err, "创建测试小说失败")

	// 创建 Repository
	novelRepo := &mockNovelRepository{db: db}
	characterRepo := repository.NewCharacterRepository(db)

	// 清理测试数据
	defer func() {
		// 清理角色
		db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Delete(&model.Character{})
		// 清理小说
		db.Delete(testNovel)
		cleanupTestFiles(testNovel.ID, testNovel.Title)
	}()

	// 构建 Graph
	graph, err := NewCharacterExtractWorkflow(ctx, chatModel, novelRepo, characterRepo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：完整的成功流程（提取人物） ===")

	// 准备输入
	input := &CharacterExtractWorkflowInput{
		NovelID: testNovel.ID,
	}

	// 执行工作流
	result, err := RunCharacterExtractWorkflow(ctx, graph, input)
	require.NoError(t, err, "工作流执行失败")

	// 验证结果
	if result.CurrentNode != "save_success" {
		t.Errorf("期望 CurrentNode=save_success，实际: %s", result.CurrentNode)
	}

	if result.NovelID == 0 {
		t.Error("小说ID为空")
	}

	// 验证保存的角色数量
	if len(result.SavedCharacterIDs) == 0 {
		t.Error("未保存任何角色")
	}

	t.Logf("✓ 成功保存 %d 个角色", len(result.SavedCharacterIDs))
	t.Logf("✓ 角色ID列表: %v", result.SavedCharacterIDs)

	// 验证数据库中的角色
	var characters []model.Character
	err = db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Find(&characters).Error
	require.NoError(t, err, "查询角色失败")

	t.Logf("数据库中的角色数量: %d", len(characters))

	// 验证角色内容
	for i, character := range characters {
		t.Logf("角色 %d: %s (%s)", i+1, character.Name, character.Role)
		if character.Name == "" {
			t.Errorf("角色 %d 的名称为空", i+1)
		}
		if character.Role == "" {
			t.Errorf("角色 %d 的角色定位为空", i+1)
		}
		if character.Appearance == "" {
			t.Errorf("角色 %d 的外貌描述为空", i+1)
		}
		if character.Personality == "" {
			t.Errorf("角色 %d 的性格特征为空", i+1)
		}
		if character.Background == "" {
			t.Errorf("角色 %d 的背景故事为空", i+1)
		}
		// 验证关系字段
		if character.Relationships != nil && len(character.Relationships) > 0 {
			t.Logf("  关系数量: %d", len(character.Relationships))
			for relatedID, relation := range character.Relationships {
				t.Logf("    - %s: %s", relatedID, relation)
			}
		}
	}

	// 验证是否提取了预期的角色
	var zhangsan model.Character
	err = db.Where("novel_id = ? AND name = ?", fmt.Sprintf("%d", testNovel.ID), "张三").First(&zhangsan).Error
	if err != nil {
		t.Errorf("未找到主角张三")
	} else {
		t.Logf("✓ 找到主角: %s - %s", zhangsan.Name, zhangsan.Role)
	}

	var lisi model.Character
	err = db.Where("novel_id = ? AND name = ?", fmt.Sprintf("%d", testNovel.ID), "李四").First(&lisi).Error
	if err != nil {
		t.Errorf("未找到配角李四")
	} else {
		t.Logf("✓ 找到配角: %s - %s", lisi.Name, lisi.Role)
	}

	var wangwu model.Character
	err = db.Where("novel_id = ? AND name = ?", fmt.Sprintf("%d", testNovel.ID), "王五").First(&wangwu).Error
	if err != nil {
		t.Errorf("未找到反派王五")
	} else {
		t.Logf("✓ 找到反派: %s - %s", wangwu.Name, wangwu.Role)
	}

	// 验证角色关系字段
	t.Log("=== 验证角色关系字段 ===")
	if len(zhangsan.Relationships) > 0 {
		t.Logf("✓ 张三的关系数量: %d", len(zhangsan.Relationships))
		for relatedID, relation := range zhangsan.Relationships {
			t.Logf("  - %s: %s", relatedID, relation)
		}
	} else {
		t.Log("⚠ 张三没有关系数据")
	}

	if len(lisi.Relationships) > 0 {
		t.Logf("✓ 李四的关系数量: %d", len(lisi.Relationships))
		for relatedID, relation := range lisi.Relationships {
			t.Logf("  - %s: %s", relatedID, relation)
		}
	} else {
		t.Log("⚠ 李四没有关系数据")
	}

	if len(wangwu.Relationships) > 0 {
		t.Logf("✓ 王五的关系数量: %d", len(wangwu.Relationships))
		for relatedID, relation := range wangwu.Relationships {
			t.Logf("  - %s: %s", relatedID, relation)
		}
	} else {
		t.Log("⚠ 王五没有关系数据")
	}

	t.Log("✓ 完整流程测试通过")
}

// TestCharacterExtractWorkflowNoOutline 测试没有大纲的情况
func TestCharacterExtractWorkflowNoOutline(t *testing.T) {
	ctx := context.Background()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        ":memory:",
	}, &gorm.Config{})
	require.NoError(t, err, "创建数据库失败")

	// 自动迁移
	err = db.AutoMigrate(&model.Novel{}, &model.Character{})
	require.NoError(t, err, "数据库迁移失败")

	// 创建 Mock ChatModel
	chatModel := &mockCharacterExtractChatModel{failCount: 0}

	// 创建一个没有大纲的测试小说
	testNovel := &model.Novel{
		Title:   "测试小说-无大纲",
		Author:  "测试作者",
		Genre:   "玄幻",
		Outline: "", // 空大纲
		Status:  "draft",
	}
	err = db.Create(testNovel).Error
	require.NoError(t, err, "创建测试小说失败")

	// 创建 Repository
	novelRepo := &mockNovelRepository{db: db}
	characterRepo := repository.NewCharacterRepository(db)

	// 清理测试数据
	defer func() {
		db.Delete(testNovel)
		cleanupTestFiles(testNovel.ID, testNovel.Title)
	}()

	// 构建 Graph
	graph, err := NewCharacterExtractWorkflow(ctx, chatModel, novelRepo, characterRepo)
	if err != nil {
		t.Fatalf("构建 Graph 失败: %v", err)
	}

	t.Log("=== 测试场景：没有大纲的情况 ===")

	// 准备输入
	input := &CharacterExtractWorkflowInput{
		NovelID: testNovel.ID,
	}

	// 执行工作流（应该失败）
	_, err = RunCharacterExtractWorkflow(ctx, graph, input)
	if err == nil {
		t.Error("期望工作流失败，但实际成功了")
	} else {
		t.Logf("✓ 正确处理了空大纲的情况: %v", err)
	}
}

// ==================== Mock NovelRepository ====================

type mockNovelRepository struct {
	db *gorm.DB
}

func (m *mockNovelRepository) GetByID(id uint) (*model.Novel, error) {
	var novel model.Novel
	if err := m.db.First(&novel, id).Error; err != nil {
		return nil, err
	}
	return &novel, nil
}

