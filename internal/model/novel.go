package model

import (
	"time"

	"gorm.io/gorm"
)

// Novel 表示一部小说的核心信息。
type Novel struct {
	// ID 是小说主键。
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// Title 是小说标题。
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Author 是作者名称。
	Author string `gorm:"type:varchar(100);not null" json:"author"`

	// Genre 是小说类型，目前大纲工作流只要求兼容玄幻和修真。
	Genre string `gorm:"type:varchar(50)" json:"genre"`

	// Synopsis 是小说简介。
	Synopsis string `gorm:"type:text" json:"synopsis"`

	// Outline 保存大纲内层 JSON 字符串。
	// 当前大纲工作流约定该 JSON 至少包含 outline 字段；
	// 当 Genre 为玄幻或修真时，还必须包含 worldview 和 cultivation_system 字段。
	Outline string `gorm:"type:text" json:"outline"`

	// Status 是小说状态：draft、writing、completed。
	Status string `gorm:"type:varchar(20);default:'draft'" json:"status"`

	// CreatedAt 是创建时间。
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 是更新时间。
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 是软删除时间。
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
