package contracts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeContentPayload_ComputesWordCountAndDefaultsStatus(t *testing.T) {
	raw := `{"chapter":{"id":15,"novel_id":1,"volume_id":2,"chapter_number":3,"title":"夜入丹阁","content":"夜色沉入丹阁。\n苏婉执灯。"}}`

	normalized, payload, err := NormalizeContentPayload(raw)

	require.NoError(t, err)
	require.Equal(t, uint(15), payload.Chapter.ID)
	require.Equal(t, "writing", payload.Chapter.Status)
	require.Equal(t, CountContentWords("夜色沉入丹阁。\n苏婉执灯。"), payload.Chapter.WordCount)
	require.True(t, json.Valid([]byte(normalized)))
}

func TestNormalizeContentPayload_RejectsEmptyContent(t *testing.T) {
	_, _, err := NormalizeContentPayload(`{"chapter":{"id":1,"novel_id":1,"volume_id":1,"chapter_number":1,"title":"空章","content":"   "}}`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "content 不能为空")
}

func TestNormalizeContentReviewPayload_AllowsEmptySuggestions(t *testing.T) {
	normalized, payload, err := NormalizeContentReviewPayload(`{"suggestions":[]}`)

	require.NoError(t, err)
	require.Empty(t, payload.Suggestions)
	require.True(t, strings.Contains(normalized, "suggestions"))
}

func TestNormalizeContentReviewPayload_RejectsInvalidTarget(t *testing.T) {
	_, _, err := NormalizeContentReviewPayload(`{"suggestions":[{"suggestion":"补充动作","target":"bad","id":1}]}`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "target 无效")
}
