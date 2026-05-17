package contracts

import (
	"encoding/json"
	"fmt"
	"strings"
	"waku/internal/model"
)

type CharacterPayload struct {
	Characters []map[string]any `json:"characters"`
}

type CharacterReviewSuggestion struct {
	OriginalContent        map[string]any `json:"original_content"`
	ModificationSuggestion string         `json:"modification_suggestion"`
}

type CharacterReviewPayload struct {
	Suggestions []CharacterReviewSuggestion `json:"suggestions"`
}

func NormalizeCharacterPayload(raw string) (string, *CharacterPayload, error) {
	var payload CharacterPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 characters JSON 失败: %w", err)
	}
	if len(payload.Characters) == 0 {
		return "", nil, fmt.Errorf("characters 不能为空")
	}

	for i, item := range payload.Characters {
		if item == nil {
			return "", nil, fmt.Errorf("第%d个角色不能为空", i+1)
		}

		name, ok := normalizeStringField(item, "name")
		if !ok || name == "" {
			return "", nil, fmt.Errorf("第%d个角色的 name 不能为空", i+1)
		}
		personality, ok := normalizeStringField(item, "personality")
		if !ok || personality == "" {
			return "", nil, fmt.Errorf("第%d个角色的 personality 不能为空", i+1)
		}
		background, ok := normalizeStringField(item, "background")
		if !ok || background == "" {
			return "", nil, fmt.Errorf("第%d个角色的 background 不能为空", i+1)
		}

		if role, ok := normalizeStringField(item, "role"); ok {
			item["role"] = role
		}
		if gender, ok := normalizeStringField(item, "gender"); ok {
			item["gender"] = gender
		}
		if appearance, ok := normalizeStringField(item, "appearance"); ok {
			item["appearance"] = appearance
		}
		item["name"] = name
		item["personality"] = personality
		item["background"] = background

		if id, exists, err := normalizeUintField(item, "id"); err != nil {
			return "", nil, fmt.Errorf("第%d个角色的 id 无效: %w", i+1, err)
		} else if exists {
			item["id"] = id
		}

		if novelID, exists, err := normalizeUintField(item, "novel_id"); err != nil {
			return "", nil, fmt.Errorf("第%d个角色的 novel_id 无效: %w", i+1, err)
		} else if exists {
			item["novel_id"] = novelID
		}

		if age, exists, err := normalizeIntField(item, "age"); err != nil {
			return "", nil, fmt.Errorf("第%d个角色的 age 无效: %w", i+1, err)
		} else if exists {
			item["age"] = age
		}

		if plotlineIDs, exists, err := normalizeUintSliceField(item, "plotline_ids"); err != nil {
			return "", nil, fmt.Errorf("第%d个角色的 plotline_ids 无效: %w", i+1, err)
		} else if exists {
			item["plotline_ids"] = plotlineIDs
		}

		if relationships, exists, err := normalizeRelationshipsField(item, "relationships"); err != nil {
			return "", nil, fmt.Errorf("第%d个角色的 relationships 无效: %w", i+1, err)
		} else if exists {
			item["relationships"] = relationships
		}
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 characters JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

func BuildCharacterPayload(characters []map[string]any) (string, *CharacterPayload, error) {
	data, err := json.Marshal(CharacterPayload{Characters: characters})
	if err != nil {
		return "", nil, fmt.Errorf("构造 characters JSON 失败: %w", err)
	}
	return NormalizeCharacterPayload(string(data))
}

func NormalizeCharacterReviewPayload(raw string) (string, *CharacterReviewPayload, error) {
	var payload CharacterReviewPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", nil, fmt.Errorf("解析 review suggestions JSON 失败: %w", err)
	}
	if payload.Suggestions == nil {
		return "", nil, fmt.Errorf("suggestions 不能为空")
	}

	for i, item := range payload.Suggestions {
		if item.OriginalContent == nil {
			return "", nil, fmt.Errorf("第%d条修改意见的 original_content 不能为空", i+1)
		}
		if _, err := CharacterMapToModel(item.OriginalContent, 0); err != nil {
			return "", nil, fmt.Errorf("第%d条修改意见的 original_content 无效: %w", i+1, err)
		}

		item.ModificationSuggestion = strings.TrimSpace(item.ModificationSuggestion)
		if item.ModificationSuggestion == "" {
			return "", nil, fmt.Errorf("第%d条修改意见的 modification_suggestion 不能为空", i+1)
		}
		payload.Suggestions[i] = item
	}

	normalized, err := json.Marshal(&payload)
	if err != nil {
		return "", nil, fmt.Errorf("序列化 review suggestions JSON 失败: %w", err)
	}
	return string(normalized), &payload, nil
}

func CharacterMapToModel(item map[string]any, novelID uint) (*model.Character, error) {
	character := &model.Character{
		NovelID:       novelID,
		Name:          getString(item, "name"),
		Role:          getString(item, "role"),
		Age:           getInt(item, "age"),
		Gender:        getString(item, "gender"),
		Appearance:    getString(item, "appearance"),
		Personality:   getString(item, "personality"),
		Background:    getString(item, "background"),
		PlotlineIDs:   getUintArray(item, "plotline_ids"),
		Relationships: getRelationshipMap(item, "relationships"),
	}
	if rawID, ok := item["id"]; ok {
		switch v := rawID.(type) {
		case uint:
			character.ID = v
		case float64:
			character.ID = uint(v)
		case int:
			character.ID = uint(v)
		}
	}
	return character, nil
}

func CharacterModelToMap(character *model.Character) map[string]any {
	return map[string]any{
		"id":            character.ID,
		"novel_id":      character.NovelID,
		"name":          character.Name,
		"role":          character.Role,
		"age":           character.Age,
		"gender":        character.Gender,
		"appearance":    character.Appearance,
		"personality":   character.Personality,
		"background":    character.Background,
		"plotline_ids":  []uint(character.PlotlineIDs),
		"relationships": map[string]string(character.Relationships),
		"created_at":    character.CreatedAt,
		"updated_at":    character.UpdatedAt,
	}
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
		num := json.Number(trimmed)
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

func normalizeIntField(item map[string]any, key string) (int, bool, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return 0, false, nil
	}
	switch v := value.(type) {
	case float64:
		return int(v), true, nil
	case int:
		return v, true, nil
	case int64:
		return int(v), true, nil
	case json.Number:
		num, err := v.Int64()
		if err != nil {
			return 0, true, err
		}
		return int(num), true, nil
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
		return int(parsed), true, nil
	default:
		return 0, true, fmt.Errorf("不支持的类型 %T", value)
	}
}

func normalizeUintSliceField(item map[string]any, key string) ([]uint, bool, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return nil, false, nil
	}
	rawItems, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]uint); ok {
			return typed, true, nil
		}
		return nil, true, fmt.Errorf("应为数组")
	}
	result := make([]uint, 0, len(rawItems))
	for _, raw := range rawItems {
		holder := map[string]any{"value": raw}
		parsed, exists, err := normalizeUintField(holder, "value")
		if err != nil {
			return nil, true, err
		}
		if exists {
			result = append(result, parsed)
		}
	}
	return result, true, nil
}

func normalizeRelationshipsField(item map[string]any, key string) (map[string]string, bool, error) {
	value, ok := item[key]
	if !ok || value == nil {
		return nil, false, nil
	}
	if typed, ok := value.(map[string]string); ok {
		return typed, true, nil
	}
	rawMap, ok := value.(map[string]any)
	if !ok {
		return nil, true, fmt.Errorf("应为对象")
	}
	result := make(map[string]string, len(rawMap))
	for rawKey, rawValue := range rawMap {
		valueStr, ok := rawValue.(string)
		if !ok {
			return nil, true, fmt.Errorf("关系值必须为字符串")
		}
		result[strings.TrimSpace(rawKey)] = strings.TrimSpace(valueStr)
	}
	return result, true, nil
}

func getString(item map[string]any, key string) string {
	value, _ := normalizeStringField(item, key)
	return value
}

func getInt(item map[string]any, key string) int {
	value, _, _ := normalizeIntField(item, key)
	return value
}

func getUintArray(item map[string]any, key string) model.UintArray {
	values, _, _ := normalizeUintSliceField(item, key)
	return model.UintArray(values)
}

func getRelationshipMap(item map[string]any, key string) model.RelationshipMap {
	values, _, _ := normalizeRelationshipsField(item, key)
	return model.RelationshipMap(values)
}
