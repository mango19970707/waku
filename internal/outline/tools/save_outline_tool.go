package tools

import (
	"context"
	"fmt"
	"waku/internal/model"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"gorm.io/gorm"
)

// SaveOutlineTool 保存大纲到数据库的工具。
type SaveOutlineTool struct {
	repo Repository
}

// Repository 是保存大纲工具依赖的仓储接口。
type Repository interface {
	Create(novel *model.Novel) error
	CheckDuplicateByAuthorAndTitle(author, title string) (bool, error)
}

// SaveOutlineToolInput 是保存大纲工具的输入。
type SaveOutlineToolInput struct {
	Title    string `json:"title" jsonschema:"required" jsonschema_description:"小说标题"`
	Author   string `json:"author" jsonschema:"required" jsonschema_description:"作者名称"`
	Genre    string `json:"genre" jsonschema:"required" jsonschema_description:"小说类型"`
	Synopsis string `json:"synopsis" jsonschema:"required" jsonschema_description:"小说简介"`
	Outline  string `json:"outline" jsonschema:"required" jsonschema_description:"完整大纲内容，建议为 JSON 字符串"`
}

// SaveOutlineToolOutput 是保存大纲工具的输出。
type SaveOutlineToolOutput struct {
	Success bool   `json:"success"`
	NovelID uint   `json:"novel_id"`
	Message string `json:"message"`
}

// NewSaveOutlineTool 创建保存大纲工具。
func NewSaveOutlineTool(_ *gorm.DB, repo Repository) (tool.InvokableTool, error) {
	saveTool := &SaveOutlineTool{repo: repo}

	desc := `保存小说大纲到数据库。

使用场景：
- 上游工作流已经产出完整标题、简介和大纲内容，需要入库时调用

参数说明：
- title: 小说标题
- author: 作者名称
- genre: 小说类型
- synopsis: 小说简介
- outline: 完整大纲内容，建议传入 JSON 字符串

返回值：
- success: 是否保存成功
- novel_id: 保存后的小说 ID
- message: 提示信息`

	return toolutils.InferTool(
		"save_outline_to_database",
		desc,
		saveTool.Invoke,
	)
}

// Invoke 执行保存大纲操作。
func (t *SaveOutlineTool) Invoke(ctx context.Context, input SaveOutlineToolInput) (SaveOutlineToolOutput, error) {
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

	duplicated, err := t.repo.CheckDuplicateByAuthorAndTitle(input.Author, input.Title)
	if err != nil {
		return SaveOutlineToolOutput{Success: false, Message: fmt.Sprintf("检查重名失败: %v", err)}, nil
	}
	if duplicated {
		return SaveOutlineToolOutput{Success: false, Message: "该作者已存在同名小说"}, nil
	}

	novel := &model.Novel{
		Title:    input.Title,
		Author:   input.Author,
		Genre:    input.Genre,
		Synopsis: input.Synopsis,
		Outline:  input.Outline,
		Status:   "draft",
	}

	if err := t.repo.Create(novel); err != nil {
		return SaveOutlineToolOutput{Success: false, Message: fmt.Sprintf("保存失败: %v", err)}, nil
	}

	return SaveOutlineToolOutput{
		Success: true,
		NovelID: novel.ID,
		Message: fmt.Sprintf("小说《%s》已成功保存到数据库，ID: %d", novel.Title, novel.ID),
	}, nil
}
