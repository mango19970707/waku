package model

import (
	"time"

	"gorm.io/gorm"
)

// Plotline 表示小说中的一条剧情线。
type Plotline struct {
	// ID 是剧情线主键，使用数据库自增 ID。
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	// NovelID 是所属小说 ID。
	NovelID uint `gorm:"not null;index" json:"novel_id"`

	// Title 是剧情线标题。
	Title string `gorm:"type:varchar(200);not null" json:"title"`

	// Type 是剧情线类型，当前约定为 main 或 sub。
	Type string `gorm:"type:varchar(20);default:'sub'" json:"type"`

	// Description 是剧情线描述。
	Description string `gorm:"type:text" json:"description"`

	// Status 是剧情线状态，例如 active、paused、completed。
	Status string `gorm:"type:varchar(20);default:'active'" json:"status"`

	// CreatedAt 是创建时间。
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt 是更新时间。
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// DeletedAt 是软删除时间。
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
