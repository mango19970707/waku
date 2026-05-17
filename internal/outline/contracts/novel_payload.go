package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	genreXuanhuan = "玄幻"
)

// NovelPayload 是 outline 模块各个 Agent 的统一输出结构。
type NovelPayload struct {
	Title    string `json:"title"`
	Synopsis string `json:"synopsis"`
	Outline  string `json:"outline"`
}

type xuanhuanOutlinePayload struct {
	Outline           string `json:"outline"`
	Worldview         string `json:"worldview"`
	CultivationSystem string `json:"cultivation_system"`
	CoreConflicts     string `json:"core_conflicts"`
	MajorCharacters   string `json:"major_characters"`
}

// NormalizeNovelPayload 校验并规范化 Agent 输出。
func NormalizeNovelPayload(raw, genre string) (string, *NovelPayload, error) {
	var payload NovelPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 Novel JSON 失败: %w", err)
	}

	payload.Title = strings.TrimSpace(payload.Title)
	payload.Synopsis = strings.TrimSpace(payload.Synopsis)
	payload.Outline = strings.TrimSpace(payload.Outline)

	if payload.Title == "" {
		return "", nil, fmt.Errorf("title 不能为空")
	}
	if payload.Synopsis == "" {
		return "", nil, fmt.Errorf("synopsis 不能为空")
	}
	if payload.Outline == "" {
		return "", nil, fmt.Errorf("outline 不能为空")
	}

	outlineJSON, _, err := NormalizeOutlineJSON(payload.Outline, genre)
	if err != nil {
		return "", nil, err
	}
	payload.Outline = outlineJSON

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 Novel JSON 失败: %w", err)
	}

	return string(normalized), &payload, nil
}

// NormalizeOutlineJSON 校验并规范化 Outline 内层 JSON 字符串。
func NormalizeOutlineJSON(raw, genre string) (string, map[string]any, error) {
	var outlineMap map[string]any
	if err := json.Unmarshal([]byte(raw), &outlineMap); err != nil {
		return "", nil, fmt.Errorf("解析 outline 内层 JSON 失败: %w", err)
	}

	outlineText, _ := outlineMap["outline"].(string)
	if strings.TrimSpace(outlineText) == "" {
		return "", nil, fmt.Errorf("outline 内层字段 outline 不能为空")
	}

	if err := validateOutlineByGenre(outlineMap, genre); err != nil {
		return "", nil, err
	}

	normalized, err := json.Marshal(outlineMap)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 outline 内层 JSON 失败: %w", err)
	}
	return string(normalized), outlineMap, nil
}

// BuildNovelPayload 构造统一的 Novel JSON 输出。
func BuildNovelPayload(title, synopsis string, outlineMap map[string]any, genre string) (string, *NovelPayload, error) {
	outlineJSON, _, err := NormalizeOutlineJSON(mustMarshal(outlineMap), genre)
	if err != nil {
		return "", nil, err
	}

	raw := mustMarshal(NovelPayload{
		Title:    strings.TrimSpace(title),
		Synopsis: strings.TrimSpace(synopsis),
		Outline:  outlineJSON,
	})
	return NormalizeNovelPayload(raw, genre)
}

func validateOutlineByGenre(outlineMap map[string]any, genre string) error {
	switch {
	case isXuanhuanGenre(genre):
		return validateXuanhuanOutline(outlineMap)
	default:
		// TODO: 按小说类型扩展更多的 Outline JSON 校验规则。
		return nil
	}
}

func validateXuanhuanOutline(outlineMap map[string]any) error {
	var payload xuanhuanOutlinePayload
	if err := decodeOutlineMap(outlineMap, &payload); err != nil {
		return fmt.Errorf("解析玄幻大纲 JSON 失败: %w", err)
	}

	payload.Outline = strings.TrimSpace(payload.Outline)
	payload.Worldview = strings.TrimSpace(payload.Worldview)
	payload.CultivationSystem = strings.TrimSpace(payload.CultivationSystem)

	if payload.Outline == "" {
		return fmt.Errorf("玄幻题材要求 outline 不能为空")
	}
	if payload.Worldview == "" {
		return fmt.Errorf("玄幻题材要求 worldview 不能为空")
	}
	if payload.CultivationSystem == "" {
		return fmt.Errorf("玄幻题材要求 cultivation_system 不能为空")
	}
	if payload.CoreConflicts == "" {
		return fmt.Errorf("玄幻题材要求 core_conflicts 不能为空")
	}
	if payload.MajorCharacters == "" {
		return fmt.Errorf("玄幻题材要求 major_characters 不能为空")
	}

	return nil
}

func decodeOutlineMap(outlineMap map[string]any, target any) error {
	data, err := json.Marshal(outlineMap)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func mustMarshal(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func isXuanhuanGenre(genre string) bool {
	normalized := strings.TrimSpace(genre)
	return strings.Contains(normalized, genreXuanhuan)
}
