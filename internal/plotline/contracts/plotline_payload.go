package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	plotlineTypeMain = "main"
	plotlineTypeSub  = "sub"

	plotlineStatusActive    = "active"
	plotlineStatusPaused    = "paused"
	plotlineStatusCompleted = "completed"
)

// PlotlinePayload 表示剧情线集合的统一 JSON 结构。
type PlotlinePayload struct {
	Plotlines []map[string]any `json:"plotlines"`
}

// PlotlineReviewPayload 表示审核节点输出的修改建议集合。
type PlotlineReviewPayload struct {
	Reviews []map[string]any `json:"reviews"`
}

// NormalizePlotlinePayload 校验并规范化剧情线集合 JSON。
func NormalizePlotlinePayload(raw string) (string, *PlotlinePayload, error) {
	var payload PlotlinePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("parse plotlines JSON failed: %w", err)
	}

	if len(payload.Plotlines) == 0 {
		return "", nil, fmt.Errorf("plotlines cannot be empty")
	}

	for i, item := range payload.Plotlines {
		if err := normalizePlotlineItem(item, i+1); err != nil {
			return "", nil, err
		}
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("marshal plotlines JSON failed: %w", err)
	}
	return string(normalized), &payload, nil
}

// BuildPlotlinePayload 从结构化数据构造统一 payload。
func BuildPlotlinePayload(plotlines []map[string]any) (string, *PlotlinePayload, error) {
	data, err := json.Marshal(PlotlinePayload{Plotlines: plotlines})
	if err != nil {
		return "", nil, fmt.Errorf("marshal plotlines JSON failed: %w", err)
	}
	return NormalizePlotlinePayload(string(data))
}

// NormalizePlotlineReviewPayload 校验并规范化审核建议 JSON。
func NormalizePlotlineReviewPayload(raw string) (string, *PlotlineReviewPayload, error) {
	var payload PlotlineReviewPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("parse plotline reviews JSON failed: %w", err)
	}

	if len(payload.Reviews) == 0 {
		return "", nil, fmt.Errorf("reviews cannot be empty")
	}

	for i, item := range payload.Reviews {
		if item == nil {
			return "", nil, fmt.Errorf("review %d cannot be nil", i+1)
		}

		originalContent, ok := item["original_content"].(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("review %d original_content is invalid", i+1)
		}
		if err := normalizePlotlineItem(originalContent, i+1); err != nil {
			return "", nil, fmt.Errorf("review %d original_content is invalid: %w", i+1, err)
		}

		suggestion, ok := normalizeStringField(item, "modification_suggestion")
		if !ok || suggestion == "" {
			return "", nil, fmt.Errorf("review %d modification_suggestion cannot be empty", i+1)
		}

		item["original_content"] = originalContent
		item["modification_suggestion"] = suggestion
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("marshal plotline reviews JSON failed: %w", err)
	}
	return string(normalized), &payload, nil
}

func normalizePlotlineItem(item map[string]any, index int) error {
	if item == nil {
		return fmt.Errorf("plotline %d cannot be nil", index)
	}

	title, ok := normalizeStringField(item, "title")
	if !ok || title == "" {
		return fmt.Errorf("plotline %d title cannot be empty", index)
	}

	plotlineType, ok := normalizeStringField(item, "type")
	if !ok || (plotlineType != plotlineTypeMain && plotlineType != plotlineTypeSub) {
		return fmt.Errorf("plotline %d type is invalid: %v", index, item["type"])
	}

	description, ok := normalizeStringField(item, "description")
	if !ok || description == "" {
		return fmt.Errorf("plotline %d description cannot be empty", index)
	}

	status, _ := normalizeStringField(item, "status")
	if status == "" {
		status = plotlineStatusActive
	}
	if !isValidPlotlineStatus(status) {
		return fmt.Errorf("plotline %d status is invalid: %s", index, status)
	}

	item["title"] = title
	item["type"] = plotlineType
	item["description"] = description
	item["status"] = status

	if id, exists, err := normalizeUintField(item, "id"); err != nil {
		return fmt.Errorf("plotline %d id is invalid: %w", index, err)
	} else if exists {
		item["id"] = id
	}

	return nil
}

func normalizeStringField(item map[string]any, key string) (string, bool) {
	value, ok := item[key]
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(str), true
}

func normalizeUintField(item map[string]any, key string) (uint, bool, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return 0, false, nil
	}

	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, true, fmt.Errorf("cannot be negative")
		}
		return uint(v), true, nil
	case int:
		if v < 0 {
			return 0, true, fmt.Errorf("cannot be negative")
		}
		return uint(v), true, nil
	case int64:
		if v < 0 {
			return 0, true, fmt.Errorf("cannot be negative")
		}
		return uint(v), true, nil
	case uint:
		return v, true, nil
	case json.Number:
		num, err := v.Int64()
		if err != nil {
			return 0, true, err
		}
		if num < 0 {
			return 0, true, fmt.Errorf("cannot be negative")
		}
		return uint(num), true, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false, nil
		}
		num := json.Number(trimmed)
		parsed, err := num.Int64()
		if err != nil {
			return 0, true, err
		}
		if parsed < 0 {
			return 0, true, fmt.Errorf("cannot be negative")
		}
		return uint(parsed), true, nil
	default:
		return 0, true, fmt.Errorf("unsupported type %T", value)
	}
}

func isValidPlotlineStatus(status string) bool {
	switch status {
	case plotlineStatusActive, plotlineStatusPaused, plotlineStatusCompleted:
		return true
	default:
		return false
	}
}
