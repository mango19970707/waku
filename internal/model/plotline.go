package model

import (
	"time"

	"gorm.io/gorm"
)

// Plotline 表示小说的一条剧情线
type Plotline struct {
	// ID 剧情线唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// NovelID 所属小说 ID
	NovelID string `gorm:"type:varchar(36);not null;index" json:"novel_id"`

	// Title 剧情线标题
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Type 剧情线类型（main-主线、sub-支线）
	Type string `gorm:"type:varchar(20);default:'sub'" json:"type"`

	// Description 剧情线描述
	Description string `gorm:"type:text" json:"description"`

	// Status 剧情线状态（active-进行中、completed-已完结、paused-暂停）
	Status string `gorm:"type:varchar(20);default:'active'" json:"status"`

	// StartChapter 起始章节序号
	StartChapter int `gorm:"default:0" json:"start_chapter"`

	// EndChapter 结束章节序号（0 表示未结束）
	EndChapter int `gorm:"default:0" json:"end_chapter"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
