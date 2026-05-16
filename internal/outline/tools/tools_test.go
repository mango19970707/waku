package tools

import (
	"context"
	"encoding/json"
	"testing"
	"waku/internal/model"
	"waku/internal/outline/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func setupTestDBForTools(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        ":memory:",
	}, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Novel{}))
	return db
}

func TestSaveOutlineTool(t *testing.T) {
	db := setupTestDBForTools(t)
	repo := repository.NewRepository(db)
	tool, err := NewSaveOutlineTool(db, repo)
	require.NoError(t, err)

	input := SaveOutlineToolInput{
		Title:    "测试标题",
		Author:   "测试作者",
		Genre:    "玄幻",
		Synopsis: "测试简介",
		Outline:  `{"content":"测试大纲","stage":"修改","file_type":"outline"}`,
	}
	body, err := json.Marshal(input)
	require.NoError(t, err)

	resultJSON, err := tool.InvokableRun(context.Background(), string(body))
	require.NoError(t, err)

	var result SaveOutlineToolOutput
	require.NoError(t, json.Unmarshal([]byte(resultJSON), &result))
	assert.True(t, result.Success)
	assert.NotZero(t, result.NovelID)

	novel, err := repo.GetByID(result.NovelID)
	require.NoError(t, err)
	assert.Equal(t, input.Title, novel.Title)
	assert.Equal(t, input.Outline, novel.Outline)
}

func TestSaveOutlineToolValidation(t *testing.T) {
	db := setupTestDBForTools(t)
	repo := repository.NewRepository(db)
	tool, err := NewSaveOutlineTool(db, repo)
	require.NoError(t, err)

	input := SaveOutlineToolInput{
		Author:   "测试作者",
		Genre:    "玄幻",
		Synopsis: "测试简介",
		Outline:  "测试大纲",
	}
	body, err := json.Marshal(input)
	require.NoError(t, err)

	resultJSON, err := tool.InvokableRun(context.Background(), string(body))
	require.NoError(t, err)

	var result SaveOutlineToolOutput
	require.NoError(t, json.Unmarshal([]byte(resultJSON), &result))
	assert.False(t, result.Success)
	assert.Contains(t, result.Message, "小说标题不能为空")
}
