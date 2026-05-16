package repository

import (
	"fmt"
	"time"
	"waku/internal/model"

	"gorm.io/gorm"
)

type CharacterRepository interface {
	CreateBatch(characters []*model.Character) error
	GetByNovelID(novelID uint) ([]*model.Character, error)
	GetByID(id uint) (*model.Character, error)
	Update(character *model.Character) error
	Delete(id uint) error
	DeleteByNovelID(novelID uint) error
}

type characterRepository struct {
	db *gorm.DB
}

func NewCharacterRepository(db *gorm.DB) CharacterRepository {
	return &characterRepository{db: db}
}

func (r *characterRepository) CreateBatch(characters []*model.Character) error {
	if len(characters) == 0 {
		return fmt.Errorf("角色列表不能为空")
	}

	now := time.Now()
	for _, character := range characters {
		if character.NovelID == 0 {
			return fmt.Errorf("角色所属小说ID不能为空")
		}
		character.CreatedAt = now
		character.UpdatedAt = now
	}

	if err := r.db.Create(characters).Error; err != nil {
		return fmt.Errorf("批量创建角色失败: %w", err)
	}
	return nil
}

func (r *characterRepository) GetByNovelID(novelID uint) ([]*model.Character, error) {
	var characters []*model.Character
	if err := r.db.Where("novel_id = ?", novelID).
		Order("created_at ASC, id ASC").
		Find(&characters).Error; err != nil {
		return nil, fmt.Errorf("获取角色列表失败: %w", err)
	}
	return characters, nil
}

func (r *characterRepository) GetByID(id uint) (*model.Character, error) {
	if id == 0 {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	var character model.Character
	if err := r.db.Where("id = ?", id).First(&character).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("角色不存在: %d", id)
		}
		return nil, fmt.Errorf("获取角色失败: %w", err)
	}
	return &character, nil
}

func (r *characterRepository) Update(character *model.Character) error {
	if character == nil {
		return fmt.Errorf("角色不能为空")
	}
	if character.ID == 0 {
		return fmt.Errorf("角色ID不能为空")
	}

	character.UpdatedAt = time.Now()
	result := r.db.Model(&model.Character{}).Where("id = ?", character.ID).Updates(character)
	if result.Error != nil {
		return fmt.Errorf("更新角色失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("角色不存在: %d", character.ID)
	}
	return nil
}

func (r *characterRepository) Delete(id uint) error {
	if id == 0 {
		return fmt.Errorf("角色ID不能为空")
	}

	result := r.db.Where("id = ?", id).Delete(&model.Character{})
	if result.Error != nil {
		return fmt.Errorf("删除角色失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("角色不存在: %d", id)
	}
	return nil
}

func (r *characterRepository) DeleteByNovelID(novelID uint) error {
	if novelID == 0 {
		return fmt.Errorf("小说ID不能为空")
	}

	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Character{})
	if result.Error != nil {
		return fmt.Errorf("删除小说角色失败: %w", result.Error)
	}
	return nil
}
