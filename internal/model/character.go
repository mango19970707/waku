package model

import (
	"time"

	"gorm.io/gorm"
)

// Character 表示小说中的一个角色
type Character struct {
	// ID 角色唯一标识
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// NovelID 所属小说 ID
	NovelID uint `gorm:"not null;index" json:"novel_id"`

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

	// PlotlineIDs 角色关联的剧情线ID列表
	PlotlineIDs UintArray `gorm:"type:text" json:"plotline_ids"`

	// Relationships 角色关系映射，格式：{"角色ID": "和当前角色的关系描述"}
	Relationships RelationshipMap `gorm:"type:text" json:"relationships"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
