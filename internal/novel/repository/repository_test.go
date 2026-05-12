package repository

import (
	"testing"
	"waku/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	// 使用内存 SQLite 数据库进行测试
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "创建测试数据库失败")

	// 自动迁移表结构
	err = db.AutoMigrate(&model.Novel{})
	require.NoError(t, err, "迁移表结构失败")

	return db
}

// TestCreate 测试创建小说
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	novel := &model.Novel{
		Title:    "测试小说",
		Author:   "测试作者",
		Genre:    "玄幻",
		Synopsis: "这是一部测试小说",
	}

	err := repo.Create(novel)
	assert.NoError(t, err, "创建小说应该成功")
	assert.NotEmpty(t, novel.ID, "小说 ID 应该自动生成")
	assert.Equal(t, "draft", novel.Status, "默认状态应该是 draft")
}

// TestGetByID 测试根据 ID 获取小说
func TestGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novel := &model.Novel{
		Title:  "测试小说",
		Author: "测试作者",
		Genre:  "玄幻",
	}
	err := repo.Create(novel)
	require.NoError(t, err)

	// 测试获取
	found, err := repo.GetByID(novel.ID)
	assert.NoError(t, err, "获取小说应该成功")
	assert.Equal(t, novel.Title, found.Title, "标题应该匹配")
	assert.Equal(t, novel.Author, found.Author, "作者应该匹配")

	// 测试获取不存在的小说
	_, err = repo.GetByID("non-existent-id")
	assert.Error(t, err, "获取不存在的小说应该返回错误")
}

// TestGetAll 测试获取所有小说
func TestGetAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建多个测试数据
	novels := []*model.Novel{
		{Title: "小说1", Author: "作者1", Genre: "玄幻"},
		{Title: "小说2", Author: "作者2", Genre: "都市"},
		{Title: "小说3", Author: "作者3", Genre: "科幻"},
	}

	for _, novel := range novels {
		err := repo.Create(novel)
		require.NoError(t, err)
	}

	// 测试获取所有小说
	all, err := repo.GetAll(0, 0)
	assert.NoError(t, err, "获取所有小说应该成功")
	assert.Len(t, all, 3, "应该返回 3 部小说")

	// 测试分页
	page1, err := repo.GetAll(0, 2)
	assert.NoError(t, err)
	assert.Len(t, page1, 2, "第一页应该返回 2 部小说")

	page2, err := repo.GetAll(2, 2)
	assert.NoError(t, err)
	assert.Len(t, page2, 1, "第二页应该返回 1 部小说")
}

// TestGetByGenre 测试根据类型获取小说
func TestGetByGenre(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建不同类型的小说
	novels := []*model.Novel{
		{Title: "玄幻小说1", Author: "作者1", Genre: "玄幻"},
		{Title: "玄幻小说2", Author: "作者2", Genre: "玄幻"},
		{Title: "都市小说1", Author: "作者3", Genre: "都市"},
	}

	for _, novel := range novels {
		err := repo.Create(novel)
		require.NoError(t, err)
	}

	// 测试获取玄幻小说
	xuanhuan, err := repo.GetByGenre("玄幻", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, xuanhuan, 2, "应该返回 2 部玄幻小说")

	// 测试获取都市小说
	dushi, err := repo.GetByGenre("都市", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, dushi, 1, "应该返回 1 部都市小说")
}

// TestGetByStatus 测试根据状态获取小说
func TestGetByStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建不同状态的小说
	novels := []*model.Novel{
		{Title: "草稿小说", Author: "作者1", Genre: "玄幻", Status: "draft"},
		{Title: "连载小说", Author: "作者2", Genre: "都市", Status: "writing"},
		{Title: "完结小说", Author: "作者3", Genre: "科幻", Status: "completed"},
	}

	for _, novel := range novels {
		err := repo.Create(novel)
		require.NoError(t, err)
	}

	// 测试获取草稿状态的小说
	drafts, err := repo.GetByStatus("draft", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, drafts, 1, "应该返回 1 部草稿小说")

	// 测试获取连载状态的小说
	writing, err := repo.GetByStatus("writing", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, writing, 1, "应该返回 1 部连载小说")
}

// TestUpdate 测试更新小说
func TestUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novel := &model.Novel{
		Title:  "原标题",
		Author: "原作者",
		Genre:  "玄幻",
	}
	err := repo.Create(novel)
	require.NoError(t, err)

	// 更新小说信息
	novel.Title = "新标题"
	novel.Synopsis = "新简介"
	err = repo.Update(novel)
	assert.NoError(t, err, "更新小说应该成功")

	// 验证更新结果
	updated, err := repo.GetByID(novel.ID)
	assert.NoError(t, err)
	assert.Equal(t, "新标题", updated.Title, "标题应该已更新")
	assert.Equal(t, "新简介", updated.Synopsis, "简介应该已更新")
	assert.Equal(t, "原作者", updated.Author, "作者应该保持不变")
}

// TestUpdateStatus 测试更新小说状态
func TestUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novel := &model.Novel{
		Title:  "测试小说",
		Author: "测试作者",
		Genre:  "玄幻",
		Status: "draft",
	}
	err := repo.Create(novel)
	require.NoError(t, err)

	// 更新状态
	err = repo.UpdateStatus(novel.ID, "writing")
	assert.NoError(t, err, "更新状态应该成功")

	// 验证更新结果
	updated, err := repo.GetByID(novel.ID)
	assert.NoError(t, err)
	assert.Equal(t, "writing", updated.Status, "状态应该已更新为 writing")
}

// TestDelete 测试软删除小说
func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novel := &model.Novel{
		Title:  "测试小说",
		Author: "测试作者",
		Genre:  "玄幻",
	}
	err := repo.Create(novel)
	require.NoError(t, err)

	// 软删除
	err = repo.Delete(novel.ID)
	assert.NoError(t, err, "删除小说应该成功")

	// 验证删除结果（软删除后无法通过普通查询获取）
	_, err = repo.GetByID(novel.ID)
	assert.Error(t, err, "软删除后应该无法获取小说")

	// 验证数据库中仍然存在（只是标记为删除）
	var count int64
	db.Unscoped().Model(&model.Novel{}).Where("id = ?", novel.ID).Count(&count)
	assert.Equal(t, int64(1), count, "软删除后数据应该仍然存在")
}

// TestHardDelete 测试永久删除小说
func TestHardDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novel := &model.Novel{
		Title:  "测试小说",
		Author: "测试作者",
		Genre:  "玄幻",
	}
	err := repo.Create(novel)
	require.NoError(t, err)

	// 永久删除
	err = repo.HardDelete(novel.ID)
	assert.NoError(t, err, "永久删除小说应该成功")

	// 验证数据库中不存在
	var count int64
	db.Unscoped().Model(&model.Novel{}).Where("id = ?", novel.ID).Count(&count)
	assert.Equal(t, int64(0), count, "永久删除后数据应该不存在")
}

// TestCount 测试统计小说数量
func TestCount(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novels := []*model.Novel{
		{Title: "小说1", Author: "作者1", Genre: "玄幻"},
		{Title: "小说2", Author: "作者2", Genre: "玄幻"},
		{Title: "小说3", Author: "作者3", Genre: "都市"},
	}

	for _, novel := range novels {
		err := repo.Create(novel)
		require.NoError(t, err)
	}

	// 测试统计总数
	count, err := repo.Count()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count, "应该有 3 部小说")

	// 测试按类型统计
	xuanhuanCount, err := repo.CountByGenre("玄幻")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), xuanhuanCount, "应该有 2 部玄幻小说")

	dushiCount, err := repo.CountByGenre("都市")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), dushiCount, "应该有 1 部都市小说")
}

// TestSearch 测试搜索小说
func TestSearch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	// 创建测试数据
	novels := []*model.Novel{
		{Title: "龙族", Author: "江南", Genre: "玄幻"},
		{Title: "斗破苍穹", Author: "天蚕土豆", Genre: "玄幻"},
		{Title: "择天记", Author: "猫腻", Genre: "玄幻"},
	}

	for _, novel := range novels {
		err := repo.Create(novel)
		require.NoError(t, err)
	}

	// 测试按标题搜索
	results, err := repo.Search("龙", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "应该找到 1 部包含'龙'的小说")
	assert.Equal(t, "龙族", results[0].Title)

	// 测试按作者搜索
	results, err = repo.Search("江南", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 1, "应该找到 1 部江南的小说")

	// 测试搜索不存在的关键词
	results, err = repo.Search("不存在的关键词", 0, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 0, "应该找不到任何小说")
}
