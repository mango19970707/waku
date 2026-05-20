package model

import (
	"time"

	"gorm.io/gorm"
)

// Event 表示小说中的一个关键事件
type Event struct {
	// ID 事件唯一标识
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// NovelID 所属小说 ID
	NovelID uint `gorm:"not null;index" json:"novel_id"`

	// ChapterID 所属章节 ID（可选，事件可能跨章节）
	ChapterID uint `gorm:"index" json:"chapter_id,omitempty"`

	// Name 事件名称
	Name string `gorm:"type:varchar(200);not null" json:"name"`

	// Description 事件描述
	Description string `gorm:"type:text" json:"description"`

	// Type 事件类型（conflict-冲突、breakthrough-突破、encounter-相遇、discovery-发现等）
	Type string `gorm:"type:varchar(50)" json:"type"`

	// PlotlineIDs 事件关联的剧情线ID列表
	PlotlineIDs UintArray `gorm:"type:text" json:"plotline_ids"`

	// CharacterIDs 事件涉及的角色ID列表
	CharacterIDs UintArray `gorm:"type:text" json:"character_ids"`

	// Impact 事件影响程度（low-低、medium-中、high-高、critical-关键）
	Impact string `gorm:"type:varchar(20);default:'medium'" json:"impact"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
