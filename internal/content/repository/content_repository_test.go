package repository

import (
	"testing"

	"waku/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newContentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Novel{}, &model.Volume{}, &model.Chapter{}))
	return db
}

func TestUpdateContentOnlyChangesContentFields(t *testing.T) {
	db := newContentTestDB(t)
	repo := NewContentRepository(db)
	chapter := &model.Chapter{NovelID: 1, VolumeID: 2, ChapterNumber: 1, Title: "原题", Outline: "原大纲", Content: "旧正文", Status: "draft"}
	require.NoError(t, db.Create(chapter).Error)

	require.NoError(t, repo.UpdateContent(chapter.ID, "新正文", 3, "writing"))

	var updated model.Chapter
	require.NoError(t, db.First(&updated, chapter.ID).Error)
	require.Equal(t, "原题", updated.Title)
	require.Equal(t, "原大纲", updated.Outline)
	require.Equal(t, "新正文", updated.Content)
	require.Equal(t, 3, updated.WordCount)
	require.Equal(t, "writing", updated.Status)
}

func TestGetEmptyContentChaptersAndNeighbors(t *testing.T) {
	db := newContentTestDB(t)
	repo := NewContentRepository(db)
	chapters := []*model.Chapter{
		{NovelID: 1, VolumeID: 2, ChapterNumber: 1, Title: "一", Outline: "一", Content: "已有正文"},
		{NovelID: 1, VolumeID: 2, ChapterNumber: 2, Title: "二", Outline: "二"},
		{NovelID: 1, VolumeID: 2, ChapterNumber: 3, Title: "三", Outline: "三"},
	}
	require.NoError(t, db.Create(&chapters).Error)

	empty, err := repo.GetEmptyContentChapters(2)
	require.NoError(t, err)
	require.Len(t, empty, 2)
	require.Equal(t, 2, empty[0].ChapterNumber)

	prev, err := repo.GetPreviousChapter(chapters[1].ID)
	require.NoError(t, err)
	require.Equal(t, 1, prev.ChapterNumber)

	next, err := repo.GetNextChapter(chapters[1].ID)
	require.NoError(t, err)
	require.Equal(t, 3, next.ChapterNumber)
}
