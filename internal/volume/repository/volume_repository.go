package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"gorm.io/gorm"
)

type volumeRepository struct {
	db *gorm.DB
}

// NewVolumeRepository 创建分卷仓储。
func NewVolumeRepository(db *gorm.DB) VolumeRepository {
	return &volumeRepository{db: db}
}

func (r *volumeRepository) CreateBatch(volumes []*model.Volume) error {
	if len(volumes) == 0 {
		return nil
	}

	now := time.Now()
	for _, volume := range volumes {
		if volume.NovelID == 0 {
			return fmt.Errorf("分卷所属小说ID不能为空")
		}
		if volume.Status == "" {
			volume.Status = "draft"
		}
		volume.CreatedAt = now
		volume.UpdatedAt = now
	}

	if err := r.db.Create(volumes).Error; err != nil {
		return fmt.Errorf("批量创建分卷失败: %w", err)
	}
	return nil
}

func (r *volumeRepository) GetByID(id uint) (*model.Volume, error) {
	if id == 0 {
		return nil, fmt.Errorf("分卷ID不能为空")
	}

	var volume model.Volume
	if err := r.db.Where("id = ?", id).First(&volume).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分卷不存在: %d", id)
		}
		return nil, fmt.Errorf("获取分卷失败: %w", err)
	}
	return &volume, nil
}

func (r *volumeRepository) GetByNovelID(novelID uint) ([]*model.Volume, error) {
	var volumes []*model.Volume
	if err := r.db.Where("novel_id = ?", novelID).
		Order("volume_number ASC, id ASC").
		Find(&volumes).Error; err != nil {
		return nil, fmt.Errorf("获取分卷列表失败: %w", err)
	}
	return volumes, nil
}

func (r *volumeRepository) Update(volume *model.Volume) error {
	if volume == nil {
		return fmt.Errorf("分卷不能为空")
	}
	if volume.ID == 0 {
		return fmt.Errorf("分卷ID不能为空")
	}

	updates := map[string]any{
		"novel_id":      volume.NovelID,
		"volume_number": volume.VolumeNumber,
		"title":         volume.Title,
		"synopsis":      volume.Synopsis,
		"theme":         volume.Theme,
		"outline":       volume.Outline,
		"status":        volume.Status,
		"updated_at":    time.Now(),
	}

	result := r.db.Model(&model.Volume{}).Where("id = ?", volume.ID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新分卷失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("分卷不存在: %d", volume.ID)
	}
	return nil
}

func (r *volumeRepository) Delete(id uint) error {
	if id == 0 {
		return fmt.Errorf("分卷ID不能为空")
	}

	result := r.db.Where("id = ?", id).Delete(&model.Volume{})
	if result.Error != nil {
		return fmt.Errorf("删除分卷失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("分卷不存在: %d", id)
	}
	return nil
}

func (r *volumeRepository) DeleteByNovelID(novelID uint) error {
	if novelID == 0 {
		return fmt.Errorf("小说ID不能为空")
	}

	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Volume{})
	if result.Error != nil {
		return fmt.Errorf("删除小说分卷失败: %w", result.Error)
	}
	return nil
}
