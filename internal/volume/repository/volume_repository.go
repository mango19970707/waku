package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"gorm.io/gorm"
)

// VolumeRepository Volume 数据仓库接口
type VolumeRepository interface {
	// Create 创建卷
	Create(volume *model.Volume) error

	// CreateBatch 批量创建卷
	CreateBatch(volumes []*model.Volume) error

	// GetByID 根据 ID 获取卷
	GetByID(id uint) (*model.Volume, error)

	// GetByNovelID 根据小说 ID 获取所有卷
	GetByNovelID(novelID uint) ([]*model.Volume, error)

	// GetByNovelIDWithChapters 根据小说 ID 获取所有卷（包含章节）
	GetByNovelIDWithChapters(novelID uint) ([]*model.Volume, error)

	// Update 更新卷信息
	Update(volume *model.Volume) error

	// UpdateStatus 更新卷状态
	UpdateStatus(id uint, status string) error

	// Delete 删除卷（软删除）
	Delete(id uint) error

	// DeleteByNovelID 删除小说的所有卷（软删除）
	DeleteByNovelID(novelID uint) error

	// Count 统计卷总数
	Count() (int64, error)

	// CountByNovelID 统计指定小说的卷数量
	CountByNovelID(novelID uint) (int64, error)
}

// volumeRepository Volume 数据仓库实现
type volumeRepository struct {
	db *gorm.DB
}

// NewVolumeRepository 创建 Volume 数据仓库实例
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - VolumeRepository: Volume 数据仓库接口
func NewVolumeRepository(db *gorm.DB) VolumeRepository {
	return &volumeRepository{db: db}
}

// Create 创建卷
//
// 参数：
//   - volume: 卷对象
//
// 返回值：
//   - error: 创建失败时返回错误
func (r *volumeRepository) Create(volume *model.Volume) error {
	// 设置创建时间和更新时间
	now := time.Now()
	volume.CreatedAt = now
	volume.UpdatedAt = now

	// 设置默认状态
	if volume.Status == "" {
		volume.Status = "draft"
	}

	// 执行创建操作
	if err := r.db.Create(volume).Error; err != nil {
		return fmt.Errorf("创建卷失败: %w", err)
	}

	return nil
}

// CreateBatch 批量创建卷
//
// 参数：
//   - volumes: 卷对象列表
//
// 返回值：
//   - error: 创建失败时返回错误
func (r *volumeRepository) CreateBatch(volumes []*model.Volume) error {
	if len(volumes) == 0 {
		return nil
	}

	// 设置创建时间和更新时间
	now := time.Now()
	for _, volume := range volumes {
		volume.CreatedAt = now
		volume.UpdatedAt = now

		// 设置默认状态
		if volume.Status == "" {
			volume.Status = "draft"
		}
	}

	// 批量插入
	if err := r.db.Create(volumes).Error; err != nil {
		return fmt.Errorf("批量创建卷失败: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取卷
//
// 参数：
//   - id: 卷 ID
//
// 返回值：
//   - *model.Volume: 卷对象
//   - error: 获取失败时返回错误
func (r *volumeRepository) GetByID(id uint) (*model.Volume, error) {
	var volume model.Volume

	if err := r.db.Where("id = ?", id).First(&volume).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("卷不存在: %d", id)
		}
		return nil, fmt.Errorf("获取卷失败: %w", err)
	}

	return &volume, nil
}

// GetByNovelID 根据小说 ID 获取所有卷
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - []*model.Volume: 卷列表
//   - error: 获取失败时返回错误
func (r *volumeRepository) GetByNovelID(novelID uint) ([]*model.Volume, error) {
	var volumes []*model.Volume

	if err := r.db.Where("novel_id = ?", novelID).
		Order("volume_number ASC").
		Find(&volumes).Error; err != nil {
		return nil, fmt.Errorf("获取卷列表失败: %w", err)
	}

	return volumes, nil
}

// GetByNovelIDWithChapters 根据小说 ID 获取所有卷（包含章节）
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - []*model.Volume: 卷列表（包含章节）
//   - error: 获取失败时返回错误
func (r *volumeRepository) GetByNovelIDWithChapters(novelID uint) ([]*model.Volume, error) {
	var volumes []*model.Volume

	if err := r.db.Where("novel_id = ?", novelID).
		Preload("Chapters", func(db *gorm.DB) *gorm.DB {
			return db.Order("chapter_number ASC")
		}).
		Order("volume_number ASC").
		Find(&volumes).Error; err != nil {
		return nil, fmt.Errorf("获取卷列表失败: %w", err)
	}

	return volumes, nil
}

// Update 更新卷信息
//
// 参数：
//   - volume: 卷对象（必须包含 ID）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *volumeRepository) Update(volume *model.Volume) error {
	if volume.ID == 0 {
		return fmt.Errorf("卷 ID 不能为空")
	}

	// 更新时间
	volume.UpdatedAt = time.Now()

	// 执行更新操作（只更新非零值字段）
	result := r.db.Model(&model.Volume{}).Where("id = ?", volume.ID).Updates(volume)
	if result.Error != nil {
		return fmt.Errorf("更新卷失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("卷不存在: %d", volume.ID)
	}

	return nil
}

// UpdateStatus 更新卷状态
//
// 参数：
//   - id: 卷 ID
//   - status: 新状态（draft、writing、completed）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *volumeRepository) UpdateStatus(id uint, status string) error {
	result := r.db.Model(&model.Volume{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("更新卷状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("卷不存在: %d", id)
	}

	return nil
}

// Delete 删除卷（软删除）
//
// 参数：
//   - id: 卷 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *volumeRepository) Delete(id uint) error {
	result := r.db.Where("id = ?", id).Delete(&model.Volume{})

	if result.Error != nil {
		return fmt.Errorf("删除卷失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("卷不存在: %d", id)
	}

	return nil
}

// DeleteByNovelID 删除小说的所有卷（软删除）
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *volumeRepository) DeleteByNovelID(novelID uint) error {
	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Volume{})

	if result.Error != nil {
		return fmt.Errorf("删除卷失败: %w", result.Error)
	}

	return nil
}

// Count 统计卷总数
//
// 返回值：
//   - int64: 卷总数
//   - error: 统计失败时返回错误
func (r *volumeRepository) Count() (int64, error) {
	var count int64

	if err := r.db.Model(&model.Volume{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计卷总数失败: %w", err)
	}

	return count, nil
}

// CountByNovelID 统计指定小说的卷数量
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - int64: 卷数量
//   - error: 统计失败时返回错误
func (r *volumeRepository) CountByNovelID(novelID uint) (int64, error) {
	var count int64

	if err := r.db.Model(&model.Volume{}).Where("novel_id = ?", novelID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计卷数量失败: %w", err)
	}

	return count, nil
}
