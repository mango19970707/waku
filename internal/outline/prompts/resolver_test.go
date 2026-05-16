package prompts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveOutlinePrompt_Xuanhuan(t *testing.T) {
	prompt, err := ResolveOutlinePrompt(StageGenerate, "玄幻")
	require.NoError(t, err)
	require.Equal(t, XuanhuanOutlineGeneratorPrompt, prompt)
}

func TestResolveOutlinePrompt_XiuzhenUsesSamePrompt(t *testing.T) {
	prompt, err := ResolveOutlinePrompt(StageReview, "修真")
	require.NoError(t, err)
	require.Equal(t, XuanhuanOutlineReviewerPrompt, prompt)
}

func TestResolveOutlinePrompt_UnsupportedGenre(t *testing.T) {
	_, err := ResolveOutlinePrompt(StagePolish, "都市")
	require.Error(t, err)
	require.ErrorContains(t, err, "暂未实现")
}

func TestResolveOutlinePrompt_UnknownStage(t *testing.T) {
	_, err := ResolveOutlinePrompt("unknown", "玄幻")
	require.Error(t, err)
	require.ErrorContains(t, err, "未知")
}
