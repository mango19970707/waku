package contracts

import (
	"encoding/json"
	"fmt"
	"strings"

	"waku/internal/model"
)

const (
	chapterStatusDraft     = "draft"
	chapterStatusWriting   = "writing"
	chapterStatusCompleted = "completed"
)

// ChapterPayload 章节生成/润色/修改 Agent 的输出格式
type ChapterPayload struct {
	Chapters []ChapterItem `json:"chapters"`
}

// ChapterItem 单个章节的数据结构
type ChapterItem struct {
	ID                 uint              `json:"id,omitempty"`
	NovelID            uint              `json:"novel_id,omitempty"`
	VolumeID           uint              `json:"volume_id,omitempty"`
	ChapterNumber      int               `json:"chapter_number"`
	Title              string            `json:"title"`
	Synopsis           string            `json:"synopsis"`
	Outline            string            `json:"outline"`
	WordCountEstimate  int               `json:"word_count_estimate"`
	KeyEvents          []string          `json:"key_events"`
	KeyCharacters      []string          `json:"key_characters"`
	PlotlineProgress   map[string]string `json:"plotline_progress"`
	Status             string            `json:"status,omitempty"`
}

// NormalizeChapterPayload 校验并规范化 Agent 输出
func NormalizeChapterPayload(raw string) (string, *ChapterPayload, error) {
	var payload ChapterPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 chapters JSON 失败: %w", err)
	}
	if len(payload.Chapters) == 0 {
		return "", nil, fmt.Errorf("chapters 不能为空")
	}

	seenNumbers := make(map[int]struct{}, len(payload.Chapters))
	for i := range payload.Chapters {
		item := &payload.Chapters[i]
		item.Title = strings.TrimSpace(item.Title)
		item.Synopsis = strings.TrimSpace(item.Synopsis)
		item.Outline = strings.TrimSpace(item.Outline)

		if item.ChapterNumber <= 0 {
			return "", nil, fmt.Errorf("第%d个章节的 chapter_number 必须大于 0", i+1)
		}
		if item.Title == "" {
			return "", nil, fmt.Errorf("第%d个章节的 title 不能为空", i+1)
		}
		if item.Outline == "" {
			return "", nil, fmt.Errorf("第%d个章节的 outline 不能为空", i+1)
		}
		if _, ok := seenNumbers[item.ChapterNumber]; ok {
			return "", nil, fmt.Errorf("重复的 chapter_number: %d", item.ChapterNumber)
		}
		seenNumbers[item.ChapterNumber] = struct{}{}

		status, err := normalizeChapterStatus(item.Status)
		if err != nil {
			return "", nil, fmt.Errorf("第%d个章节的 status 无效: %w", i+1, err)
		}
		item.Status = status
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 chapters JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

// ToModel 将 ChapterItem 转换为 model.Chapter
func (c *ChapterItem) ToModel() *model.Chapter {
	return &model.Chapter{
		ID:                c.ID,
		NovelID:           c.NovelID,
		VolumeID:          c.VolumeID,
		ChapterNumber:     c.ChapterNumber,
		Title:             c.Title,
		Synopsis:          c.Synopsis,
		Outline:           c.Outline,
		WordCountEstimate: c.WordCountEstimate,
		KeyEvents:         model.StringArray(c.KeyEvents),
		KeyCharacters:     model.StringArray(c.KeyCharacters),
		PlotlineProgress:  model.PlotlineProgressMap(c.PlotlineProgress),
		Status:            c.Status,
	}
}

// FromModel 从 model.Chapter 创建 ChapterItem
func FromModel(chapter *model.Chapter) *ChapterItem {
	return &ChapterItem{
		ID:                chapter.ID,
		NovelID:           chapter.NovelID,
		VolumeID:          chapter.VolumeID,
		ChapterNumber:     chapter.ChapterNumber,
		Title:             chapter.Title,
		Synopsis:          chapter.Synopsis,
		Outline:           chapter.Outline,
		WordCountEstimate: chapter.WordCountEstimate,
		KeyEvents:         []string(chapter.KeyEvents),
		KeyCharacters:     []string(chapter.KeyCharacters),
		PlotlineProgress:  map[string]string(chapter.PlotlineProgress),
		Status:            chapter.Status,
	}
}

// ReviewSuggestion 审核建议
type ReviewSuggestion struct {
	Suggestion string `json:"suggestion"`
	Target     string `json:"target"` // outline/plotline/character/volume/chapter
	ID         uint   `json:"id"`
}

// ReviewPayload 章节审核 Agent 的输出格式
type ReviewPayload struct {
	Suggestions []ReviewSuggestion `json:"suggestions"`
}

func normalizeChapterStatus(status string) (string, error) {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return chapterStatusDraft, nil
	}
	switch normalized {
	case chapterStatusDraft, chapterStatusWriting, chapterStatusCompleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported status %q", normalized)
	}
}

// NormalizeChapterReviewPayload 校验并规范化审核 Agent 输出
func NormalizeChapterReviewPayload(raw string) (string, *ReviewPayload, error) {
	var payload ReviewPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 review suggestions JSON 失败: %w", err)
	}
	if payload.Suggestions == nil {
		return "", nil, fmt.Errorf("suggestions 字段不能为空（应为数组，可以是空数组 []）")
	}

	// 允许空的 suggestions 数组（表示所有章节都合理，无需修改）
	if len(payload.Suggestions) == 0 {
		normalized, err := json.Marshal(&payload)
		if err != nil {
			return "", nil, fmt.Errorf("序列化 review suggestions JSON 失败: %w", err)
		}
		return string(normalized), &payload, nil
	}

	for i, item := range payload.Suggestions {
		item.Suggestion = strings.TrimSpace(item.Suggestion)
		item.Target = strings.TrimSpace(item.Target)

		if item.Suggestion == "" {
			return "", nil, fmt.Errorf("第%d条建议的 suggestion 不能为空", i+1)
		}
		if item.Target == "" {
			return "", nil, fmt.Errorf("第%d条建议的 target 不能为空", i+1)
		}
		if item.ID == 0 {
			return "", nil, fmt.Errorf("第%d条建议的 id 不能为 0", i+1)
		}

		// 验证 target 字段的有效性
		switch item.Target {
		case "outline", "plotline", "character", "volume", "chapter":
			// 有效的 target
		default:
			return "", nil, fmt.Errorf("第%d条建议的 target 无效: %s（应为 outline/plotline/character/volume/chapter）", i+1, item.Target)
		}

		payload.Suggestions[i] = item
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 review suggestions JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}
