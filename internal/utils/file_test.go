package utils

import (
	"os"
	"path/filepath"
	"testing"
	"waku/internal/constant"
)

func TestSaveAndLoadFile(t *testing.T) {
	// 测试数据
	author := "test_author"
	fileType := constant.FileTypeOutline
	stage := constant.StageGenerate
	content := "这是测试内容\n包含多行文本"

	// 清理测试数据
	defer func() {
		testDir := filepath.Join(constant.DataDir, author)
		os.RemoveAll(testDir)
	}()

	// 测试保存
	err := SaveToFile(author, fileType, stage, content)
	if err != nil {
		t.Fatalf("保存文件失败: %v", err)
	}

	// 验证文件是否存在
	filePath := filepath.Join(constant.DataDir, author, fileType, stage+constant.FileExtension)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("文件未创建: %s", filePath)
	}

	// 测试加载
	loadedContent, err := LoadFromFile(author, fileType, stage)
	if err != nil {
		t.Fatalf("加载文件失败: %v", err)
	}

	// 验证内容是否一致
	if loadedContent != content {
		t.Fatalf("内容不一致\n期望: %s\n实际: %s", content, loadedContent)
	}

	t.Log("✓ 文件保存和加载测试通过")
}

func TestLoadNonExistentFile(t *testing.T) {
	// 测试加载不存在的文件
	content, err := LoadFromFile("nonexistent", "outline", "test")
	if err != nil {
		t.Fatalf("加载不存在的文件应返回空字符串，但返回错误: %v", err)
	}
	if content != "" {
		t.Fatalf("加载不存在的文件应返回空字符串，但返回: %s", content)
	}

	t.Log("✓ 加载不存在文件测试通过")
}

func TestSaveAndLoadMultipleStages(t *testing.T) {
	author := "test_author_multi"
	fileType := constant.FileTypeOutline

	// 清理测试数据
	defer func() {
		testDir := filepath.Join(constant.DataDir, author)
		os.RemoveAll(testDir)
	}()

	// 测试多个阶段
	stages := []struct {
		stage   string
		content string
	}{
		{constant.StageGenerate, "生成的大纲内容"},
		{constant.StageExpand, "扩写的大纲内容"},
		{constant.StagePolish, "润色的大纲内容"},
		{constant.StageRevise, "修改的大纲内容"},
	}

	// 保存所有阶段
	for _, s := range stages {
		if err := SaveToFile(author, fileType, s.stage, s.content); err != nil {
			t.Fatalf("保存 %s 阶段失败: %v", s.stage, err)
		}
	}

	// 加载并验证所有阶段
	for _, s := range stages {
		content, err := LoadFromFile(author, fileType, s.stage)
		if err != nil {
			t.Fatalf("加载 %s 阶段失败: %v", s.stage, err)
		}
		if content != s.content {
			t.Fatalf("%s 阶段内容不一致\n期望: %s\n实际: %s", s.stage, s.content, content)
		}
	}

	t.Log("✓ 多阶段文件保存和加载测试通过")
}
