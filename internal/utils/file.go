package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"waku/internal/constant"
)

// SaveToFile 保存内容到文件（通用版本）
//
// 功能：将内容保存到指定的文件路径，自动创建目录
//
// 参数：
//   - author: 作者名称
//   - novelID: 小说ID（必填）
//   - novelName: 小说名称
//   - fileType: 文件类型（outline/volume/chapter）
//   - stage: 阶段名称（如"生成"、"扩写"、"润色"、"分卷"等）
//   - content: 要保存的内容
//
// 返回值：
//   - error: 保存失败时返回错误
//
// 文件路径格式：data/{作者名称}/{小说ID-名称}/{fileType}/{阶段}.md
func SaveToFile(author string, novelID uint, novelName, fileType, stage, content string) error {
	// 构建文件路径：data/{作者名称}/{小说ID-名称}/{fileType}/{阶段}.md
	dirName := fmt.Sprintf("%d-%s", novelID, novelName)
	dir := filepath.Join(constant.DataDir, author, dirName, fileType)

	// 创建目录（如果不存在）
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(dir, stage+constant.FileExtension)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// LoadFromFile 从文件加载内容（用于断点续传，通用版本）
//
// 功能：从指定路径加载文件内容
//
// 参数：
//   - author: 作者名称
//   - novelID: 小说ID（必填）
//   - novelName: 小说名称
//   - fileType: 文件类型（outline/volume/chapter）
//   - stage: 阶段名称（如"生成"、"扩写"、"润色"、"分卷"等）
//
// 返回值：
//   - string: 文件内容，文件不存在时返回空字符串
//   - error: 读取失败时返回错误（文件不存在不算错误）
func LoadFromFile(author string, novelID uint, novelName, fileType, stage string) (string, error) {
	dirName := fmt.Sprintf("%d-%s", novelID, novelName)
	filePath := filepath.Join(constant.DataDir, author, dirName, fileType, stage+constant.FileExtension)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空字符串
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	return string(content), nil
}

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
