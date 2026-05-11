package model

import (
	"time"

	"gorm.io/gorm"
)

// Novel 表示一部小说
type Novel struct {
	// ID 小说唯一标识
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// Title 小说标题
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Author 作者名称
	Author string `gorm:"type:varchar(100);not null" json:"author"`

	// Genre 小说类型（如：玄幻、都市、科幻等）
	Genre string `gorm:"type:varchar(50)" json:"genre"`

	// Synopsis 小说简介
	Synopsis string `gorm:"type:text" json:"synopsis"`

	// Worldview 世界观设定（包括世界背景、时代设定、社会结构等）
	Worldview string `gorm:"type:text" json:"worldview"`

	// CultivationSystem 修行体系（包括境界划分、修炼方法、力量体系等）
	CultivationSystem string `gorm:"type:text" json:"cultivation_system"`

	// Status 小说状态（draft-草稿、writing-连载中、completed-已完结）
	Status string `gorm:"type:varchar(20);default:'draft'" json:"status"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
