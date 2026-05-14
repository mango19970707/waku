package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlotlineRepository 剧情线数据仓库接口
type PlotlineRepository interface {
	// CreateBatch 批量创建剧情线
	CreateBatch(plotlines []*model.Plotline) error

	// GetByNovelID 根据小说 ID 获取所有剧情线
	GetByNovelID(novelID string) ([]*model.Plotline, error)

	// GetByID 根据 ID 获取剧情线
	GetByID(id string) (*model.Plotline, error)

	// Update 更新剧情线
	Update(plotline *model.Plotline) error

	// Delete 删除剧情线（软删除）
	Delete(id string) error

	// DeleteByNovelID 删除小说的所有剧情线（软删除）
	DeleteByNovelID(novelID string) error
}

// plotlineRepository 剧情线数据仓库实现
type plotlineRepository struct {
	db *gorm.DB
}

// NewPlotlineRepository 创建剧情线数据仓库实例
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - PlotlineRepository: 剧情线数据仓库接口
func NewPlotlineRepository(db *gorm.DB) PlotlineRepository {
	return &plotlineRepository{db: db}
}

// CreateBatch 批量创建剧情线
//
// 参数：
//   - plotlines: 剧情线对象列表
//
// 返回值：
//   - error: 创建失败时返回错误
func (r *plotlineRepository) CreateBatch(plotlines []*model.Plotline) error {
	if len(plotlines) == 0 {
		return fmt.Errorf("剧情线列表不能为空")
	}

	// 设置创建时间和更新时间，生成 UUID
	now := time.Now()
	for _, plotline := range plotlines {
		if plotline.ID == "" {
			plotline.ID = uuid.New().String()
		}
		plotline.CreatedAt = now
		plotline.UpdatedAt = now

		// 设置默认状态
		if plotline.Status == "" {
			plotline.Status = "active"
		}
	}

	// 批量插入
	if err := r.db.Create(plotlines).Error; err != nil {
		return fmt.Errorf("批量创建剧情线失败: %w", err)
	}

	return nil
}

// GetByNovelID 根据小说 ID 获取所有剧情线
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - []*model.Plotline: 剧情线列表
//   - error: 获取失败时返回错误
func (r *plotlineRepository) GetByNovelID(novelID string) ([]*model.Plotline, error) {
	var plotlines []*model.Plotline

	if err := r.db.Where("novel_id = ?", novelID).
		Order("type DESC, start_chapter ASC").
		Find(&plotlines).Error; err != nil {
		return nil, fmt.Errorf("获取剧情线列表失败: %w", err)
	}

	return plotlines, nil
}

// GetByID 根据 ID 获取剧情线
//
// 参数：
//   - id: 剧情线 ID
//
// 返回值：
//   - *model.Plotline: 剧情线对象
//   - error: 获取失败时返回错误
func (r *plotlineRepository) GetByID(id string) (*model.Plotline, error) {
	var plotline model.Plotline

	if err := r.db.Where("id = ?", id).First(&plotline).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("剧情线不存在: %s", id)
		}
		return nil, fmt.Errorf("获取剧情线失败: %w", err)
	}

	return &plotline, nil
}

// Update 更新剧情线
//
// 参数：
//   - plotline: 剧情线对象（必须包含 ID）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *plotlineRepository) Update(plotline *model.Plotline) error {
	if plotline.ID == "" {
		return fmt.Errorf("剧情线 ID 不能为空")
	}

	// 更新时间
	plotline.UpdatedAt = time.Now()

	// 执行更新操作
	result := r.db.Model(&model.Plotline{}).Where("id = ?", plotline.ID).Updates(plotline)
	if result.Error != nil {
		return fmt.Errorf("更新剧情线失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("剧情线不存在: %s", plotline.ID)
	}

	return nil
}

// Delete 删除剧情线（软删除）
//
// 参数：
//   - id: 剧情线 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *plotlineRepository) Delete(id string) error {
	result := r.db.Where("id = ?", id).Delete(&model.Plotline{})

	if result.Error != nil {
		return fmt.Errorf("删除剧情线失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("剧情线不存在: %s", id)
	}

	return nil
}

// DeleteByNovelID 删除小说的所有剧情线（软删除）
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *plotlineRepository) DeleteByNovelID(novelID string) error {
	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Plotline{})

	if result.Error != nil {
		return fmt.Errorf("删除剧情线失败: %w", result.Error)
	}

	return nil
}
