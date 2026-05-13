package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// SaveVolumeFile 保存卷相关文件
//
// 功能：将卷内容保存到指定的文件路径，自动创建目录
//
// 参数：
//   - novelID: 小说ID（0表示临时文件）
//   - novelName: 小说名称
//   - stage: 阶段名称（如"分卷"、"第1卷-章节"）
//   - content: 要保存的内容
//
// 返回值：
//   - error: 保存失败时返回错误
//
// 文件路径格式：data/content/{小说ID-名称}/volume/{阶段}.md
func SaveVolumeFile(novelID uint, novelName, stage, content string) error {
	// 构建文件路径：data/content/{小说ID-名称}/volume/{阶段}.md
	var dirName string
	if novelID > 0 {
		dirName = fmt.Sprintf("%d-%s", novelID, novelName)
	} else {
		// 如果没有ID，使用临时名称
		dirName = fmt.Sprintf("temp-%s", novelName)
	}

	dir := filepath.Join("data", "content", dirName, "volume")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(dir, fmt.Sprintf("%s.md", stage))

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// LoadVolumeFile 从文件加载卷内容（用于断点续传）
//
// 功能：从指定路径加载文件内容
//
// 参数：
//   - novelID: 小说ID（0表示临时文件）
//   - novelName: 小说名称
//   - stage: 阶段名称（如"分卷"、"第1卷-章节"）
//
// 返回值：
//   - string: 文件内容，文件不存在时返回空字符串
//   - error: 读取失败时返回错误（文件不存在不算错误）
func LoadVolumeFile(novelID uint, novelName, stage string) (string, error) {
	var dirName string
	if novelID > 0 {
		dirName = fmt.Sprintf("%d-%s", novelID, novelName)
	} else {
		dirName = fmt.Sprintf("temp-%s", novelName)
	}

	filePath := filepath.Join("data", "content", dirName, "volume", fmt.Sprintf("%s.md", stage))

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空字符串
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	return string(content), nil
}
