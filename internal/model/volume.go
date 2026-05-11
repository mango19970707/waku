package model

import (
	"time"

	"gorm.io/gorm"
)

// Volume 表示小说的一个分卷
type Volume struct {
	// ID 分卷唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// NovelID 所属小说 ID
	NovelID string `gorm:"type:varchar(36);not null;index" json:"novel_id"`

	// Title 分卷标题
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Order 分卷序号（从 1 开始）
	Order int `gorm:"not null" json:"order"`

	// Synopsis 分卷简介
	Synopsis string `gorm:"type:text" json:"synopsis"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
