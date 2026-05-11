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

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言失败：期望 []byte")
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
