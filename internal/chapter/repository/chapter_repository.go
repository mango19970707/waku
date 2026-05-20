package repository

import (
	"fmt"
	"waku/internal/model"

	"gorm.io/gorm"
)

// ChapterRepository 章节数据仓储接口
type ChapterRepository interface {
	// CreateBatch 批量创建章节
	CreateBatch(chapters []*model.Chapter) error

	// GetByID 根据ID获取章节
	GetByID(id uint) (*model.Chapter, error)

	// GetByVolumeID 根据分卷ID获取所有章节
	GetByVolumeID(volumeID uint) ([]*model.Chapter, error)

	// GetByNovelID 根据小说ID获取所有章节
	GetByNovelID(novelID uint) ([]*model.Chapter, error)

	// Update 更新章节
	Update(chapter *model.Chapter) error

	// Delete 删除章节
	Delete(id uint) error

	// DeleteByVolumeID 删除分卷下的所有章节
	DeleteByVolumeID(volumeID uint) error

	// DeleteByNovelID 删除小说下的所有章节
	DeleteByNovelID(novelID uint) error

	// GetMaxChapterNumber 获取分卷下的最大章节号
	GetMaxChapterNumber(volumeID uint) (int, error)
}

// chapterRepository 章节数据仓储实现
type chapterRepository struct {
	db *gorm.DB
}

// NewChapterRepository 创建章节数据仓储实例
func NewChapterRepository(db *gorm.DB) ChapterRepository {
	return &chapterRepository{db: db}
}

// CreateBatch 批量创建章节
func (r *chapterRepository) CreateBatch(chapters []*model.Chapter) error {
	if len(chapters) == 0 {
		return nil
	}
	return r.db.Create(&chapters).Error
}

// GetByID 根据ID获取章节
func (r *chapterRepository) GetByID(id uint) (*model.Chapter, error) {
	var chapter model.Chapter
	err := r.db.First(&chapter, id).Error
	if err != nil {
		return nil, fmt.Errorf("获取章节失败: %w", err)
	}
	return &chapter, nil
}

// GetByVolumeID 根据分卷ID获取所有章节
func (r *chapterRepository) GetByVolumeID(volumeID uint) ([]*model.Chapter, error) {
	var chapters []*model.Chapter
	err := r.db.Where("volume_id = ?", volumeID).Order("chapter_number ASC").Find(&chapters).Error
	if err != nil {
		return nil, fmt.Errorf("获取分卷章节列表失败: %w", err)
	}
	return chapters, nil
}

// GetByNovelID 根据小说ID获取所有章节
func (r *chapterRepository) GetByNovelID(novelID uint) ([]*model.Chapter, error) {
	var chapters []*model.Chapter
	err := r.db.Where("novel_id = ?", novelID).Order("volume_id ASC, chapter_number ASC").Find(&chapters).Error
	if err != nil {
		return nil, fmt.Errorf("获取小说章节列表失败: %w", err)
	}
	return chapters, nil
}

// Update 更新章节
func (r *chapterRepository) Update(chapter *model.Chapter) error {
	return r.db.Save(chapter).Error
}

// Delete 删除章节
func (r *chapterRepository) Delete(id uint) error {
	return r.db.Delete(&model.Chapter{}, id).Error
}

// DeleteByVolumeID 删除分卷下的所有章节
func (r *chapterRepository) DeleteByVolumeID(volumeID uint) error {
	return r.db.Where("volume_id = ?", volumeID).Delete(&model.Chapter{}).Error
}

// DeleteByNovelID 删除小说下的所有章节
func (r *chapterRepository) DeleteByNovelID(novelID uint) error {
	return r.db.Where("novel_id = ?", novelID).Delete(&model.Chapter{}).Error
}

// GetMaxChapterNumber 获取分卷下的最大章节号
func (r *chapterRepository) GetMaxChapterNumber(volumeID uint) (int, error) {
	var maxNumber int
	err := r.db.Model(&model.Chapter{}).
		Where("volume_id = ?", volumeID).
		Select("COALESCE(MAX(chapter_number), 0)").
		Scan(&maxNumber).Error
	if err != nil {
		return 0, fmt.Errorf("获取最大章节号失败: %w", err)
	}
	return maxNumber, nil
}
