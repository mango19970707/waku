package model

import (
	"time"

	"gorm.io/gorm"
)

// PlotlineProgress 表示剧情线在某个章节的推进进度
type PlotlineProgress struct {
	// ID 进度记录唯一标识
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// NovelID 所属小说 ID
	NovelID uint `gorm:"not null;index" json:"novel_id"`

	// PlotlineID 剧情线 ID
	PlotlineID uint `gorm:"not null;index" json:"plotline_id"`

	// ChapterID 章节 ID
	ChapterID uint `gorm:"not null;index" json:"chapter_id"`

	// Progress 进度描述（例如："主角正式进入修仙世界"）
	Progress string `gorm:"type:text;not null" json:"progress"`

	// ProgressPercentage 进度百分比（0-100）
	ProgressPercentage int `gorm:"default:0" json:"progress_percentage"`

	// Status 状态（active-进行中、paused-暂停、completed-完成）
	Status string `gorm:"type:varchar(20);default:'active'" json:"status"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Plotline Plotline `gorm:"foreignKey:PlotlineID" json:"-"`
	Chapter  Chapter  `gorm:"foreignKey:ChapterID" json:"-"`
}
