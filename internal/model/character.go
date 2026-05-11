package model

import (
	"time"

	"gorm.io/gorm"
)

// Character 表示小说中的一个角色
type Character struct {
	// ID 角色唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// NovelID 所属小说 ID
	NovelID string `gorm:"type:varchar(36);not null;index" json:"novel_id"`

	// Name 角色名称
	Name string `gorm:"type:varchar(100);not null" json:"name"`

	// Role 角色定位（protagonist-主角、supporting-配角、antagonist-反派等）
	Role string `gorm:"type:varchar(50)" json:"role"`

	// Age 年龄
	Age int `json:"age,omitempty"`

	// Gender 性别
	Gender string `gorm:"type:varchar(20)" json:"gender,omitempty"`

	// Appearance 外貌描述
	Appearance string `gorm:"type:text" json:"appearance"`

	// Personality 性格特征
	Personality string `gorm:"type:text" json:"personality"`

	// Background 背景故事
	Background string `gorm:"type:text" json:"background"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
