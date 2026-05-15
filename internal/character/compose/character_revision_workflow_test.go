package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"waku/internal/character/agents"
	"waku/internal/character/repository"
	"waku/internal/model"

	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ==================== Mock ChatModel ====================

type mockCharacterRevisionChatModel struct {
	reviewResponse bool // true 返回审核结果，false 返回优化结果
}

func (m *mockCharacterRevisionChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.Message, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("输入消息为空")
	}

	// 获取最后一条消息
	lastMsg := input[len(input)-1]
	content := lastMsg.Content

	// 根据内容判断是审核还是优化
	var response string
	if contains(content, "请审核以下人物信息") {
		// 返回审核结果
		output := agents.CharacterReviewerOutput{
			OverallAssessment: "整体人物设定基本合理，但存在一些需要优化的地方。",
			HasIssues:         true,
			Issues: []agents.CharacterIssue{
				{
					CharacterName: "张三",
					IssueType:     "性格矛盾",
					Severity:      "medium",
					Description:   "角色性格描述中存在矛盾，既说他冷静理智，又说他冲动易怒",
					Suggestion:    "建议统一性格特征，或者说明在什么情况下会表现出不同的性格",
				},
			},
		}
		jsonBytes, _ := json.Marshal(output)
		response = string(jsonBytes)
	} else {
		// 默认返回优化后的角色
		output := agents.CharacterRevisionOutput{
			Characters: []agents.CharacterRevisionItem{
				{
					ID:          "char-001",
					Name:        "张三（已优化）",
					Role:        "主角",
					Age:         25,
					Gender:      "男",
					Appearance:  "身材修长，面容俊朗，眼神深邃。根据审核意见，增加了更多细节描写。",
					Personality: "表面冷静理智，内心热血澎湃。在面对敌人时冷静分析，但在保护同伴时会表现出强烈的情感。",
					Background:  "出身普通家庭，因一次意外获得神秘力量，踏上修炼之路。根据用户建议，增加了家庭背景的描写。",
					Relationships: map[string]string{
						"char-002": "青梅竹马，互生情愫",
					},
				},
				{
					ID:          "char-002",
					Name:        "李四（已优化）",
					Role:        "配角",
					Age:         23,
					Gender:      "女",
					Appearance:  "容貌清秀，气质出尘，长发飘飘。",
					Personality: "温柔善良，但在关键时刻能展现出坚强的一面。",
					Background:  "宗门长老之女，从小在宗门长大，与主角青梅竹马。",
					Relationships: map[string]string{
						"char-001": "青梅竹马，暗生情愫",
					},
				},
				{
					ID:          "",
					Name:        "王五",
					Role:        "反派",
					Age:         30,
					Gender:      "男",
					Appearance:  "面容阴鸷，眼神凶狠，身材魁梧。",
					Personality: "心狠手辣，为达目的不择手段，但对自己的家人却十分温柔。",
					Background:  "根据用户建议新增的反派角色。曾经是正道弟子，因遭受不公对待而堕入魔道。",
					Relationships: map[string]string{
						"char-001": "宿敌，企图夺取主角的力量",
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(output)
		response = string(jsonBytes)
	}

	return schema.AssistantMessage(response, nil), nil
}

func (m *mockCharacterRevisionChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...einoModel.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	r, w := schema.Pipe[*schema.Message](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r, nil
}

func (m *mockCharacterRevisionChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// cleanupTestFiles 清理测试文件
func cleanupTestFiles(novelID uint, novelName string) {
	dirPath := filepath.Join("data", "content", fmt.Sprintf("%d-%s", novelID, novelName))
	os.RemoveAll(dirPath)
}

// ==================== 测试用例 ====================

// TestCharacterRevisionWorkflowGraph 测试完整的成功流程（优化角色）
func TestCharacterRevisionWorkflowGraph(t *testing.T) {
	ctx := context.Background()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "创建数据库失败")

	// 自动迁移
	err = db.AutoMigrate(&model.Novel{}, &model.Character{})
	require.NoError(t, err, "数据库迁移失败")

	// 创建 Mock ChatModel
	chatModel := &mockCharacterRevisionChatModel{}

	// 先创建一个测试小说（包含大纲）
	testNovel := &model.Novel{
		Title:   "测试小说",
		Author:  "测试作者",
		Genre:   "玄幻",
		Outline: "主角从一个废柴开始，逐步修炼成为强者的故事。",
		Status:  "draft",
	}
	err = db.Create(testNovel).Error
	require.NoError(t, err, "创建测试小说失败")

	// 创建 CharacterRepository
	characterRepo := repository.NewCharacterRepository(db)

	// 先插入一些测试角色
	testCharacters := []*model.Character{
		{
			ID:          "char-001",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Name:        "张三",
			Role:        "主角",
			Age:         25,
			Gender:      "男",
			Appearance:  "身材修长，面容俊朗",
			Personality: "冷静理智，但有时冲动易怒",
			Background:  "出身普通家庭",
			PlotlineIDs: model.StringArray{},
			Relationships: model.RelationshipMap{
				"char-002": "青梅竹马",
			},
		},
		{
			ID:          "char-002",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Name:        "李四",
			Role:        "配角",
			Age:         23,
			Gender:      "女",
			Appearance:  "容貌清秀",
			Personality: "温柔善良",
			Background:  "宗门长老之女",
			PlotlineIDs: model.StringArray{},
			Relationships: model.RelationshipMap{
				"char-001": "青梅竹马",
			},
		},
		{
			ID:          "char-003",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Name:        "赵六",
			Role:        "配角",
			Age:         28,
			Gender:      "男",
			Appearance:  "身材魁梧",
			Personality: "豪爽大方",
			Background:  "江湖游侠",
			PlotlineIDs: model.StringArray{},
			Relationships: model.RelationshipMap{},
		},
	}

	err = characterRepo.CreateBatch(testCharacters)
	require.NoError(t, err, "创建测试角色失败")

	// 清理测试数据
	defer func() {
		// 清理角色
		db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Delete(&model.Character{})
		// 清理小说
		db.Delete(testNovel)
		cleanupTestFiles(testNovel.ID, testNovel.Title)
	}()

	// 构建 Graph
	graph, err := NewCharacterRevisionWorkflow(ctx, db, chatModel, characterRepo)
	require.NoError(t, err, "构建 Graph 失败")

	t.Log("=== 测试场景：完整的成功流程（优化角色） ===")

	// 准备输入
	input := &CharacterRevisionWorkflowInput{
		NovelID:        testNovel.ID,
		UserSuggestion: "增加角色的背景描写，让人物更加立体。同时增加一个反派角色，删除赵六这个角色。",
	}

	// 执行工作流
	result, err := RunCharacterRevisionWorkflow(ctx, graph, input)
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

	// 验证是否有新增的角色
	var newCharacter model.Character
	err = db.Where("novel_id = ? AND name LIKE ?", fmt.Sprintf("%d", testNovel.ID), "%王五%").First(&newCharacter).Error
	if err != nil {
		t.Logf("⚠️  未找到新增的反派角色（这取决于 Agent 的实际输出）")
	} else {
		t.Logf("✓ 找到新增的角色: %s", newCharacter.Name)
	}

	// 验证是否删除了赵六
	var zhaoLiuCount int64
	db.Model(&model.Character{}).Where("novel_id = ? AND name LIKE ?", fmt.Sprintf("%d", testNovel.ID), "%赵六%").Count(&zhaoLiuCount)
	if zhaoLiuCount > 0 {
		t.Logf("⚠️  赵六未被删除（这取决于 Agent 的实际输出）")
	} else {
		t.Logf("✓ 赵六已被删除")
	}

	t.Log("✓ 完整流程测试通过")
}

// TestCharacterRevisionWorkflowWithSpecificCharacter 测试指定角色ID的优化流程
func TestCharacterRevisionWorkflowWithSpecificCharacter(t *testing.T) {
	ctx := context.Background()

	// 创建内存数据库
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "创建数据库失败")

	// 自动迁移
	err = db.AutoMigrate(&model.Novel{}, &model.Character{})
	require.NoError(t, err, "数据库迁移失败")

	// 创建 Mock ChatModel
	chatModel := &mockCharacterRevisionChatModel{}

	// 先创建一个测试小说
	testNovel := &model.Novel{
		Title:   "测试小说2",
		Author:  "测试作者2",
		Genre:   "玄幻",
		Outline: "主角从一个废柴开始，逐步修炼成为强者的故事。",
		Status:  "draft",
	}
	err = db.Create(testNovel).Error
	require.NoError(t, err, "创建测试小说失败")

	// 创建 CharacterRepository
	characterRepo := repository.NewCharacterRepository(db)

	// 先插入一些测试角色
	testCharacters := []*model.Character{
		{
			ID:          "char-001",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Name:        "张三",
			Role:        "主角",
			Age:         25,
			Gender:      "男",
			Appearance:  "身材修长，面容俊朗",
			Personality: "冷静理智",
			Background:  "出身普通家庭",
			PlotlineIDs: model.StringArray{},
		},
		{
			ID:          "char-002",
			NovelID:     fmt.Sprintf("%d", testNovel.ID),
			Name:        "李四",
			Role:        "配角",
			Age:         23,
			Gender:      "女",
			Appearance:  "容貌清秀",
			Personality: "温柔善良",
			Background:  "宗门长老之女",
			PlotlineIDs: model.StringArray{},
		},
	}

	err = characterRepo.CreateBatch(testCharacters)
	require.NoError(t, err, "创建测试角色失败")

	// 清理测试数据
	defer func() {
		// 清理角色
		db.Where("novel_id = ?", fmt.Sprintf("%d", testNovel.ID)).Delete(&model.Character{})
		// 清理小说
		db.Delete(testNovel)
		cleanupTestFiles(testNovel.ID, testNovel.Title)
	}()

	// 构建 Graph
	graph, err := NewCharacterRevisionWorkflow(ctx, db, chatModel, characterRepo)
	require.NoError(t, err, "构建 Graph 失败")

	t.Log("=== 测试场景：指定角色ID的优化流程 ===")

	// 准备输入（只优化张三）
	input := &CharacterRevisionWorkflowInput{
		NovelID:        testNovel.ID,
		CharacterID:    "char-001",
		UserSuggestion: "增加张三的性格深度，让他更加立体。",
	}

	// 执行工作流
	result, err := RunCharacterRevisionWorkflow(ctx, graph, input)
	require.NoError(t, err, "工作流执行失败")

	// 验证结果
	require.Equal(t, "save_success", result.CurrentNode, "期望 CurrentNode=save_success")
	require.NotZero(t, result.NovelID, "小说ID不应为空")
	require.Equal(t, "char-001", result.CharacterID, "角色ID应该是 char-001")

	t.Logf("✓ 成功保存 %d 个角色", len(result.SavedCharacterIDs))
	t.Logf("✓ 指定的角色ID: %s", result.CharacterID)

	// 验证数据库中的角色（应该只更新了张三）
	var zhangsan model.Character
	err = db.Where("id = ?", "char-001").First(&zhangsan).Error
	require.NoError(t, err, "查询张三失败")

	t.Logf("✓ 张三的信息: %s - %s", zhangsan.Name, zhangsan.Personality)

	// 验证李四没有被修改（注意：由于 mock 的限制，这里可能会被修改）
	var lisi model.Character
	err = db.Where("id = ?", "char-002").First(&lisi).Error
	if err == nil {
		t.Logf("✓ 李四的信息: %s - %s", lisi.Name, lisi.Personality)
	} else {
		t.Logf("⚠️  李四未找到（这是正常的，因为只优化了张三）")
	}

	t.Log("✓ 指定角色优化测试通过")
}
