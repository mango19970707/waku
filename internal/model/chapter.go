package model

import (
	"time"

	"gorm.io/gorm"
)

// Chapter 表示小说的一个章节
type Chapter struct {
	// ID 章节唯一标识（自增ID）
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// VolumeID 所属卷ID（外键）
	VolumeID uint `gorm:"not null;index" json:"volume_id"`

	// NovelID 所属小说ID（外键，冗余字段便于查询）
	NovelID uint `gorm:"not null;index" json:"novel_id"`

	// ChapterNumber 章节号（第1章、第2章...）
	ChapterNumber int `gorm:"not null" json:"chapter_number"`

	// Title 章节标题
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Synopsis 章节简介
	Synopsis string `gorm:"type:text" json:"synopsis"`

	// Outline 章节大纲
	Outline string `gorm:"type:text" json:"outline"`

	// Content 章节正文内容
	Content string `gorm:"type:text" json:"content"`

	// WordCount 字数统计
	WordCount int `gorm:"default:0" json:"word_count"`

	// Status 章节状态（draft-草稿、writing-连载中、completed-已完结）
	Status string `gorm:"type:varchar(20);default:'draft'" json:"status"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 软删除时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联关系
	Volume Volume `gorm:"foreignKey:VolumeID" json:"-"`
	Novel  Novel  `gorm:"foreignKey:NovelID" json:"-"`
}
