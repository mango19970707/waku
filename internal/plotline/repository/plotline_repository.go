package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"gorm.io/gorm"
)

// PlotlineRepository 定义剧情线数据访问接口。
type PlotlineRepository interface {
	CreateBatch(plotlines []*model.Plotline) error
	GetByNovelID(novelID uint) ([]*model.Plotline, error)
	GetByID(id uint) (*model.Plotline, error)
	Update(plotline *model.Plotline) error
	Delete(id uint) error
	DeleteByNovelID(novelID uint) error
}

type plotlineRepository struct {
	db *gorm.DB
}

// NewPlotlineRepository 创建剧情线仓储实例。
func NewPlotlineRepository(db *gorm.DB) PlotlineRepository {
	return &plotlineRepository{db: db}
}

// CreateBatch 批量创建剧情线。
func (r *plotlineRepository) CreateBatch(plotlines []*model.Plotline) error {
	if len(plotlines) == 0 {
		return fmt.Errorf("剧情线列表不能为空")
	}

	now := time.Now()
	for _, plotline := range plotlines {
		if plotline.NovelID == 0 {
			return fmt.Errorf("剧情线所属小说 ID 不能为空")
		}
		if plotline.Status == "" {
			plotline.Status = "active"
		}
		plotline.CreatedAt = now
		plotline.UpdatedAt = now
	}

	if err := r.db.Create(plotlines).Error; err != nil {
		return fmt.Errorf("批量创建剧情线失败: %w", err)
	}
	return nil
}

// GetByNovelID 根据小说 ID 获取所有剧情线。
func (r *plotlineRepository) GetByNovelID(novelID uint) ([]*model.Plotline, error) {
	var plotlines []*model.Plotline
	if err := r.db.Where("novel_id = ?", novelID).
		Order("type DESC, created_at ASC, id ASC").
		Find(&plotlines).Error; err != nil {
		return nil, fmt.Errorf("获取剧情线列表失败: %w", err)
	}
	return plotlines, nil
}

// GetByID 根据剧情线 ID 获取单条剧情线。
func (r *plotlineRepository) GetByID(id uint) (*model.Plotline, error) {
	if id == 0 {
		return nil, fmt.Errorf("剧情线 ID 不能为空")
	}

	var plotline model.Plotline
	if err := r.db.Where("id = ?", id).First(&plotline).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("剧情线不存在: %d", id)
		}
		return nil, fmt.Errorf("获取剧情线失败: %w", err)
	}
	return &plotline, nil
}

// Update 更新剧情线。
func (r *plotlineRepository) Update(plotline *model.Plotline) error {
	if plotline == nil {
		return fmt.Errorf("剧情线不能为空")
	}
	if plotline.ID == 0 {
		return fmt.Errorf("剧情线 ID 不能为空")
	}

	plotline.UpdatedAt = time.Now()
	result := r.db.Model(&model.Plotline{}).Where("id = ?", plotline.ID).Updates(plotline)
	if result.Error != nil {
		return fmt.Errorf("更新剧情线失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("剧情线不存在: %d", plotline.ID)
	}
	return nil
}

// Delete 软删除单条剧情线。
func (r *plotlineRepository) Delete(id uint) error {
	if id == 0 {
		return fmt.Errorf("剧情线 ID 不能为空")
	}

	result := r.db.Where("id = ?", id).Delete(&model.Plotline{})
	if result.Error != nil {
		return fmt.Errorf("删除剧情线失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("剧情线不存在: %d", id)
	}
	return nil
}

// DeleteByNovelID 删除指定小说的全部剧情线。
func (r *plotlineRepository) DeleteByNovelID(novelID uint) error {
	if novelID == 0 {
		return fmt.Errorf("小说 ID 不能为空")
	}

	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Plotline{})
	if result.Error != nil {
		return fmt.Errorf("删除剧情线失败: %w", result.Error)
	}
	return nil
}
