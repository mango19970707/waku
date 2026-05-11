package model

import (
	"time"

	"gorm.io/gorm"
)

// Abstract 表示章节大纲
type Abstract struct {
	// ID 大纲唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// ChapterID 所属章节 ID
	ChapterID string `gorm:"type:varchar(36);not null;uniqueIndex" json:"chapter_id"`

	// NovelID 所属小说 ID
	NovelID string `gorm:"type:varchar(36);not null;index" json:"novel_id"`

	// Summary 章节摘要
	Summary string `gorm:"type:text" json:"summary"`

	// KeyPoints 关键要点列表
	KeyPoints StringArray `gorm:"type:jsonb" json:"key_points"`

	// Characters 涉及的角色 ID 列表
	Characters StringArray `gorm:"type:jsonb" json:"characters"`

	// Plotlines 涉及的剧情线 ID 列表
	Plotlines StringArray `gorm:"type:jsonb" json:"plotlines"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
