package compose

import (
	"context"
	"encoding/json"
	"testing"

	"waku/internal/content/contracts"
	contentrepository "waku/internal/content/repository"
	"waku/internal/model"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newContentComposeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Novel{}, &model.Volume{}, &model.Chapter{}))
	return db
}

func TestRouteAfterReviewSkipsReviseWhenSuggestionsEmpty(t *testing.T) {
	state := &ContentWorkflowState{CurrentNode: "review_success", MaxRetries: 3, NeedRevise: false}

	next, err := routeAfterReview(context.Background(), state)

	require.NoError(t, err)
	require.Equal(t, "saveToDB", next)
}

func TestRouteAfterReviewUsesReviseWhenSuggestionsExist(t *testing.T) {
	state := &ContentWorkflowState{CurrentNode: "review_success", MaxRetries: 3, NeedRevise: true}

	next, err := routeAfterReview(context.Background(), state)

	require.NoError(t, err)
	require.Equal(t, "revise", next)
}

func TestSaveToDBNodeOnlyUpdatesContentFields(t *testing.T) {
	db := newContentComposeTestDB(t)
	repo := contentrepository.NewContentRepository(db)
	chapter := &model.Chapter{NovelID: 1, VolumeID: 2, ChapterNumber: 1, Title: "原题", Outline: "原大纲", Content: "旧正文", Status: "draft"}
	require.NoError(t, db.Create(chapter).Error)
	payload := contracts.ContentPayload{Chapter: contracts.ContentItem{ID: chapter.ID, NovelID: 1, VolumeID: 2, ChapterNumber: 1, Title: "被忽略标题", Content: "新正文", Status: "completed"}}
	data, err := jsonMarshal(payload)
	require.NoError(t, err)
	state := &ContentWorkflowState{NovelID: 1, VolumeID: 2, ChapterID: chapter.ID, PolishedContent: data}

	result, err := saveToDBNode(repo)(context.Background(), state)

	require.NoError(t, err)
	require.Equal(t, "save_success", result.CurrentNode)
	require.Equal(t, []uint{chapter.ID}, result.SavedChapterIDs)
	var updated model.Chapter
	require.NoError(t, db.First(&updated, chapter.ID).Error)
	require.Equal(t, "原题", updated.Title)
	require.Equal(t, "原大纲", updated.Outline)
	require.Equal(t, "新正文", updated.Content)
	require.Equal(t, "completed", updated.Status)
}

func TestPrepareNextNodeResetsStageFields(t *testing.T) {
	db := newContentComposeTestDB(t)
	repo := contentrepository.NewContentRepository(db)
	chapters := []*model.Chapter{
		{NovelID: 1, VolumeID: 2, ChapterNumber: 1, Title: "一", Outline: "一"},
		{NovelID: 1, VolumeID: 2, ChapterNumber: 2, Title: "二", Outline: "二"},
	}
	require.NoError(t, db.Create(&chapters).Error)
	state := &ContentWorkflowState{NovelID: 1, VolumeID: 2, ChapterID: chapters[0].ID, PendingChapterIDs: []uint{chapters[1].ID}, GeneratedContent: "old", PolishedContent: "old", ReviewedContent: "old", RevisedContent: "old", NeedRevise: true}

	result, err := prepareNextNode(repo)(context.Background(), state)

	require.NoError(t, err)
	require.Equal(t, "prepare_next_success", result.CurrentNode)
	require.Equal(t, chapters[1].ID, result.ChapterID)
	require.Empty(t, result.GeneratedContent)
	require.Empty(t, result.PolishedContent)
	require.False(t, result.NeedRevise)
	require.NotEmpty(t, result.TargetContent)
}

func jsonMarshal(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
