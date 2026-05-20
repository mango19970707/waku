package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// StringArray 字符串数组类型，用于在数据库中存储 JSON 数组
type StringArray []string

// Scan 实现 sql.Scanner 接口，从数据库读取数据
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败：期望 []byte 或 string")
	}

	if len(bytes) == 0 {
		*s = []string{}
		return nil
	}

	return json.Unmarshal(bytes, s)
}

// Value 实现 driver.Valuer 接口，向数据库写入数据
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

// UintArray 无符号整数数组类型，用于在数据库中存储 JSON 数组
type UintArray []uint

// Scan 实现 sql.Scanner 接口，从数据库读取数据
func (u *UintArray) Scan(value interface{}) error {
	if value == nil {
		*u = []uint{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败：期望 []byte 或 string")
	}

	if len(bytes) == 0 {
		*u = []uint{}
		return nil
	}

	return json.Unmarshal(bytes, u)
}

// Value 实现 driver.Valuer 接口，向数据库写入数据
func (u UintArray) Value() (driver.Value, error) {
	if len(u) == 0 {
		return "[]", nil
	}
	return json.Marshal(u)
}

// RelationshipMap 角色关系映射类型，用于在数据库中存储 JSON 对象
// 格式：{"角色ID": "和当前角色的关系描述"}
type RelationshipMap map[string]string

// Scan 实现 sql.Scanner 接口，从数据库读取数据
func (r *RelationshipMap) Scan(value interface{}) error {
	if value == nil {
		*r = make(map[string]string)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败：期望 []byte 或 string")
	}

	if len(bytes) == 0 {
		*r = make(map[string]string)
		return nil
	}

	return json.Unmarshal(bytes, r)
}

// Value 实现 driver.Valuer 接口，向数据库写入数据
func (r RelationshipMap) Value() (driver.Value, error) {
	if len(r) == 0 {
		return "{}", nil
	}
	return json.Marshal(r)
}

// PlotlineProgressMap 剧情线进度映射类型，用于在数据库中存储 JSON 对象
// 格式：{"剧情线ID": "进度描述"}
type PlotlineProgressMap map[string]string

// Scan 实现 sql.Scanner 接口，从数据库读取数据
func (p *PlotlineProgressMap) Scan(value interface{}) error {
	if value == nil {
		*p = make(map[string]string)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("类型断言失败：期望 []byte 或 string")
	}

	if len(bytes) == 0 {
		*p = make(map[string]string)
		return nil
	}

	return json.Unmarshal(bytes, p)
}

// Value 实现 driver.Valuer 接口，向数据库写入数据
func (p PlotlineProgressMap) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "{}", nil
	}
	return json.Marshal(p)
}
