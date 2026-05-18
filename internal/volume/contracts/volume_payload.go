package contracts

import (
	"encoding/json"
	"fmt"
	"strings"

	"waku/internal/model"
)

const (
	volumeStatusDraft     = "draft"
	volumeStatusWriting   = "writing"
	volumeStatusCompleted = "completed"
)

// VolumePayload 是分卷工作流统一输出结构。
type VolumePayload struct {
	Volumes []VolumeItem `json:"volumes"`
}

// VolumeItem 表示单个分卷。
type VolumeItem struct {
	ID           uint   `json:"id,omitempty"`
	NovelID      uint   `json:"novel_id,omitempty"`
	VolumeNumber int    `json:"volume_number"`
	Title        string `json:"title"`
	Synopsis     string `json:"synopsis,omitempty"`
	Theme        string `json:"theme,omitempty"`
	Outline      string `json:"outline"`
	Status       string `json:"status,omitempty"`
}

// NormalizeVolumePayload 校验并规范化 Agent 输出。
func NormalizeVolumePayload(raw string) (string, *VolumePayload, error) {
	var payload VolumePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 volumes JSON 失败: %w", err)
	}
	if len(payload.Volumes) == 0 {
		return "", nil, fmt.Errorf("volumes 不能为空")
	}

	seenNumbers := make(map[int]struct{}, len(payload.Volumes))
	for i := range payload.Volumes {
		item := &payload.Volumes[i]
		item.Title = strings.TrimSpace(item.Title)
		item.Synopsis = strings.TrimSpace(item.Synopsis)
		item.Theme = strings.TrimSpace(item.Theme)
		item.Outline = strings.TrimSpace(item.Outline)

		if item.VolumeNumber <= 0 {
			return "", nil, fmt.Errorf("第%d个分卷的 volume_number 必须大于 0", i+1)
		}
		if item.Title == "" {
			return "", nil, fmt.Errorf("第%d个分卷的 title 不能为空", i+1)
		}
		if item.Outline == "" {
			return "", nil, fmt.Errorf("第%d个分卷的 outline 不能为空", i+1)
		}
		if _, ok := seenNumbers[item.VolumeNumber]; ok {
			return "", nil, fmt.Errorf("重复的 volume_number: %d", item.VolumeNumber)
		}
		seenNumbers[item.VolumeNumber] = struct{}{}

		status, err := normalizeVolumeStatus(item.Status)
		if err != nil {
			return "", nil, fmt.Errorf("第%d个分卷的 status 无效: %w", i+1, err)
		}
		item.Status = status
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 volumes JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

// BuildVolumePayload 根据结构化数据构造统一输出。
func BuildVolumePayload(volumes []VolumeItem) (string, *VolumePayload, error) {
	data, err := json.Marshal(VolumePayload{Volumes: volumes})
	if err != nil {
		return "", nil, fmt.Errorf("构造 volumes JSON 失败: %w", err)
	}
	return NormalizeVolumePayload(string(data))
}

// VolumeItemToModel 将统一输出项转换为数据库模型。
func VolumeItemToModel(item VolumeItem, novelID uint) (*model.Volume, error) {
	status, err := normalizeVolumeStatus(item.Status)
	if err != nil {
		return nil, err
	}

	if item.VolumeNumber <= 0 {
		return nil, fmt.Errorf("volume_number 必须大于 0")
	}
	if strings.TrimSpace(item.Title) == "" {
		return nil, fmt.Errorf("title 不能为空")
	}
	if strings.TrimSpace(item.Outline) == "" {
		return nil, fmt.Errorf("outline 不能为空")
	}

	return &model.Volume{
		ID:           item.ID,
		NovelID:      novelID,
		VolumeNumber: item.VolumeNumber,
		Title:        strings.TrimSpace(item.Title),
		Synopsis:     strings.TrimSpace(item.Synopsis),
		Theme:        strings.TrimSpace(item.Theme),
		Outline:      strings.TrimSpace(item.Outline),
		Status:       status,
	}, nil
}

// VolumeModelToItem 将数据库模型转换为统一输出项。
func VolumeModelToItem(volume *model.Volume) VolumeItem {
	if volume == nil {
		return VolumeItem{}
	}

	status, _ := normalizeVolumeStatus(volume.Status)
	return VolumeItem{
		ID:           volume.ID,
		NovelID:      volume.NovelID,
		VolumeNumber: volume.VolumeNumber,
		Title:        strings.TrimSpace(volume.Title),
		Synopsis:     strings.TrimSpace(volume.Synopsis),
		Theme:        strings.TrimSpace(volume.Theme),
		Outline:      strings.TrimSpace(volume.Outline),
		Status:       status,
	}
}

func normalizeVolumeStatus(status string) (string, error) {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return volumeStatusDraft, nil
	}
	switch normalized {
	case volumeStatusDraft, volumeStatusWriting, volumeStatusCompleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported status %q", normalized)
	}
}

// VolumeReviewPayload 是分卷审核输出结构。
type VolumeReviewPayload struct {
	Suggestions []VolumeReviewSuggestion `json:"suggestions"`
}

// VolumeReviewSuggestion 表示单条审核建议。
type VolumeReviewSuggestion struct {
	Suggestion string `json:"suggestion"` // 修改建议
	Target     string `json:"target"`     // 修改目标：outline/plotline/character/volume
	ID         uint   `json:"id"`         // 目标ID
}

// NormalizeVolumeReviewPayload 校验并规范化审核 Agent 输出。
func NormalizeVolumeReviewPayload(raw string) (string, *VolumeReviewPayload, error) {
	var payload VolumeReviewPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 review suggestions JSON 失败: %w", err)
	}
	if payload.Suggestions == nil {
		return "", nil, fmt.Errorf("suggestions 字段不能为空（应为数组，可以是空数组 []）")
	}

	// 允许空的 suggestions 数组（表示所有分卷都合理，无需修改）
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
		case "outline", "plotline", "character", "volume":
			// 有效的 target
		default:
			return "", nil, fmt.Errorf("第%d条建议的 target 无效: %s（应为 outline/plotline/character/volume）", i+1, item.Target)
		}

		payload.Suggestions[i] = item
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 review suggestions JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}
