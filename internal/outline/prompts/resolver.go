package prompts

import (
	"fmt"
	"strings"
)

const (
	StageGenerate = "generate"
	StageExpand   = "expand"
	StagePolish   = "polish"
	StageReview   = "review"
	StageRevise   = "revise"
)

// ResolveOutlinePrompt 按阶段和小说类型返回对应提示词。
func ResolveOutlinePrompt(stage, genre string) (string, error) {
	switch {
	case isXuanhuanOrXiuzhenGenre(genre):
		return resolveXuanhuanPrompt(stage)
	default:
		return "", fmt.Errorf("小说类型 %q 的 %s 提示词暂未实现", genre, stage)
	}
}

func resolveXuanhuanPrompt(stage string) (string, error) {
	switch stage {
	case StageGenerate:
		return XuanhuanOutlineGeneratorPrompt, nil
	case StageExpand:
		return XuanhuanOutlineExpanderPrompt, nil
	case StagePolish:
		return XuanhuanOutlinePolisherPrompt, nil
	case StageReview:
		return XuanhuanOutlineReviewerPrompt, nil
	case StageRevise:
		return XuanhuanOutlineReviserPrompt, nil
	default:
		return "", fmt.Errorf("未知的大纲提示词阶段: %s", stage)
	}
}

func isXuanhuanOrXiuzhenGenre(genre string) bool {
	normalized := strings.TrimSpace(genre)
	return strings.Contains(normalized, "玄幻") || strings.Contains(normalized, "修真")
}
