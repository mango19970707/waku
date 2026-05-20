package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"waku/internal/model"
)

const (
	contentStatusWriting   = "writing"
	contentStatusCompleted = "completed"
)

// ContentPayload 是正文生成、润色、修改 Agent 的统一输出契约。
type ContentPayload struct {
	Chapter ContentItem `json:"chapter"`
}

// ContentItem 是单章正文的可保存数据结构，只承载正文工作流允许修改的字段。
type ContentItem struct {
	ID            uint   `json:"id"`
	NovelID       uint   `json:"novel_id"`
	VolumeID      uint   `json:"volume_id"`
	ChapterNumber int    `json:"chapter_number"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	WordCount     int    `json:"word_count"`
	Status        string `json:"status,omitempty"`
}

// ReviewSuggestion 是正文审核 Agent 输出的单条修改建议。
type ReviewSuggestion struct {
	Suggestion string `json:"suggestion"`
	Target     string `json:"target"`
	ID         uint   `json:"id"`
}

// ReviewPayload 是正文审核 Agent 的统一输出契约。
type ReviewPayload struct {
	Suggestions []ReviewSuggestion `json:"suggestions"`
}

// NormalizeContentPayload 校验并规范化正文 Agent 输出，确保保存层只接收结构稳定的 JSON。
func NormalizeContentPayload(raw string) (string, *ContentPayload, error) {
	var payload ContentPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 chapter JSON 失败: %w", err)
	}

	item := &payload.Chapter
	item.Title = strings.TrimSpace(item.Title)
	item.Content = strings.TrimSpace(item.Content)
	if item.ID == 0 {
		return "", nil, fmt.Errorf("chapter.id 不能为空")
	}
	if item.NovelID == 0 {
		return "", nil, fmt.Errorf("chapter.novel_id 不能为空")
	}
	if item.VolumeID == 0 {
		return "", nil, fmt.Errorf("chapter.volume_id 不能为空")
	}
	if item.ChapterNumber <= 0 {
		return "", nil, fmt.Errorf("chapter.chapter_number 必须大于 0")
	}
	if item.Title == "" {
		return "", nil, fmt.Errorf("chapter.title 不能为空")
	}
	if item.Content == "" {
		return "", nil, fmt.Errorf("chapter.content 不能为空")
	}

	status, err := normalizeContentStatus(item.Status)
	if err != nil {
		return "", nil, fmt.Errorf("chapter.status 无效: %w", err)
	}
	item.Status = status
	item.WordCount = CountContentWords(item.Content)

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 chapter JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

// NormalizeContentReviewPayload 校验并规范化正文审核 Agent 输出，允许空建议表示审核通过。
func NormalizeContentReviewPayload(raw string) (string, *ReviewPayload, error) {
	var payload ReviewPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 suggestions JSON 失败: %w", err)
	}
	if payload.Suggestions == nil {
		return "", nil, fmt.Errorf("suggestions 字段不能为空（应为数组，可以是空数组 []）")
	}

	for i := range payload.Suggestions {
		item := &payload.Suggestions[i]
		item.Suggestion = strings.TrimSpace(item.Suggestion)
		item.Target = strings.TrimSpace(item.Target)
		if item.Suggestion == "" {
			return "", nil, fmt.Errorf("第%d条建议的 suggestion 不能为空", i+1)
		}
		if item.ID == 0 {
			return "", nil, fmt.Errorf("第%d条建议的 id 不能为 0", i+1)
		}
		if !isValidReviewTarget(item.Target) {
			return "", nil, fmt.Errorf("第%d条建议的 target 无效: %s（应为 content/chapter/character/plotline/volume/outline）", i+1, item.Target)
		}
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 suggestions JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

// FromModel 从章节模型提取正文工作流需要的字段，避免把大纲字段交给正文 Agent 随意修改。
func FromModel(chapter *model.Chapter) *ContentItem {
	return &ContentItem{
		ID:            chapter.ID,
		NovelID:       chapter.NovelID,
		VolumeID:      chapter.VolumeID,
		ChapterNumber: chapter.ChapterNumber,
		Title:         chapter.Title,
		Content:       chapter.Content,
		WordCount:     chapter.WordCount,
		Status:        chapter.Status,
	}
}

// ToModel 将正文契约转换为章节模型，保存层只会读取正文相关字段。
func (c *ContentItem) ToModel() *model.Chapter {
	return &model.Chapter{
		ID:            c.ID,
		NovelID:       c.NovelID,
		VolumeID:      c.VolumeID,
		ChapterNumber: c.ChapterNumber,
		Title:         c.Title,
		Content:       c.Content,
		WordCount:     c.WordCount,
		Status:        c.Status,
	}
}

// BuildContentPayload 将章节模型包装为 Agent 可消费的单章正文 JSON。
func BuildContentPayload(chapter *model.Chapter) (string, error) {
	payload := ContentPayload{Chapter: *FromModel(chapter)}
	data, err := json.Marshal(&payload)
	if err != nil {
		return "", fmt.Errorf("序列化章节正文失败: %w", err)
	}
	return string(data), nil
}

// CountContentWords 按非空白字符统计正文长度，用于兼容中文无空格文本。
func CountContentWords(content string) int {
	count := 0
	for _, r := range content {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func normalizeContentStatus(status string) (string, error) {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return contentStatusWriting, nil
	}
	switch normalized {
	case contentStatusWriting, contentStatusCompleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported status %q", normalized)
	}
}

func isValidReviewTarget(target string) bool {
	switch target {
	case "content", "chapter", "character", "plotline", "volume", "outline":
		return true
	default:
		return false
	}
}
