package repository

import (
	"fmt"
	"strings"
	"time"

	"waku/internal/model"

	"gorm.io/gorm"
)

// ContentRepository 定义正文工作流需要的章节正文数据访问能力。
type ContentRepository interface {
	GetChapterByID(id uint) (*model.Chapter, error)
	GetChaptersByVolumeID(volumeID uint) ([]*model.Chapter, error)
	GetEmptyContentChapters(volumeID uint) ([]*model.Chapter, error)
	UpdateContent(chapterID uint, content string, wordCount int, status string) error
	UpdateContentBatch(chapters []*model.Chapter) error
	GetPreviousChapter(chapterID uint) (*model.Chapter, error)
	GetNextChapter(chapterID uint) (*model.Chapter, error)
}

type contentRepository struct {
	db *gorm.DB
}

// NewContentRepository 创建正文仓储实例。
func NewContentRepository(db *gorm.DB) ContentRepository {
	return &contentRepository{db: db}
}

// GetChapterByID 根据章节 ID 获取正文工作流目标章节。
func (r *contentRepository) GetChapterByID(id uint) (*model.Chapter, error) {
	if id == 0 {
		return nil, fmt.Errorf("章节ID不能为空")
	}

	var chapter model.Chapter
	if err := r.db.Where("id = ?", id).First(&chapter).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("章节不存在: %d", id)
		}
		return nil, fmt.Errorf("获取章节失败: %w", err)
	}
	return &chapter, nil
}

// GetChaptersByVolumeID 按章节号升序获取分卷下全部章节，供批量正文生成选择目标。
func (r *contentRepository) GetChaptersByVolumeID(volumeID uint) ([]*model.Chapter, error) {
	if volumeID == 0 {
		return nil, fmt.Errorf("分卷ID不能为空")
	}

	var chapters []*model.Chapter
	if err := r.db.Where("volume_id = ?", volumeID).
		Order("chapter_number ASC, id ASC").
		Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("获取分卷章节失败: %w", err)
	}
	return chapters, nil
}

// GetEmptyContentChapters 获取当前分卷中正文为空的章节，用于低成本补齐正文。
func (r *contentRepository) GetEmptyContentChapters(volumeID uint) ([]*model.Chapter, error) {
	if volumeID == 0 {
		return nil, fmt.Errorf("分卷ID不能为空")
	}

	var chapters []*model.Chapter
	if err := r.db.Where("volume_id = ? AND (content = '' OR content IS NULL)", volumeID).
		Order("chapter_number ASC, id ASC").
		Find(&chapters).Error; err != nil {
		return nil, fmt.Errorf("获取空正文章节失败: %w", err)
	}
	return chapters, nil
}

// UpdateContent 只更新正文相关字段，避免正文工作流误改章节规划字段。
func (r *contentRepository) UpdateContent(chapterID uint, content string, wordCount int, status string) error {
	if chapterID == 0 {
		return fmt.Errorf("章节ID不能为空")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("正文不能为空")
	}
	if status == "" {
		status = "writing"
	}

	updates := map[string]any{
		"content":    content,
		"word_count": wordCount,
		"status":     status,
		"updated_at": time.Now(),
	}
	result := r.db.Model(&model.Chapter{}).Where("id = ?", chapterID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新章节正文失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("章节不存在: %d", chapterID)
	}
	return nil
}

// UpdateContentBatch 按顺序更新多章正文；任一章节失败时直接返回，避免批量结果半静默失败。
func (r *contentRepository) UpdateContentBatch(chapters []*model.Chapter) error {
	if len(chapters) == 0 {
		return fmt.Errorf("章节列表不能为空")
	}
	for _, chapter := range chapters {
		if chapter == nil {
			return fmt.Errorf("章节不能为空")
		}
		if err := r.UpdateContent(chapter.ID, chapter.Content, chapter.WordCount, chapter.Status); err != nil {
			return err
		}
	}
	return nil
}

// GetPreviousChapter 获取同一分卷内目标章节的上一章，用于构造正文承接上下文。
func (r *contentRepository) GetPreviousChapter(chapterID uint) (*model.Chapter, error) {
	target, err := r.GetChapterByID(chapterID)
	if err != nil {
		return nil, err
	}

	var chapter model.Chapter
	err = r.db.Where("volume_id = ? AND chapter_number < ?", target.VolumeID, target.ChapterNumber).
		Order("chapter_number DESC, id DESC").
		First(&chapter).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取上一章失败: %w", err)
	}
	return &chapter, nil
}

// GetNextChapter 获取同一分卷内目标章节的下一章，用于约束正文结尾方向。
func (r *contentRepository) GetNextChapter(chapterID uint) (*model.Chapter, error) {
	target, err := r.GetChapterByID(chapterID)
	if err != nil {
		return nil, err
	}

	var chapter model.Chapter
	err = r.db.Where("volume_id = ? AND chapter_number > ?", target.VolumeID, target.ChapterNumber).
		Order("chapter_number ASC, id ASC").
		First(&chapter).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("获取下一章失败: %w", err)
	}
	return &chapter, nil
}
