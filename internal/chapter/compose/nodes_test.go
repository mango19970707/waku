package compose

import (
	"encoding/json"
	"testing"

	"waku/internal/chapter/contracts"
	"waku/internal/model"

	"github.com/stretchr/testify/require"
)

func TestMergeRevisedChapters_保留未修改章节并覆盖补丁(t *testing.T) {
	base := chapterPayloadJSON(t, []contracts.ChapterItem{
		newChapterItem(1, 1, "第一章", "原始一"),
		newChapterItem(2, 2, "第二章", "原始二"),
		newChapterItem(3, 3, "第三章", "原始三"),
		newChapterItem(4, 4, "第四章", "原始四"),
	})
	patch := chapterPayloadJSON(t, []contracts.ChapterItem{
		newChapterItem(3, 3, "第三章改", "修改三"),
	})

	merged, payload, err := mergeRevisedChapters(base, patch)

	require.NoError(t, err)
	require.JSONEq(t, chapterPayloadJSON(t, []contracts.ChapterItem{
		newChapterItem(1, 1, "第一章", "原始一"),
		newChapterItem(2, 2, "第二章", "原始二"),
		newChapterItem(3, 3, "第三章改", "修改三"),
		newChapterItem(4, 4, "第四章", "原始四"),
	}), merged)
	require.Len(t, payload.Chapters, 4)
}

func TestMergeRevisedChapters_拒绝未知章节补丁(t *testing.T) {
	base := chapterPayloadJSON(t, []contracts.ChapterItem{
		newChapterItem(1, 1, "第一章", "原始一"),
	})
	patch := chapterPayloadJSON(t, []contracts.ChapterItem{
		newChapterItem(0, 2, "第二章", "新增二"),
	})

	_, _, err := mergeRevisedChapters(base, patch)

	require.ErrorContains(t, err, "未匹配到原章节")
}

func TestSaveChaptersToDB_完整划分覆盖旧章节(t *testing.T) {
	repo := &fakeChapterRepository{}
	items := []contracts.ChapterItem{
		newChapterItem(10, 1, "新第一章", "新大纲一"),
		newChapterItem(11, 2, "新第二章", "新大纲二"),
	}

	ids, err := saveChaptersToDB(repo, items, 1, 2, true)

	require.NoError(t, err)
	require.Equal(t, []uint{101, 102}, ids)
	require.Equal(t, []uint{2}, repo.deletedVolumeIDs)
	require.Len(t, repo.created, 2)
	require.Zero(t, repo.created[0].ID)
	require.Zero(t, repo.created[1].ID)
	require.Equal(t, 1, repo.created[0].ChapterNumber)
	require.Equal(t, 2, repo.created[1].ChapterNumber)
	require.Empty(t, repo.updated)
}

func TestSaveChaptersToDB_非覆盖模式按ID更新无ID创建(t *testing.T) {
	repo := &fakeChapterRepository{}
	items := []contracts.ChapterItem{
		newChapterItem(10, 1, "旧第一章改", "修改大纲一"),
		newChapterItem(0, 2, "新第二章", "新大纲二"),
	}

	ids, err := saveChaptersToDB(repo, items, 1, 2, false)

	require.NoError(t, err)
	require.Equal(t, []uint{10, 101}, ids)
	require.Empty(t, repo.deletedVolumeIDs)
	require.Len(t, repo.updated, 1)
	require.Equal(t, uint(10), repo.updated[0].ID)
	require.Len(t, repo.created, 1)
	require.Zero(t, repo.created[0].ID)
}

type fakeChapterRepository struct {
	created          []*model.Chapter
	updated          []*model.Chapter
	deletedVolumeIDs []uint
}

func (r *fakeChapterRepository) CreateBatch(chapters []*model.Chapter) error {
	for _, chapter := range chapters {
		chapterSnapshot := *chapter
		r.created = append(r.created, &chapterSnapshot)
	}
	for i, chapter := range chapters {
		if chapter.ID == 0 {
			chapter.ID = uint(101 + i)
		}
	}
	return nil
}

func (r *fakeChapterRepository) GetByID(id uint) (*model.Chapter, error) {
	return nil, nil
}

func (r *fakeChapterRepository) GetByVolumeID(volumeID uint) ([]*model.Chapter, error) {
	return nil, nil
}

func (r *fakeChapterRepository) GetByNovelID(novelID uint) ([]*model.Chapter, error) {
	return nil, nil
}

func (r *fakeChapterRepository) Update(chapter *model.Chapter) error {
	r.updated = append(r.updated, chapter)
	return nil
}

func (r *fakeChapterRepository) Delete(id uint) error {
	return nil
}

func (r *fakeChapterRepository) DeleteByVolumeID(volumeID uint) error {
	r.deletedVolumeIDs = append(r.deletedVolumeIDs, volumeID)
	return nil
}

func (r *fakeChapterRepository) DeleteByNovelID(novelID uint) error {
	return nil
}

func (r *fakeChapterRepository) GetMaxChapterNumber(volumeID uint) (int, error) {
	return 0, nil
}

func newChapterItem(id uint, chapterNumber int, title, outline string) contracts.ChapterItem {
	return contracts.ChapterItem{
		ID:                id,
		ChapterNumber:     chapterNumber,
		Title:             title,
		Synopsis:          title + "简介",
		Outline:           outline,
		WordCountEstimate: 3000,
		KeyEvents:         []string{"事件"},
		KeyCharacters:     []string{"主角"},
		PlotlineProgress:  map[string]string{"1": "推进"},
		Status:            "draft",
	}
}

func chapterPayloadJSON(t *testing.T, chapters []contracts.ChapterItem) string {
	t.Helper()
	data, err := json.Marshal(contracts.ChapterPayload{Chapters: chapters})
	require.NoError(t, err)
	return string(data)
}
