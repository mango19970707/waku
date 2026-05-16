package tools

import (
	"context"
	"encoding/json"
	"testing"
	"waku/internal/outline/repository"

	"waku/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	_ "modernc.org/sqlite" // 使用纯 Go 的 SQLite 驱动（不需要 CGO）
	"gorm.io/gorm"
)

// setupTestDBForTools 创建测试数据库
func SetupTestDBForTools(t *testing.T) *gorm.DB {
	// 使用项目根目录的 data 目录存储测试数据库
	dbPath := "../../data/waku.db"

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dbPath,
	}, &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	err = db.AutoMigrate(&model.Novel{})
	require.NoError(t, err, "迁移表结构失败")

	return db
}

// TestSaveOutlineTool 测试保存大纲工具
func TestSaveOutlineTool(t *testing.T) {
	db := SetupTestDBForTools(t)
	ctx := context.Background()

	// 创建工具
	repo := repository.NewRepository(db)
	tool, err := NewSaveOutlineTool(db, repo)
	require.NoError(t, err, "创建工具失败")

	// 准备测试大纲
	outline := `# 斗破苍穹大纲

## 1. 故事背景
- 时代背景：斗气大陆，修炼斗气的世界
- 地理环境：西北域、中州、东域等
- 社会结构：宗门、家族、散修

## 2. 世界观
- 世界构成：斗气大陆，分为多个区域
- 天地规则：斗气修炼体系
- 种族分布：人族为主，魔兽族为辅
- 历史脉络：远古斗帝消失，斗气衰落

## 3. 修行体系
- 境界划分：
  1. 斗之气（1-10段）
  2. 斗者（1-9星）
  3. 斗师（1-9星）
  4. 大斗师（1-9星）
  5. 斗灵（1-9星）
  6. 斗王（1-9星）
  7. 斗皇（1-9星）
  8. 斗宗（1-9星）
  9. 斗尊（1-9星）
  10. 斗圣（1-9星）
  11. 斗帝
- 修炼方式：吸收天地灵气，修炼功法
- 力量体系：斗气
- 特殊能力：异火、血脉
- 突破条件：感悟、资源、天劫

## 4. 核心冲突
- 主要矛盾：萧炎复仇与成长
- 次要矛盾：家族恩怨、宗门争斗
- 冲突根源：纳兰嫣然退婚

## 5. 主线剧情
### 开端
- 萧炎被退婚，斗气消失
- 遇到药老，获得异火

### 发展
- 修炼提升，进入迦南学院
- 收集异火，提升实力

### 高潮
- 与魂族决战
- 成为斗帝

### 结局
- 击败魂天帝
- 成为斗气大陆最强者`

	// 构建工具输入参数
	input := SaveOutlineToolInput{
		Title:             "斗破苍穹",
		Author:            "天蚕土豆",
		Genre:             "玄幻",
		Synopsis:          "三十年河东，三十年河西，莫欺少年穷！",
		Outline:           outline,
		Worldview:         "斗气大陆，修炼斗气的世界，包含西北域、中州、东域等区域",
		CultivationSystem: "斗之气、斗者、斗师、大斗师、斗灵、斗王、斗皇、斗宗、斗尊、斗圣、斗帝",
	}

	// 将输入转换为 JSON
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err, "序列化输入失败")

	// 调用工具
	resultJSON, err := tool.InvokableRun(ctx, string(inputJSON))
	require.NoError(t, err, "调用工具失败")

	t.Logf("工具返回结果: %s", resultJSON)

	// 解析返回结果
	var result SaveOutlineToolOutput
	err = json.Unmarshal([]byte(resultJSON), &result)
	require.NoError(t, err, "解析返回结果失败")

	// 验证结果
	assert.True(t, result.Success, "结果应该成功")
	assert.NotEmpty(t, result.NovelID, "结果应包含 novel_id")
	assert.Contains(t, result.Message, "成功", "结果应包含成功信息")

	// 验证数据库中的数据
	novels, err := repo.GetAll(0, 10)
	require.NoError(t, err)
	assert.Len(t, novels, 1, "数据库中应该有 1 部小说")

	novel := novels[0]
	assert.Equal(t, "斗破苍穹", novel.Title)
	assert.Equal(t, "天蚕土豆", novel.Author)
	assert.Equal(t, "玄幻", novel.Genre)
	assert.Contains(t, novel.Worldview, "斗气大陆")
	assert.Contains(t, novel.CultivationSystem, "斗帝")
}

// TestSaveOutlineToolValidation 测试工具参数验证
func TestSaveOutlineToolValidation(t *testing.T) {
	db := SetupTestDBForTools(t)
	ctx := context.Background()

	repo := repository.NewRepository(db)
	tool, err := NewSaveOutlineTool(db, repo)
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       SaveOutlineToolInput
		expectError bool
		errorMsg    string
	}{
		{
			name: "缺少标题",
			input: SaveOutlineToolInput{
				Author:            "作者",
				Genre:             "玄幻",
				Synopsis:          "简介",
				Outline:           "大纲",
				Worldview:         "世界观",
				CultivationSystem: "修行体系",
			},
			expectError: false,
			errorMsg:    "小说标题不能为空",
		},
		{
			name: "缺少作者",
			input: SaveOutlineToolInput{
				Title:             "标题",
				Genre:             "玄幻",
				Synopsis:          "简介",
				Outline:           "大纲",
				Worldview:         "世界观",
				CultivationSystem: "修行体系",
			},
			expectError: false,
			errorMsg:    "作者名称不能为空",
		},
		{
			name: "缺少类型",
			input: SaveOutlineToolInput{
				Title:             "标题",
				Author:            "作者",
				Synopsis:          "简介",
				Outline:           "大纲",
				Worldview:         "世界观",
				CultivationSystem: "修行体系",
			},
			expectError: false,
			errorMsg:    "小说类型不能为空",
		},
		{
			name: "缺少简介",
			input: SaveOutlineToolInput{
				Title:             "标题",
				Author:            "作者",
				Genre:             "玄幻",
				Outline:           "大纲",
				Worldview:         "世界观",
				CultivationSystem: "修行体系",
			},
			expectError: false,
			errorMsg:    "小说简介不能为空",
		},
		{
			name: "缺少大纲",
			input: SaveOutlineToolInput{
				Title:             "标题",
				Author:            "作者",
				Genre:             "玄幻",
				Synopsis:          "简介",
				Worldview:         "世界观",
				CultivationSystem: "修行体系",
			},
			expectError: false,
			errorMsg:    "大纲内容不能为空",
		},
		{
			name: "缺少世界观",
			input: SaveOutlineToolInput{
				Title:             "标题",
				Author:            "作者",
				Genre:             "玄幻",
				Synopsis:          "简介",
				Outline:           "大纲",
				CultivationSystem: "修行体系",
			},
			expectError: false,
			errorMsg:    "世界观不能为空",
		},
		{
			name: "缺少修行体系",
			input: SaveOutlineToolInput{
				Title:     "标题",
				Author:    "作者",
				Genre:     "玄幻",
				Synopsis:  "简介",
				Outline:   "大纲",
				Worldview: "世界观",
			},
			expectError: false,
			errorMsg:    "修行体系不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 将输入转换为 JSON
			inputJSON, err := json.Marshal(tt.input)
			require.NoError(t, err, "序列化输入失败")

			// 调用工具
			resultJSON, err := tool.InvokableRun(ctx, string(inputJSON))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// 解析返回结果
				var result SaveOutlineToolOutput
				err = json.Unmarshal([]byte(resultJSON), &result)
				require.NoError(t, err, "解析返回结果失败")

				assert.False(t, result.Success)
				assert.Contains(t, result.Message, tt.errorMsg)
			}
		})
	}
}

// TestExtractWorldview 测试世界观提取
func TestExtractWorldview(t *testing.T) {
	// 注意：世界观现在由上游节点提取，不再在 SaveOutlineTool 中处理
	t.Skip("世界观由上游提取节点处理，SaveOutlineTool 直接接收提取好的数据")
}

// TestExtractCultivationSystem 测试修行体系提取
func TestExtractCultivationSystem(t *testing.T) {
	// 注意：修行体系现在由上游节点提取，不再在 SaveOutlineTool 中处理
	t.Skip("修行体系由上游提取节点处理，SaveOutlineTool 直接接收提取好的数据")
}
