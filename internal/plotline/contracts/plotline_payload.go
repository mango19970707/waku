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

// PlotlinePayload 是 plotline 模块各 Agent 的统一输出结构。
type PlotlinePayload struct {
	Plotlines []map[string]any `json:"plotlines"`
}

// NormalizePlotlinePayload 校验并规范化 Agent 输出。
func NormalizePlotlinePayload(raw string) (string, *PlotlinePayload, error) {
	var payload PlotlinePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 plotlines JSON 失败: %w", err)
	}

	if len(payload.Plotlines) == 0 {
		return "", nil, fmt.Errorf("plotlines 不能为空")
	}

	for i, item := range payload.Plotlines {
		if item == nil {
			return "", nil, fmt.Errorf("第 %d 条剧情线不能为空", i+1)
		}

		title, ok := normalizeStringField(item, "title")
		if !ok || title == "" {
			return "", nil, fmt.Errorf("第 %d 条剧情线的 title 不能为空", i+1)
		}

		plotlineType, ok := normalizeStringField(item, "type")
		if !ok || (plotlineType != plotlineTypeMain && plotlineType != plotlineTypeSub) {
			return "", nil, fmt.Errorf("第 %d 条剧情线的 type 无效: %v", i+1, item["type"])
		}

		description, ok := normalizeStringField(item, "description")
		if !ok || description == "" {
			return "", nil, fmt.Errorf("第 %d 条剧情线的 description 不能为空", i+1)
		}

		status, _ := normalizeStringField(item, "status")
		if status == "" {
			status = plotlineStatusActive
		}
		if !isValidPlotlineStatus(status) {
			return "", nil, fmt.Errorf("第 %d 条剧情线的 status 无效: %s", i+1, status)
		}

		item["title"] = title
		item["type"] = plotlineType
		item["description"] = description
		item["status"] = status

		if id, exists, err := normalizeUintField(item, "id"); err != nil {
			return "", nil, fmt.Errorf("第 %d 条剧情线的 id 无效: %w", i+1, err)
		} else if exists {
			item["id"] = id
		}
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 plotlines JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

// BuildPlotlinePayload 由结构化数据构造统一 payload。
func BuildPlotlinePayload(plotlines []map[string]any) (string, *PlotlinePayload, error) {
	data, err := json.Marshal(PlotlinePayload{Plotlines: plotlines})
	if err != nil {
		return "", nil, fmt.Errorf("构造 plotlines JSON 失败: %w", err)
	}
	return NormalizePlotlinePayload(string(data))
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
			return 0, true, fmt.Errorf("不能为负数")
		}
		return uint(v), true, nil
	case int:
		if v < 0 {
			return 0, true, fmt.Errorf("不能为负数")
		}
		return uint(v), true, nil
	case int64:
		if v < 0 {
			return 0, true, fmt.Errorf("不能为负数")
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
			return 0, true, fmt.Errorf("不能为负数")
		}
		return uint(num), true, nil
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false, nil
		}
		var num json.Number = json.Number(trimmed)
		parsed, err := num.Int64()
		if err != nil {
			return 0, true, err
		}
		if parsed < 0 {
			return 0, true, fmt.Errorf("不能为负数")
		}
		return uint(parsed), true, nil
	default:
		return 0, true, fmt.Errorf("不支持的类型 %T", value)
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
