package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"gorm.io/gorm"
)

type Repository interface {
	Create(novel *model.Novel) error
	GetByID(id uint) (*model.Novel, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(novel *model.Novel) error {
	now := time.Now()
	novel.CreatedAt = now
	novel.UpdatedAt = now
	if novel.Status == "" {
		novel.Status = "draft"
	}
	if err := r.db.Create(novel).Error; err != nil {
		return fmt.Errorf("创建小说失败: %w", err)
	}
	return nil
}

func (r *repository) GetByID(id uint) (*model.Novel, error) {
	var novel model.Novel
	if err := r.db.Where("id = ?", id).First(&novel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("小说不存在: %d", id)
		}
		return nil, fmt.Errorf("获取小说失败: %w", err)
	}
	return &novel, nil
}
