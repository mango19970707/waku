package tools

import (
	"context"
	"fmt"
	"waku/internal/model"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/gorm"
)

// SaveOutlineTool 保存大纲到数据库的工具
type SaveOutlineTool struct {
	db   *gorm.DB
	repo Repository
}

// Repository 小说仓储接口
type Repository interface {
	Create(novel *model.Novel) error
}

// SaveOutlineToolInput 保存大纲工具的输入参数
type SaveOutlineToolInput struct {
	// Title 小说标题
	Title string `json:"title" jsonschema:"required" jsonschema_description:"小说标题"`
	// Author 作者名称
	Author string `json:"author" jsonschema:"required" jsonschema_description:"作者名称"`
	// Genre 小说类型
	Genre string `json:"genre" jsonschema:"required" jsonschema_description:"小说类型（如：玄幻、都市、科幻等）"`
	// Synopsis 小说简介
	Synopsis string `json:"synopsis" jsonschema:"required" jsonschema_description:"小说简介"`
	// Outline 完整的大纲内容
	Outline string `json:"outline" jsonschema:"required" jsonschema_description:"完整的大纲内容"`
	// Worldview 世界观设定
	Worldview string `json:"worldview" jsonschema:"required" jsonschema_description:"世界观设定"`
	// CultivationSystem 修行体系
	CultivationSystem string `json:"cultivation_system" jsonschema:"required" jsonschema_description:"修行体系"`
}

// SaveOutlineToolOutput 保存大纲工具的输出
type SaveOutlineToolOutput struct {
	// Success 是否成功
	Success bool `json:"success"`
	// NovelID 小说ID
	NovelID uint `json:"novel_id"`
	// Message 提示信息
	Message string `json:"message"`
}

// NewSaveOutlineTool 创建保存大纲工具
//
// 参数：
//   - db: 数据库连接
//   - repo: 小说仓储实现
//
// 返回值：
//   - tool.InvokableTool: 可调用的工具
func NewSaveOutlineTool(db *gorm.DB, repo Repository) (tool.InvokableTool, error) {
	saveTool := &SaveOutlineTool{
		db:   db,
		repo: repo,
	}

	toolDesc := `保存小说大纲到数据库。
该工具将完整的小说信息（包括世界观和修行体系）保存到数据库中。

使用场景：
- 当大纲润色完成后，需要保存到数据库
- 需要持久化小说的基本信息和大纲内容

参数说明：
- title: 小说标题（必填）
- author: 作者名称（必填）
- genre: 小说类型，如"玄幻"、"都市"、"科幻"等（必填）
- synopsis: 小说简介，简短描述故事梗概（必填）
- outline: 完整的大纲内容（必填）
- worldview: 世界观设定（必填）
- cultivation_system: 修行体系（必填）

返回值：
- success: 是否保存成功
- novel_id: 保存后的小说ID
- message: 提示信息`

	return utils.InferTool(
		"save_outline_to_database",
		toolDesc,
		saveTool.Invoke,
	)
}

// Invoke 执行保存大纲操作
func (t *SaveOutlineTool) Invoke(ctx context.Context, input SaveOutlineToolInput) (SaveOutlineToolOutput, error) {
	// 验证必填参数
	if input.Title == "" {
		return SaveOutlineToolOutput{Success: false, Message: "小说标题不能为空"}, nil
	}
	if input.Author == "" {
		return SaveOutlineToolOutput{Success: false, Message: "作者名称不能为空"}, nil
	}
	if input.Genre == "" {
		return SaveOutlineToolOutput{Success: false, Message: "小说类型不能为空"}, nil
	}
	if input.Synopsis == "" {
		return SaveOutlineToolOutput{Success: false, Message: "小说简介不能为空"}, nil
	}
	if input.Outline == "" {
		return SaveOutlineToolOutput{Success: false, Message: "大纲内容不能为空"}, nil
	}
	if input.Worldview == "" {
		return SaveOutlineToolOutput{Success: false, Message: "世界观不能为空"}, nil
	}
	if input.CultivationSystem == "" {
		return SaveOutlineToolOutput{Success: false, Message: "修行体系不能为空"}, nil
	}

	// 创建小说对象
	novel := &model.Novel{
		Title:             input.Title,
		Author:            input.Author,
		Genre:             input.Genre,
		Synopsis:          input.Synopsis,
		Worldview:         input.Worldview,
		CultivationSystem: input.CultivationSystem,
		Status:            "draft",
	}

	// 保存到数据库
	if err := t.repo.Create(novel); err != nil {
		return SaveOutlineToolOutput{
			Success: false,
			Message: fmt.Sprintf("保存失败: %v", err),
		}, nil
	}

	message := fmt.Sprintf("小说《%s》已成功保存到数据库，ID: %d", novel.Title, novel.ID)
	return SaveOutlineToolOutput{
		Success: true,
		NovelID: novel.ID,
		Message: message,
	}, nil
}
