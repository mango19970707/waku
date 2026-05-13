package model

import (
	"time"

	"gorm.io/gorm"
)

// Volume 表示小说的一个分卷
type Volume struct {
	// ID 分卷唯一标识（自增ID）
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// NovelID 所属小说ID（外键）
	NovelID uint `gorm:"not null;index" json:"novel_id"`

	// VolumeNumber 卷号（第1卷、第2卷...）
	VolumeNumber int `gorm:"not null" json:"volume_number"`

	// Title 卷标题
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Synopsis 卷简介
	Synopsis string `gorm:"type:text" json:"synopsis"`

	// Theme 卷主题
	Theme string `gorm:"type:text" json:"theme"`

	// Outline 卷大纲
	Outline string `gorm:"type:text" json:"outline"`

	// Status 卷状态（draft-草稿、writing-连载中、completed-已完结）
	Status string `gorm:"type:varchar(20);default:'draft'" json:"status"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Novel    Novel     `gorm:"foreignKey:NovelID" json:"-"`
	Chapters []Chapter `gorm:"foreignKey:VolumeID" json:"chapters,omitempty"`
}
