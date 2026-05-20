package compose

import (
	"encoding/json"
	"testing"

	"waku/internal/chapter/contracts"

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
