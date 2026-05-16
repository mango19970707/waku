package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"waku/internal/constant"
)

// SaveToFile 保存内容到文件（通用版本）
//
// 功能：将内容保存到指定的文件路径，自动创建目录，内容以 JSON 格式保存
//
// 参数：
//   - author: 作者名称
//   - fileType: 文件类型（outline/plotline/volume/chapter）
//   - stage: 阶段名称（如"生成"、"扩写"、"润色"、"分卷"等）
//   - content: 要保存的内容
//
// 返回值：
//   - error: 保存失败时返回错误
//
// 文件路径格式：data/{作者名称}/{fileType}/{阶段}.json
// 文件内容格式：{"content": "实际内容", "stage": "阶段名称", "file_type": "文件类型"}
func SaveToFile(author, fileType, stage, content string) error {
	// 构建文件路径：data/{作者名称}/{fileType}/{阶段}.json
	dir := filepath.Join(constant.DataDir, author, fileType)

	// 创建目录（如果不存在）
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(dir, stage+constant.FileExtension)

	// 构建 JSON 数据结构
	data := map[string]interface{}{
		"content":   content,
		"stage":     stage,
		"file_type": fileType,
	}

	// 序列化为 JSON（带缩进，便于阅读）
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// LoadFromFile 从文件加载内容（用于断点续传，通用版本）
//
// 功能：从指定路径加载文件内容，支持 JSON 格式和纯文本格式（向后兼容）
//
// 参数：
//   - author: 作者名称
//   - fileType: 文件类型（outline/plotline/volume/chapter）
//   - stage: 阶段名称（如"生成"、"扩写"、"润色"、"分卷"等）
//
// 返回值：
//   - string: 文件内容，文件不存在时返回空字符串
//   - error: 读取失败时返回错误（文件不存在不算错误）
func LoadFromFile(author, fileType, stage string) (string, error) {
	filePath := filepath.Join(constant.DataDir, author, fileType, stage+constant.FileExtension)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空字符串
		}
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 尝试解析为 JSON
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err == nil {
		// 成功解析为 JSON，提取 content 字段
		if contentStr, ok := data["content"].(string); ok {
			return contentStr, nil
		}
		return "", fmt.Errorf("JSON 文件缺少 content 字段")
	}

	// 如果不是 JSON 格式，当作纯文本返回（向后兼容旧的 .md 文件）
	return string(content), nil
}

// Min 返回两个整数中的较小值
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
