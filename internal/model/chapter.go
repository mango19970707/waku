package model

import (
	"time"

	"gorm.io/gorm"
)

// Chapter 表示小说的一个章节
type Chapter struct {
	// ID 章节唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// NovelID 所属小说 ID
	NovelID string `gorm:"type:varchar(36);not null;index" json:"novel_id"`

	// VolumeID 所属分卷 ID
	VolumeID string `gorm:"type:varchar(36);not null;index" json:"volume_id"`

	// Title 章节标题
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Order 章节序号（从 1 开始）
	Order int `gorm:"not null" json:"order"`

	// Content 章节内容
	Content string `gorm:"type:text" json:"content"`

	// WordCount 字数统计
	WordCount int `gorm:"default:0" json:"word_count"`

	// Status 章节状态（draft-草稿、published-已发布）
	Status string `gorm:"type:varchar(20);default:'draft'" json:"status"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
