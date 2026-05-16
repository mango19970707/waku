package contracts

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeNovelPayload_XuanhuanSuccess(t *testing.T) {
	raw := `{
		"title": "诸天剑主",
		"synopsis": "少年踏上修行路。",
		"outline": "{\"outline\":\"第一卷少年出山\",\"worldview\":\"诸天万界并立\",\"cultivation_system\":\"炼气、筑基、金丹\"}"
	}`

	normalized, payload, err := NormalizeNovelPayload(raw, "玄幻")
	require.NoError(t, err)
	require.NotEmpty(t, normalized)
	require.Equal(t, "诸天剑主", payload.Title)
	require.Equal(t, "少年踏上修行路。", payload.Synopsis)

	var outline map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload.Outline), &outline))
	require.Equal(t, "诸天万界并立", outline["worldview"])
	require.Equal(t, "炼气、筑基、金丹", outline["cultivation_system"])
}

func TestNormalizeNovelPayload_XuanhuanMissingWorldview(t *testing.T) {
	raw := `{
		"title": "诸天剑主",
		"synopsis": "少年踏上修行路。",
		"outline": "{\"outline\":\"第一卷少年出山\",\"cultivation_system\":\"炼气、筑基、金丹\"}"
	}`

	_, _, err := NormalizeNovelPayload(raw, "玄幻")
	require.Error(t, err)
	require.ErrorContains(t, err, "worldview")
}

func TestNormalizeNovelPayload_OtherGenreUsesTODOBranch(t *testing.T) {
	raw := `{
		"title": "赛博迷城",
		"synopsis": "黑客追查失控 AI。",
		"outline": "{\"outline\":\"第一幕潜入巨企\"}"
	}`

	normalized, payload, err := NormalizeNovelPayload(raw, "科幻")
	require.NoError(t, err)
	require.NotEmpty(t, normalized)
	require.Equal(t, "赛博迷城", payload.Title)
}
