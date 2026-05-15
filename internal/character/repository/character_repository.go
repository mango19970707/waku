package repository

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"waku/internal/model"
)

// CharacterRepository 人物数据仓储接口
type CharacterRepository interface {
	// CreateBatch 批量创建人物
	CreateBatch(characters []*model.Character) error

	// GetByNovelID 根据小说ID获取所有人物
	GetByNovelID(novelID string) ([]*model.Character, error)

	// GetByID 根据ID获取人物
	GetByID(id string) (*model.Character, error)

	// Update 更新人物信息
	Update(character *model.Character) error

	// Delete 软删除人物
	Delete(id string) error

	// DeleteByNovelID 软删除指定小说的所有人物
	DeleteByNovelID(novelID string) error
}

// characterRepository 人物数据仓储实现
type characterRepository struct {
	db *gorm.DB
}

// NewCharacterRepository 创建人物数据仓储实例
//
// 参数：
//   - db: 数据库连接
//
// 返回：
//   - CharacterRepository: 仓储接口实例
func NewCharacterRepository(db *gorm.DB) CharacterRepository {
	return &characterRepository{db: db}
}

// CreateBatch 批量创建人物
//
// 参数：
//   - characters: 人物列表
//
// 返回：
//   - error: 错误信息
func (r *characterRepository) CreateBatch(characters []*model.Character) error {
	if len(characters) == 0 {
		return fmt.Errorf("人物列表为空")
	}

	// 为每个人物生成 UUID 和时间戳
	now := time.Now()
	for _, char := range characters {
		if char.ID == "" {
			char.ID = uuid.New().String()
		}
		char.CreatedAt = now
		char.UpdatedAt = now
	}

	// 批量插入
	if err := r.db.Create(characters).Error; err != nil {
		return fmt.Errorf("批量创建人物失败: %w", err)
	}

	return nil
}

// GetByNovelID 根据小说ID获取所有人物
//
// 参数：
//   - novelID: 小说ID
//
// 返回：
//   - []*model.Character: 人物列表
//   - error: 错误信息
func (r *characterRepository) GetByNovelID(novelID string) ([]*model.Character, error) {
	var characters []*model.Character

	// 查询人物，按角色定位和姓名排序
	// 排序规则：主角 > 配角 > 反派 > 导师 > 感情线角色
	err := r.db.Where("novel_id = ?", novelID).
		Order("CASE role WHEN 'protagonist' THEN 1 WHEN 'supporting' THEN 2 WHEN 'antagonist' THEN 3 WHEN 'mentor' THEN 4 WHEN 'love_interest' THEN 5 ELSE 6 END").
		Order("name ASC").
		Find(&characters).Error

	if err != nil {
		return nil, fmt.Errorf("查询人物失败: %w", err)
	}

	return characters, nil
}

// GetByID 根据ID获取人物
//
// 参数：
//   - id: 人物ID
//
// 返回：
//   - *model.Character: 人物信息
//   - error: 错误信息
func (r *characterRepository) GetByID(id string) (*model.Character, error) {
	var character model.Character

	err := r.db.Where("id = ?", id).First(&character).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("人物不存在: %s", id)
		}
		return nil, fmt.Errorf("查询人物失败: %w", err)
	}

	return &character, nil
}

// Update 更新人物信息
//
// 参数：
//   - character: 人物信息
//
// 返回：
//   - error: 错误信息
func (r *characterRepository) Update(character *model.Character) error {
	if character.ID == "" {
		return fmt.Errorf("人物ID为空")
	}

	// 更新时间戳
	character.UpdatedAt = time.Now()

	// 更新人物信息
	result := r.db.Model(&model.Character{}).
		Where("id = ?", character.ID).
		Updates(character)

	if result.Error != nil {
		return fmt.Errorf("更新人物失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("人物不存在: %s", character.ID)
	}

	return nil
}

// Delete 软删除人物
//
// 参数：
//   - id: 人物ID
//
// 返回：
//   - error: 错误信息
func (r *characterRepository) Delete(id string) error {
	result := r.db.Delete(&model.Character{}, "id = ?", id)

	if result.Error != nil {
		return fmt.Errorf("删除人物失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("人物不存在: %s", id)
	}

	return nil
}

// DeleteByNovelID 软删除指定小说的所有人物
//
// 参数：
//   - novelID: 小说ID
//
// 返回：
//   - error: 错误信息
func (r *characterRepository) DeleteByNovelID(novelID string) error {
	result := r.db.Where("novel_id = ?", novelID).Delete(&model.Character{})

	if result.Error != nil {
		return fmt.Errorf("删除小说人物失败: %w", result.Error)
	}

	return nil
}
