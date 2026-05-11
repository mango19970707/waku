package model

import (
	"time"

	"gorm.io/gorm"
)

// Memory 表示故事上下文记忆
type Memory struct {
	// ID 记忆唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// NovelID 所属小说 ID
	NovelID string `gorm:"type:varchar(36);not null;index" json:"novel_id"`

	// ChapterID 关联章节 ID
	ChapterID string `gorm:"type:varchar(36);index" json:"chapter_id"`

	// Type 记忆类型（event-事件、state-状态、setting-设定）
	Type string `gorm:"type:varchar(20);not null" json:"type"`

	// Content 记忆内容
	Content string `gorm:"type:text;not null" json:"content"`

	// Importance 重要程度（1-5，5 最重要）
	Importance int `gorm:"default:3" json:"importance"`

	// Tags 标签列表，用于快速检索
	Tags StringArray `gorm:"type:jsonb" json:"tags"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
