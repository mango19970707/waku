package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"gorm.io/gorm"
)

// ChapterRepository Chapter 数据仓库接口
type ChapterRepository interface {
	// Create 创建章节
	Create(chapter *model.Chapter) error

	// CreateBatch 批量创建章节
	CreateBatch(chapters []*model.Chapter) error

	// GetByID 根据 ID 获取章节
	GetByID(id uint) (*model.Chapter, error)

	// GetByVolumeID 根据卷 ID 获取所有章节
	GetByVolumeID(volumeID uint) ([]*model.Chapter, error)

	// GetByNovelID 根据小说 ID 获取所有章节
	GetByNovelID(novelID uint) ([]*model.Chapter, error)

	// Update 更新章节信息
	Update(chapter *model.Chapter) error

	// UpdateStatus 更新章节状态
	UpdateStatus(id uint, status string) error

	// UpdateContent 更新章节内容
	UpdateContent(id uint, content string, wordCount int) error

	// Delete 删除章节（软删除）
	Delete(id uint) error

	// DeleteByVolumeID 删除卷的所有章节（软删除）
	DeleteByVolumeID(volumeID uint) error

	// DeleteByNovelID 删除小说的所有章节（软删除）
	DeleteByNovelID(novelID uint) error

	// Count 统计章节总数
	Count() (int64, error)

	// CountByVolumeID 统计指定卷的章节数量
	CountByVolumeID(volumeID uint) (int64, error)

	// CountByNovelID 统计指定小说的章节数量
	CountByNovelID(novelID uint) (int64, error)
}

// chapterRepository Chapter 数据仓库实现
type chapterRepository struct {
	db *gorm.DB
}

// NewChapterRepository 创建 Chapter 数据仓库实例
//
// 参数：
//   - db: GORM 数据库实例
//
// 返回值：
//   - ChapterRepository: Chapter 数据仓库接口
func NewChapterRepository(db *gorm.DB) ChapterRepository {
	return &chapterRepository{db: db}
}

// Create 创建章节
//
// 参数：
//   - chapter: 章节对象
//
// 返回值：
//   - error: 创建失败时返回错误
func (r *chapterRepository) Create(chapter *model.Chapter) error {
	// 设置创建时间和更新时间
	now := time.Now()
	chapter.CreatedAt = now
	chapter.UpdatedAt = now

	// 设置默认状态
	if chapter.Status == "" {
		chapter.Status = "draft"
	}

	// 执行创建操作
	if err := r.db.Create(chapter).Error; err != nil {
		return fmt.Errorf("创建章节失败: %w", err)
	}

	return nil
}

// CreateBatch 批量创建章节
//
// 参数：
//   - chapters: 章节对象列表
//
// 返回值：
//   - error: 创建失败时返回错误
func (r *chapterRepository) CreateBatch(chapters []*model.Chapter) error {
	if len(chapters) == 0 {
		return nil
	}

	// 设置创建时间和更新时间
	now := time.Now()
	for _, chapter := range chapters {
		chapter.CreatedAt = now
		chapter.UpdatedAt = now

		// 设置默认状态
		if chapter.Status == "" {
			chapter.Status = "draft"
		}
	}

	// 批量插入
	if err := r.db.Create(chapters).Error; err != nil {
		return fmt.Errorf("批量创建章节失败: %w", err)
	}

	return nil
}

// GetByID 根据 ID 获取章节
//
// 参数：
//   - id: 章节 ID
//
// 返回值：
//   - *model.Chapter: 章节对象
//   - error: 获取失败时返回错误
func (r *chapterRepository) GetByID(id uint) (*model.Chapter, error) {
	var chapter model.Chapter

	if err := r.db.Where("id = ?", id).First(&chapter).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("章节不存在: %d", id)
		}
		return nil, fmt.Errorf("获取章节失败: %w", err)
	}

	return &chapter, nil
}

// GetByVolumeID 根据卷 ID 获取所有章节
//
// 参数：
//   - volumeID: 卷 ID
//
// 返回值：
//   - []*model.Chapter: 章节列表
//   - error: 获取失败时返回错误
func (r *chapterRepository) GetByVolumeID(volumeID uint) ([]*model.Chapter, error) {
	var chapters []*model.Chapter

	if err := r.db.Where("volume_id = ?", volumeID).
		Order("chapter_number ASC").
		Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("获取章节列表失败: %w", err)
	}

	return chapters, nil
}

// GetByNovelID 根据小说 ID 获取所有章节
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - []*model.Chapter: 章节列表
//   - error: 获取失败时返回错误
func (r *chapterRepository) GetByNovelID(novelID uint) ([]*model.Chapter, error) {
	var chapters []*model.Chapter

	if err := r.db.Where("novel_id = ?", novelID).
		Order("volume_id ASC, chapter_number ASC").
		Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("获取章节列表失败: %w", err)
	}

	return chapters, nil
}

// Update 更新章节信息
//
// 参数：
//   - chapter: 章节对象（必须包含 ID）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *chapterRepository) Update(chapter *model.Chapter) error {
	if chapter.ID == 0 {
		return fmt.Errorf("章节 ID 不能为空")
	}

	// 更新时间
	chapter.UpdatedAt = time.Now()

	// 执行更新操作（只更新非零值字段）
	result := r.db.Model(&model.Chapter{}).Where("id = ?", chapter.ID).Updates(chapter)
	if result.Error != nil {
		return fmt.Errorf("更新章节失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("章节不存在: %d", chapter.ID)
	}

	return nil
}

// UpdateStatus 更新章节状态
//
// 参数：
//   - id: 章节 ID
//   - status: 新状态（draft、writing、completed）
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *chapterRepository) UpdateStatus(id uint, status string) error {
	result := r.db.Model(&model.Chapter{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("更新章节状态失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("章节不存在: %d", id)
	}

	return nil
}

// UpdateContent 更新章节内容
//
// 参数：
//   - id: 章节 ID
//   - content: 章节内容
//   - wordCount: 字数统计
//
// 返回值：
//   - error: 更新失败时返回错误
func (r *chapterRepository) UpdateContent(id uint, content string, wordCount int) error {
	result := r.db.Model(&model.Chapter{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"content":    content,
			"word_count": wordCount,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("更新章节内容失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("章节不存在: %d", id)
	}

	return nil
}

// Delete 删除章节（软删除）
//
// 参数：
//   - id: 章节 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *chapterRepository) Delete(id uint) error {
	result := r.db.Where("id = ?", id).Delete(&model.Chapter{})

	if result.Error != nil {
		return fmt.Errorf("删除章节失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("章节不存在: %d", id)
	}

	return nil
}

// DeleteByVolumeID 删除卷的所有章节（软删除）
//
// 参数：
//   - volumeID: 卷 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *chapterRepository) DeleteByVolumeID(volumeID uint) error {
	result := r.db.Where("volume_id = ?", volumeID).Delete(&model.Chapter{})

	if result.Error != nil {
		return fmt.Errorf("删除章节失败: %w", result.Error)
	}

	return nil
}

// DeleteByNovelID 删除小说的所有章节（软删除）
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - error: 删除失败时返回错误
func (r *chapterRepository) DeleteByNovelID(novelID uint) error {
	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Chapter{})

	if result.Error != nil {
		return fmt.Errorf("删除章节失败: %w", result.Error)
	}

	return nil
}

// Count 统计章节总数
//
// 返回值：
//   - int64: 章节总数
//   - error: 统计失败时返回错误
func (r *chapterRepository) Count() (int64, error) {
	var count int64

	if err := r.db.Model(&model.Chapter{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计章节总数失败: %w", err)
	}

	return count, nil
}

// CountByVolumeID 统计指定卷的章节数量
//
// 参数：
//   - volumeID: 卷 ID
//
// 返回值：
//   - int64: 章节数量
//   - error: 统计失败时返回错误
func (r *chapterRepository) CountByVolumeID(volumeID uint) (int64, error) {
	var count int64

	if err := r.db.Model(&model.Chapter{}).Where("volume_id = ?", volumeID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计章节数量失败: %w", err)
	}

	return count, nil
}

// CountByNovelID 统计指定小说的章节数量
//
// 参数：
//   - novelID: 小说 ID
//
// 返回值：
//   - int64: 章节数量
//   - error: 统计失败时返回错误
func (r *chapterRepository) CountByNovelID(novelID uint) (int64, error) {
	var count int64

	if err := r.db.Model(&model.Chapter{}).Where("novel_id = ?", novelID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计章节数量失败: %w", err)
	}

	return count, nil
}
