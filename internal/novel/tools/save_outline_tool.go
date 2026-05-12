package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
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
	Outline string `json:"outline" jsonschema:"required" jsonschema_description:"完整的大纲内容（包含世界观、修行体系等）"`
}

// SaveOutlineToolOutput 保存大纲工具的输出
type SaveOutlineToolOutput struct {
	// Success 是否成功
	Success bool `json:"success"`
	// NovelID 小说ID
	NovelID string `json:"novel_id"`
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
该工具会从大纲内容中提取世界观和修行体系信息，并将完整的小说信息保存到数据库中。

使用场景：
- 当大纲润色完成后，需要保存到数据库
- 需要持久化小说的基本信息和大纲内容

参数说明：
- title: 小说标题（必填）
- author: 作者名称（必填）
- genre: 小说类型，如"玄幻"、"都市"、"科幻"等（必填）
- synopsis: 小说简介，简短描述故事梗概（必填）
- outline: 完整的大纲内容，包含世界观、修行体系、剧情等所有信息（必填）

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

	// 从大纲中提取世界观和修行体系
	worldview := t.extractWorldview(input.Outline)
	cultivationSystem := t.extractCultivationSystem(input.Outline)

	// 创建小说对象
	novel := &model.Novel{
		Title:             input.Title,
		Author:            input.Author,
		Genre:             input.Genre,
		Synopsis:          input.Synopsis,
		Worldview:         worldview,
		CultivationSystem: cultivationSystem,
		Status:            "draft",
	}

	// 保存到数据库
	if err := t.repo.Create(novel); err != nil {
		return SaveOutlineToolOutput{
			Success: false,
			Message: fmt.Sprintf("保存失败: %v", err),
		}, nil
	}

	message := fmt.Sprintf("小说《%s》已成功保存到数据库，ID: %s", novel.Title, novel.ID)
	return SaveOutlineToolOutput{
		Success: true,
		NovelID: novel.ID,
		Message: message,
	}, nil
}

// extractWorldview 从大纲中提取世界观内容
func (t *SaveOutlineTool) extractWorldview(outline string) string {
	// 尝试匹配世界观相关的章节
	patterns := []string{
		`(?s)##?\s*世界观\s*\n(.*?)(?:\n##|\z)`,
		`(?s)##?\s*2\.\s*世界观\s*\n(.*?)(?:\n##|\z)`,
		`(?s)世界观[:：]\s*\n(.*?)(?:\n##|\z)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(outline); len(matches) > 1 {
			content := strings.TrimSpace(matches[1])
			if content != "" {
				return content
			}
		}
	}

	// 如果没有找到专门的世界观章节，尝试提取相关内容
	worldviewKeywords := []string{"世界构成", "天地规则", "种族分布", "历史脉络", "大陆", "位面", "界域"}
	var worldviewParts []string

	lines := strings.Split(outline, "\n")
	inWorldviewSection := false
	var currentSection strings.Builder

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 检查是否进入世界观相关章节
		for _, keyword := range worldviewKeywords {
			if strings.Contains(trimmedLine, keyword) {
				inWorldviewSection = true
				break
			}
		}

		// 如果在世界观章节中，收集内容
		if inWorldviewSection {
			if strings.HasPrefix(trimmedLine, "#") && !containsAny(trimmedLine, worldviewKeywords) {
				// 遇到新的章节标题，结束当前章节
				if currentSection.Len() > 0 {
					worldviewParts = append(worldviewParts, currentSection.String())
					currentSection.Reset()
				}
				inWorldviewSection = false
			} else {
				currentSection.WriteString(line)
				currentSection.WriteString("\n")
			}
		}
	}

	// 添加最后一个章节
	if currentSection.Len() > 0 {
		worldviewParts = append(worldviewParts, currentSection.String())
	}

	if len(worldviewParts) > 0 {
		return strings.TrimSpace(strings.Join(worldviewParts, "\n\n"))
	}

	return "暂无详细世界观设定"
}

// extractCultivationSystem 从大纲中提取修行体系内容
func (t *SaveOutlineTool) extractCultivationSystem(outline string) string {
	// 尝试匹配修行体系相关的章节
	patterns := []string{
		`(?s)##?\s*修行体系\s*\n(.*?)(?:\n##|\z)`,
		`(?s)##?\s*3\.\s*修行体系\s*\n(.*?)(?:\n##|\z)`,
		`(?s)修行体系[:：]\s*\n(.*?)(?:\n##|\z)`,
		`(?s)##?\s*修炼体系\s*\n(.*?)(?:\n##|\z)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(outline); len(matches) > 1 {
			content := strings.TrimSpace(matches[1])
			if content != "" {
				return content
			}
		}
	}

	// 如果没有找到专门的修行体系章节，尝试提取相关内容
	cultivationKeywords := []string{"境界划分", "修炼方式", "力量体系", "特殊能力", "突破条件", "炼气", "筑基", "金丹"}
	var cultivationParts []string

	lines := strings.Split(outline, "\n")
	inCultivationSection := false
	var currentSection strings.Builder

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// 检查是否进入修行体系相关章节
		for _, keyword := range cultivationKeywords {
			if strings.Contains(trimmedLine, keyword) {
				inCultivationSection = true
				break
			}
		}

		// 如果在修行体系章节中，收集内容
		if inCultivationSection {
			if strings.HasPrefix(trimmedLine, "#") && !containsAny(trimmedLine, cultivationKeywords) {
				// 遇到新的章节标题，结束当前章节
				if currentSection.Len() > 0 {
					cultivationParts = append(cultivationParts, currentSection.String())
					currentSection.Reset()
				}
				inCultivationSection = false
			} else {
				currentSection.WriteString(line)
				currentSection.WriteString("\n")
			}
		}
	}

	// 添加最后一个章节
	if currentSection.Len() > 0 {
		cultivationParts = append(cultivationParts, currentSection.String())
	}

	if len(cultivationParts) > 0 {
		return strings.TrimSpace(strings.Join(cultivationParts, "\n\n"))
	}

	return "暂无详细修行体系设定"
}

// containsAny 检查字符串是否包含任意一个关键词
func containsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}
